package scheduler

import (
	"testing"
	"time"

	"github.com/imcitius/checker/internal/actors"
	"github.com/imcitius/checker/pkg/checks"
	"github.com/imcitius/checker/pkg/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCheckerFactory_Http verifies that CheckerFactory returns an HTTPCheck for a valid HTTP configuration.
func TestCheckerFactory_Http(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_Http")
	checkDef := models.CheckDefinition{
		ID:   primitive.NewObjectID(),
		UUID: "test-uuid",
		Type: "http",
		Config: &models.HTTPCheckConfig{
			URL:                 "https://example.com",
			Timeout:             "5s",
			Answer:              "ok",
			AnswerPresent:       true,
			Code:                []int{200},
			Headers:             []map[string]string{{"Content-Type": "application/json"}},
			SkipCheckSSL:        false,
			SSLExpirationPeriod: "720h",
			StopFollowRedirects: false,
		},
	}
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for HTTP check")
	}

	// Assert the returned checker is of type *checks.HTTPCheck.
	httpCheck, ok := checker.(*checks.HTTPCheck)
	if !ok {
		t.Errorf("Expected *checks.HTTPCheck, got %T", checker)
	}

	// Verify fields were set correctly
	if httpCheck.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got '%s'", httpCheck.URL)
	}
	if httpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", httpCheck.Timeout)
	}
	if httpCheck.Answer != "ok" {
		t.Errorf("Expected Answer to be 'ok', got '%s'", httpCheck.Answer)
	}
	if httpCheck.AnswerPresent != true {
		t.Errorf("Expected AnswerPresent to be true, got %v", httpCheck.AnswerPresent)
	}
	if !httpCheck.SkipCheckSSL == checkDef.Config.(*models.HTTPCheckConfig).SkipCheckSSL {
		t.Errorf("Expected SkipCheckSSL to be %v, got %v", checkDef.Config.(*models.HTTPCheckConfig).SkipCheckSSL, httpCheck.SkipCheckSSL)
	}
}

// TestCheckerFactory_TCP verifies that CheckerFactory returns a TCPCheck for a valid TCP configuration.
func TestCheckerFactory_TCP(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_TCP")
	checkDef := models.CheckDefinition{
		ID:   primitive.NewObjectID(),
		UUID: "test-uuid",
		Type: "tcp",
		Config: &models.TCPCheckConfig{
			Host:    "example.com",
			Port:    80,
			Timeout: "5s",
		},
	}
	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for TCP check")
	}

	// Assert the returned checker is of type *checks.TCPCheck.
	tcpCheck, ok := checker.(*checks.TCPCheck)
	if !ok {
		t.Errorf("Expected *checks.TCPCheck, got %T", checker)
	}

	// Verify fields were set correctly
	if tcpCheck.Host != "example.com" {
		t.Errorf("Expected Host to be 'example.com', got '%s'", tcpCheck.Host)
	}
	if tcpCheck.Port != 80 {
		t.Errorf("Expected Port to be 80, got %d", tcpCheck.Port)
	}
	if tcpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", tcpCheck.Timeout)
	}
}

// TestCheckerFactory_ICMP verifies that CheckerFactory returns an ICMPCheck for a valid ICMP configuration.
func TestCheckerFactory_ICMP(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_ICMP")

	checkDef := models.CheckDefinition{
		ID:   primitive.NewObjectID(),
		UUID: "test-uuid",
		Type: "icmp",
		Config: &models.ICMPCheckConfig{
			Host:    "example.com",
			Count:   4,
			Timeout: "5s",
		},
	}

	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for ICMP check")
	}

	// Verify the type
	icmpCheck, ok := checker.(*checks.ICMPCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type")
	}

	// Verify the fields
	if icmpCheck.Host != "example.com" {
		t.Errorf("Expected Host to be 'example.com', got '%s'", icmpCheck.Host)
	}
	if icmpCheck.Count != 4 {
		t.Errorf("Expected Count to be 4, got %d", icmpCheck.Count)
	}
	if icmpCheck.Timeout != "5s" {
		t.Errorf("Expected Timeout to be '5s', got '%s'", icmpCheck.Timeout)
	}
}

