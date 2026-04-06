package edge

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/imcitius/checker/pkg/models"
	"github.com/imcitius/checker/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

const (
	defaultEdgeWorkers = 10
)

// CheckResult holds the outcome of a single check execution.
type CheckResult struct {
	CheckUUID string
	IsHealthy bool
	Message   string
	Duration  time.Duration
	Timestamp time.Time
}

// edgeCheckItem is an item in the edge scheduler's priority queue.
type edgeCheckItem struct {
	CheckDef models.CheckDefinition
	NextRun  time.Time
	Index    int
}

// edgeCheckHeap is a min-heap of edgeCheckItems ordered by NextRun.
type edgeCheckHeap []*edgeCheckItem

func (h edgeCheckHeap) Len() int            { return len(h) }
func (h edgeCheckHeap) Less(i, j int) bool  { return h[i].NextRun.Before(h[j].NextRun) }
func (h edgeCheckHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}
func (h *edgeCheckHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*edgeCheckItem)
	item.Index = n
	*h = append(*h, item)
}
func (h *edgeCheckHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[0 : n-1]
	return item
}
func (h *edgeCheckHeap) peek() *edgeCheckItem {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

// EdgeScheduler is a lightweight in-memory scheduler for the edge checker.
// It uses no DB and produces no alerts — it just executes checks and reports
// results via a callback channel.
type EdgeScheduler struct {
	mu       sync.Mutex
	heap     *edgeCheckHeap
	checkMap map[string]*edgeCheckItem // UUID -> item

	workers    int
	jobs       chan models.CheckDefinition
	results    chan<- CheckResult
	workerWg   sync.WaitGroup
	stopWorkers chan struct{}

	// notify the scheduling loop that the heap changed
	changed chan struct{}
}

// NewEdgeScheduler creates a new EdgeScheduler. results is the channel that
// completed check results are written to; the caller is responsible for
// draining it.
func NewEdgeScheduler(workers int, results chan<- CheckResult) *EdgeScheduler {
	if workers <= 0 {
		workers = defaultEdgeWorkers
	}
	h := &edgeCheckHeap{}
	heap.Init(h)
	return &EdgeScheduler{
		heap:        h,
		checkMap:    make(map[string]*edgeCheckItem),
		workers:     workers,
		results:     results,
		jobs:        make(chan models.CheckDefinition, workers*2),
		stopWorkers: make(chan struct{}),
		changed:     make(chan struct{}, 1),
	}
}

// Run starts the scheduler and blocks until ctx is cancelled.
func (s *EdgeScheduler) Run(ctx context.Context) {
	// Start worker goroutines.
	for i := 0; i < s.workers; i++ {
		s.workerWg.Add(1)
		go s.worker(ctx, i)
	}

	s.loop(ctx)

	// Drain: close jobs so workers finish in-flight checks.
	close(s.jobs)
	s.workerWg.Wait()
	logrus.Info("EdgeScheduler: all workers stopped")
}

// loop is the main scheduling loop.
func (s *EdgeScheduler) loop(ctx context.Context) {
	for {
		s.mu.Lock()
		top := s.heap.peek()
		s.mu.Unlock()

		var timer *time.Timer
		if top == nil {
			// No checks — wait for a change signal or cancellation.
			timer = time.NewTimer(time.Hour)
		} else {
			d := time.Until(top.NextRun)
			if d < 0 {
				d = 0
			}
			timer = time.NewTimer(d)
		}

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.changed:
			timer.Stop()
			// Rebuild timer on next iteration.
			continue
		case <-timer.C:
			s.dispatchDue()
		}
	}
}

// dispatchDue pops all due checks from the heap and submits them to the worker pool.
func (s *EdgeScheduler) dispatchDue() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.heap.peek() != nil && !s.heap.peek().NextRun.After(now) {
		item := heap.Pop(s.heap).(*edgeCheckItem)

		// Reschedule.
		d := parseDuration(item.CheckDef.Duration)
		if d <= 0 {
			d = time.Minute
		}
		item.NextRun = now.Add(d)
		heap.Push(s.heap, item)
		// Also update checkMap reference (same pointer, heap updated it).

		// Submit to worker pool (non-blocking; if full, skip this cycle).
		def := item.CheckDef
		select {
		case s.jobs <- def:
		default:
			logrus.Warnf("EdgeScheduler: worker pool full, skipping check %s", def.UUID)
		}
	}
}