// TestCheckerFactory_PostgreSQL verifies that CheckerFactory returns a PostgreSQLCheck for valid PostgreSQL configuration.
func TestCheckerFactory_PostgreSQL(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_PostgreSQL")

	checkDef := models.CheckDefinition{
		Type: "pgsql_query",
		Config: &models.PostgreSQLCheckConfig{
			Host:     "db.example.com",
			Port:     5432,
			Timeout:  "5s",
			UserName: "postgres",
			Password: "password",
			DBName:   "testdb",
			Query:    "SELECT 1",
			Response: "1",
		},
	}

	checker := CheckerFactory(checkDef, logger)

	if checker == nil {
		t.Fatal("CheckerFactory returned nil for PostgreSQL check")
	}

	_, ok := checker.(*checks.PostgreSQLCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type")
	}
}

// TestCheckerFactory_PostgreSQLTime verifies that CheckerFactory returns a PostgreSQLTimeCheck for valid PostgreSQL time check configuration.
func TestCheckerFactory_PostgreSQLTime(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_PostgreSQLTime")

	// Test timestamp type
	timestampCheckDef := models.CheckDefinition{
		Type: "pgsql_query_timestamp",
		Config: &models.PostgreSQLCheckConfig{
			Host:       "db.example.com",
			Port:       5432,
			Timeout:    "5s",
			UserName:   "postgres",
			Password:   "password",
			DBName:     "testdb",
			Query:      "SELECT NOW()",
			Difference: "5m",
		},
	}

	checker := CheckerFactory(timestampCheckDef, logger)

	if checker == nil {
		t.Fatal("CheckerFactory returned nil for PostgreSQL timestamp check")
	}

	timeChecker, ok := checker.(*checks.PostgreSQLTimeCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type for timestamp check")
	}

	if timeChecker.TimeType != "timestamp" {
		t.Fatalf("Expected TimeType to be 'timestamp', got '%s'", timeChecker.TimeType)
	}

	// Test unixtime type
	unixtimeCheckDef := models.CheckDefinition{
		Type: "pgsql_query_unixtime",
		Config: &models.PostgreSQLCheckConfig{
			Host:       "db.example.com",
			Port:       5432,
			Timeout:    "5s",
			UserName:   "postgres",
			Password:   "password",
			DBName:     "testdb",
			Query:      "SELECT EXTRACT(EPOCH FROM NOW())",
			Difference: "5m",
		},
	}

	unixtimeChecker := CheckerFactory(unixtimeCheckDef, logger)

	if unixtimeChecker == nil {
		t.Fatal("CheckerFactory returned nil for PostgreSQL unixtime check")
	}

	timeChecker, ok = unixtimeChecker.(*checks.PostgreSQLTimeCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type for unixtime check")
	}

	if timeChecker.TimeType != "unixtime" {
		t.Fatalf("Expected TimeType to be 'unixtime', got '%s'", timeChecker.TimeType)
	}
}

// TestCheckerFactory_PostgreSQLReplication verifies that CheckerFactory returns a PostgreSQLReplicationCheck for valid PostgreSQL replication configuration.
func TestCheckerFactory_PostgreSQLReplication(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_PostgreSQLReplication")

	// Test replication check
	replCheckDef := models.CheckDefinition{
		Type: "pgsql_replication",
		Config: &models.PostgreSQLCheckConfig{
			Host:       "db.example.com",
			Port:       5432,
			Timeout:    "5s",
			UserName:   "postgres",
			Password:   "password",
			DBName:     "testdb",
			TableName:  "repl_test",
			Lag:        "5s",
			ServerList: []string{"replica1", "replica2"},
		},
	}

	checker := CheckerFactory(replCheckDef, logger)

	if checker == nil {
		t.Fatal("CheckerFactory returned nil for PostgreSQL replication check")
	}

	replChecker, ok := checker.(*checks.PostgreSQLReplicationCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type for replication check")
	}

	if replChecker.CheckType != "replication" {
		t.Fatalf("Expected CheckType to be 'replication', got '%s'", replChecker.CheckType)
	}

	// Test replication status check
	statusCheckDef := models.CheckDefinition{
		Type: "pgsql_replication_status",
		Config: &models.PostgreSQLCheckConfig{
			Host:             "db.example.com",
			Port:             5432,
			Timeout:          "5s",
			UserName:         "postgres",
			Password:         "password",
			DBName:           "testdb",
			Lag:              "5s",
			AnalyticReplicas: []string{"replica_analytics"},
		},
	}

	statusChecker := CheckerFactory(statusCheckDef, logger)

	if statusChecker == nil {
		t.Fatal("CheckerFactory returned nil for PostgreSQL replication status check")
	}

	replChecker, ok = statusChecker.(*checks.PostgreSQLReplicationCheck)
	if !ok {
		t.Fatal("CheckerFactory returned wrong type for replication status check")
	}

	if replChecker.CheckType != "replication_status" {
		t.Fatalf("Expected CheckType to be 'replication_status', got '%s'", replChecker.CheckType)
	}
}

// TestCheckerFactory_Passive verifies that CheckerFactory returns a PassiveCheck with all required fields populated.
func TestCheckerFactory_Passive(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_Passive")
	lastRun := time.Now().Add(-2 * time.Minute)
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		Name:      "my-passive-check",
		Project:   "my-project",
		GroupName: "my-group",
		Type:      "passive",
		LastRun:   lastRun,
		Config: &models.PassiveCheckConfig{
			Timeout: "15m",
		},
	}

	checker := CheckerFactory(checkDef, logger)
	if checker == nil {
		t.Fatal("CheckerFactory returned nil for passive check")
	}

	passiveCheck, ok := checker.(*checks.PassiveCheck)
	if !ok {
		t.Fatalf("Expected *checks.PassiveCheck, got %T", checker)
	}

	if passiveCheck.Timeout != "15m" {
		t.Errorf("Expected Timeout '15m', got '%s'", passiveCheck.Timeout)
	}
	if passiveCheck.Logger == nil {
		t.Error("Expected Logger to be set, got nil")
	}
	if passiveCheck.ErrorHeader == "" {
		t.Error("Expected ErrorHeader to be set, got empty string")
	}
	if !passiveCheck.LastPing.Equal(lastRun) {
		t.Errorf("Expected LastPing to be %v, got %v", lastRun, passiveCheck.LastPing)
	}
}

// TestCheckerFactory_Unknown verifies that CheckerFactory returns nil for an unknown check type.
func TestCheckerFactory_Unknown(t *testing.T) {
	logger := logrus.WithField("test", "TestCheckerFactory_Unknown")
	checkDef := models.CheckDefinition{
		ID:   primitive.NewObjectID(),
		UUID: "test-uuid",
		Type: "unsupported",
	}
	checker := CheckerFactory(checkDef, logger)
	if checker != nil {
		t.Errorf("Expected nil for unknown check type, got %T", checker)
	}
}

// TestActorFactory_Log verifies that ActorFactory returns a LogActor for a valid Log configuration.
func TestActorFactory_Log(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "log",
	}
	actor, err := ActorFactory(checkDef)
	if err != nil {
		t.Fatalf("ActorFactory returned error for Log actor: %v", err)
	}
	if actor == nil {
		t.Fatal("ActorFactory returned nil for Log actor")
	}
}

// TestActorFactory_Webhook verifies that ActorFactory returns a WebhookActor for a valid Webhook configuration.
func TestActorFactory_Webhook(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "webhook",
		ActorConfig: &models.WebhookConfig{
			URL:     "https://webhook.site/test",
			Method:  "POST",
			Payload: `{"text": "hello"}`,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}
	actor, err := ActorFactory(checkDef)
	if err != nil {
		t.Fatalf("ActorFactory returned error for Webhook actor: %v", err)
	}
	if actor == nil {
		t.Fatal("ActorFactory returned nil for Webhook actor")
	}

	webhookActor, ok := actor.(*actors.WebhookActor)
	if !ok {
		t.Fatalf("Expected *actors.WebhookActor, got %T", actor)
	}

	if webhookActor.URL != "https://webhook.site/test" {
		t.Errorf("Expected URL https://webhook.site/test, got %s", webhookActor.URL)
	}
	if webhookActor.Method != "POST" {
		t.Errorf("Expected Method POST, got %s", webhookActor.Method)
	}
}

// TestActorFactory_Alert verifies that ActorFactory rejects alert actor type.
// Alerts are now handled separately in the sendAlerts() function and should not go through ActorFactory.
func TestActorFactory_Alert(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "alert",
	}
	actor, err := ActorFactory(checkDef)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if actor != nil {
		t.Errorf("Expected nil actor, got %T", actor)
	}
}

// TestActorFactory_Unknown verifies that ActorFactory returns an error for an unknown actor type.
func TestActorFactory_Unknown(t *testing.T) {
	checkDef := models.CheckDefinition{
		ID:        primitive.NewObjectID(),
		UUID:      "test-uuid",
		ActorType: "unsupported",
	}
	actor, err := ActorFactory(checkDef)
	if err == nil {
		t.Error("Expected error for unknown actor type, got nil")
	}
	if actor != nil {
		t.Errorf("Expected nil actor for unknown actor type, got %T", actor)
	}
}