// worker pulls jobs from s.jobs and executes them.
func (s *EdgeScheduler) worker(ctx context.Context, id int) {
	defer s.workerWg.Done()
	log := logrus.WithField("edge_worker", id)
	for {
		select {
		case def, ok := <-s.jobs:
			if !ok {
				return
			}
			log.Debugf("executing check %s (%s)", def.UUID, def.Type)
			result := s.executeCheck(def)
			select {
			case s.results <- result:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// executeCheck runs a single check and returns the result.
func (s *EdgeScheduler) executeCheck(def models.CheckDefinition) CheckResult {
	log := logrus.WithFields(logrus.Fields{
		"check_uuid": def.UUID,
		"check_type": def.Type,
	})
	checker := scheduler.CheckerFactory(def, log)
	if checker == nil {
		return CheckResult{
			CheckUUID: def.UUID,
			IsHealthy: false,
			Message:   "unknown check type: " + def.Type,
			Timestamp: time.Now(),
		}
	}

	start := time.Now()
	dur, err := checker.Run()
	elapsed := time.Since(start)
	if dur > 0 {
		elapsed = dur
	}

	healthy := err == nil
	msg := ""
	if err != nil {
		msg = err.Error()
	}

	return CheckResult{
		CheckUUID: def.UUID,
		IsHealthy: healthy,
		Message:   msg,
		Duration:  elapsed,
		Timestamp: time.Now(),
	}
}

// ReplaceAll replaces the entire check set (used on config_sync).
func (s *EdgeScheduler) ReplaceAll(defs []models.CheckDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Rebuild heap and map from scratch.
	*s.heap = (*s.heap)[:0]
	s.checkMap = make(map[string]*edgeCheckItem, len(defs))

	now := time.Now()
	for _, def := range defs {
		if !def.Enabled {
			continue
		}
		d := parseDuration(def.Duration)
		if d <= 0 {
			d = time.Minute
		}
		item := &edgeCheckItem{
			CheckDef: def,
			NextRun:  now.Add(d),
		}
		heap.Push(s.heap, item)
		s.checkMap[def.UUID] = item
	}

	heap.Init(s.heap)
	s.notifyChanged()
}

// AddOrUpdate adds or updates a single check definition.
func (s *EdgeScheduler) AddOrUpdate(def models.CheckDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.checkMap[def.UUID]; ok {
		// Update in place and fix heap invariant.
		existing.CheckDef = def
		if !def.Enabled {
			// Remove from heap.
			heap.Remove(s.heap, existing.Index)
			delete(s.checkMap, def.UUID)
		} else {
			heap.Fix(s.heap, existing.Index)
		}
	} else if def.Enabled {
		d := parseDuration(def.Duration)
		if d <= 0 {
			d = time.Minute
		}
		item := &edgeCheckItem{
			CheckDef: def,
			NextRun:  time.Now().Add(d),
		}
		heap.Push(s.heap, item)
		s.checkMap[def.UUID] = item
	}

	s.notifyChanged()
}

// Delete removes a check by UUID.
func (s *EdgeScheduler) Delete(uuid string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item, ok := s.checkMap[uuid]; ok {
		heap.Remove(s.heap, item.Index)
		delete(s.checkMap, uuid)
		s.notifyChanged()
	}
}

// ActiveCount returns the number of currently scheduled checks.
func (s *EdgeScheduler) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.checkMap)
}

// notifyChanged sends a non-blocking signal to the scheduling loop.
// Must be called with s.mu held.
func (s *EdgeScheduler) notifyChanged() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// parseDuration parses a duration string (e.g. "30s", "5m").
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

// viewModelToCheckDef converts a flat CheckDefinitionViewModel (as received
// over the WebSocket wire) to a CheckDefinition with the proper polymorphic
// Config field populated.
func viewModelToCheckDef(vm models.CheckDefinitionViewModel) models.CheckDefinition {
	def := models.CheckDefinition{
		UUID:                 vm.UUID,
		Name:                 vm.Name,
		Project:              vm.Project,
		GroupName:            vm.GroupName,
		Type:                 vm.Type,
		Description:          vm.Description,
		Enabled:              vm.Enabled,
		Duration:             vm.Duration,
		ActorType:            vm.ActorType,
		Severity:             vm.Severity,
		AlertChannels:        vm.AlertChannels,
		ReAlertInterval:      vm.ReAlertInterval,
		RetryCount:           vm.RetryCount,
		RetryInterval:        vm.RetryInterval,
		EscalationPolicyName: vm.EscalationPolicyName,
		RunMode:              vm.RunMode,
		TargetRegions:        vm.TargetRegions,
	}

	// Parse MaintenanceUntil string pointer to *time.Time.
	if vm.MaintenanceUntil != nil && *vm.MaintenanceUntil != "" {
		if t, err := time.Parse(time.RFC3339, *vm.MaintenanceUntil); err == nil {
			def.MaintenanceUntil = &t
		}
	}

	// Populate polymorphic Config based on Type.
	switch vm.Type {
	case "http":
		def.Config = &models.HTTPCheckConfig{
			URL:                 vm.URL,
			Timeout:             vm.Timeout,
			Answer:              vm.Answer,
			AnswerPresent:       vm.AnswerPresent,
			Code:                vm.Code,
			Headers:             vm.Headers,
			Cookies:             vm.Cookies,
			SkipCheckSSL:        vm.SkipCheckSSL,
			SSLExpirationPeriod: vm.SSLExpirationPeriod,
			StopFollowRedirects: vm.StopFollowRedirects,
			Auth: models.AuthConfig{
				User:     vm.Auth.User,
				Password: vm.Auth.Password,
			},
		}
	case "tcp":
		def.Config = &models.TCPCheckConfig{
			Host:    vm.Host,
			Port:    vm.Port,
			Timeout: vm.Timeout,
		}
	case "icmp":
		def.Config = &models.ICMPCheckConfig{
			Host:    vm.Host,
			Count:   vm.Count,
			Timeout: vm.Timeout,
		}
	case "passive":
		def.Config = &models.PassiveCheckConfig{
			Timeout: vm.Timeout,
		}
	case "mysql_query", "mysql_query_unixtime", "mysql_replication":
		def.Config = &models.MySQLCheckConfig{
			Host:       vm.Host,
			Port:       vm.Port,
			Timeout:    vm.Timeout,
			UserName:   vm.MySQL.UserName,
			Password:   vm.MySQL.Password,
			DBName:     vm.MySQL.DBName,
			Query:      vm.MySQL.Query,
			Response:   vm.MySQL.Response,
			Difference: vm.MySQL.Difference,
			TableName:  vm.MySQL.TableName,
			Lag:        vm.MySQL.Lag,
			ServerList: vm.MySQL.ServerList,
		}
	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		def.Config = &models.PostgreSQLCheckConfig{
			Host:             vm.Host,
			Port:             vm.Port,
			Timeout:          vm.Timeout,
			UserName:         vm.PgSQL.UserName,
			Password:         vm.PgSQL.Password,
			DBName:           vm.PgSQL.DBName,
			SSLMode:          vm.PgSQL.SSLMode,
			Query:            vm.PgSQL.Query,
			Response:         vm.PgSQL.Response,
			Difference:       vm.PgSQL.Difference,
			TableName:        vm.PgSQL.TableName,
			Lag:              vm.PgSQL.Lag,
			ServerList:       vm.PgSQL.ServerList,
			AnalyticReplicas: vm.PgSQL.AnalyticReplicas,
		}
	case "domain_expiry":
		def.Config = &models.DomainExpiryCheckConfig{
			Domain:            vm.Domain,
			Timeout:           vm.Timeout,
			ExpiryWarningDays: vm.ExpiryWarningDays,
		}
	case "dns":
		def.Config = &models.DNSCheckConfig{
			Host:       vm.Host,
			Domain:     vm.Domain,
			RecordType: vm.RecordType,
			Timeout:    vm.Timeout,
			Expected:   vm.Expected,
		}
	case "ssh":
		def.Config = &models.SSHCheckConfig{
			Host:         vm.Host,
			Port:         vm.Port,
			Timeout:      vm.Timeout,
			ExpectBanner: vm.ExpectBanner,
		}
	case "redis":
		def.Config = &models.RedisCheckConfig{
			Host:     vm.Host,
			Port:     vm.Port,
			Timeout:  vm.Timeout,
			Password: vm.RedisPassword,
			DB:       vm.RedisDB,
		}
	case "ssl_cert":
		def.Config = &models.SSLCertCheckConfig{
			Host:              vm.Host,
			Port:              vm.Port,
			Timeout:           vm.Timeout,
			ExpiryWarningDays: vm.ExpiryWarningDays,
			ValidateChain:     vm.ValidateChain,
		}
	case "smtp":
		def.Config = &models.SMTPCheckConfig{
			Host:     vm.Host,
			Port:     vm.Port,
			Timeout:  vm.Timeout,
			StartTLS: vm.StartTLS,
			Username: vm.SMTPUsername,
			Password: vm.SMTPPassword,
		}
	case "grpc_health":
		def.Config = &models.GRPCHealthCheckConfig{
			Host:    vm.Host,
			Timeout: vm.Timeout,
			UseTLS:  vm.UseTLS,
		}
	case "mongodb":
		def.Config = &models.MongoDBCheckConfig{
			URI:     vm.MongoDBURI,
			Timeout: vm.Timeout,
		}
	case "websocket":
		def.Config = &models.WebSocketCheckConfig{
			URL:     vm.URL,
			Timeout: vm.Timeout,
		}
	default:
		logrus.Warnf("viewModelToCheckDef: unknown check type %q for UUID %s", vm.Type, vm.UUID)
	}

	return def
}

// checkDefsFromViewModels converts a slice of ViewModels to CheckDefinitions.
func checkDefsFromViewModels(vms []models.CheckDefinitionViewModel) []models.CheckDefinition {
	defs := make([]models.CheckDefinition, 0, len(vms))
	for _, vm := range vms {
		defs = append(defs, viewModelToCheckDef(vm))
	}
	return defs
}
