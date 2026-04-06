package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imcitius/checker/pkg/models"
	"github.com/sirupsen/logrus"
)

const (
	heartbeatInterval  = 30 * time.Second
	initialBackoff     = 1 * time.Second
	maxBackoff         = 60 * time.Second
	resultBufferSize   = 256
	edgeClientVersion  = "1.0.0"
)

// ClientConfig holds the configuration for the edge WebSocket client.
type ClientConfig struct {
	SaaSURL    string // wss://app.example.com/ws/edge
	APIKey     string // ck_... token
	Region     string // e.g. "edge-tokyo-1"
	MaxWorkers int    // default 10
}

// Client is the edge WebSocket client. It connects to the SaaS core, receives
// check definitions, executes them via the EdgeScheduler, and reports results back.
type Client struct {
	cfg       ClientConfig
	scheduler *EdgeScheduler
	results   chan CheckResult
	startTime time.Time
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg ClientConfig) *Client {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = defaultEdgeWorkers
	}
	results := make(chan CheckResult, resultBufferSize)
	return &Client{
		cfg:     cfg,
		results: results,
	}
}

// Run is the main loop: connects to SaaS, handles messages, reconnects on error.
// It returns when ctx is cancelled, or with a fatal error if the context is still
// active.
func (c *Client) Run(ctx context.Context) error {
	c.startTime = time.Now()
	backoff := initialBackoff

	for {
		if ctx.Err() != nil {
			return nil
		}
		logrus.Infof("EdgeClient: connecting to %s (region=%s)", c.cfg.SaaSURL, c.cfg.Region)
		err := c.connect(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Cancelled — clean exit.
				return nil
			}
			logrus.Errorf("EdgeClient: connection error: %v — reconnecting in %s", err, backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = nextBackoff(backoff)
			continue
		}
		// Successful session — reset backoff.
		backoff = initialBackoff
	}
}

// connect establishes one WebSocket session and runs until it ends.
func (c *Client) connect(ctx context.Context) error {
	u, err := buildURL(c.cfg.SaaSURL, c.cfg.APIKey, c.cfg.Region)
	if err != nil {
		return fmt.Errorf("invalid SaaSURL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	logrus.Infof("EdgeClient: connected to %s", u)

	// Create a fresh scheduler for this session.
	sched := NewEdgeScheduler(c.cfg.MaxWorkers, c.results)
	c.scheduler = sched

	schedCtx, cancelSched := context.WithCancel(ctx)
	defer cancelSched()

	// Start scheduler.
	go func() {
		sched.Run(schedCtx)
	}()

	// Start result sender.
	go c.sendResults(schedCtx, conn)

	// Start heartbeat sender.
	go c.sendHeartbeats(schedCtx, conn, sched)

	// Read loop (blocks until connection closes or ctx cancelled).
	return c.readLoop(ctx, conn, sched)
}

// readLoop reads incoming messages from the WebSocket connection and dispatches them.
func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, sched *EdgeScheduler) error {
	for {
		// Set a generous read deadline; heartbeats keep the connection alive.
		_ = conn.SetReadDeadline(time.Now().Add(2 * heartbeatInterval))

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		if err := c.handleMessage(raw, conn, sched); err != nil {
			logrus.Warnf("EdgeClient: handleMessage error: %v", err)
		}
	}
}

// handleMessage decodes and dispatches a single incoming message.
func (c *Client) handleMessage(raw []byte, conn *websocket.Conn, sched *EdgeScheduler) error {
	// Peek at the type field.
	var base models.EdgeMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	switch base.Type {
	case "config_sync":
		var msg models.EdgeConfigSync
		if err := json.Unmarshal(raw, &msg); err != nil {
			return fmt.Errorf("unmarshal config_sync: %w", err)
		}
		defs := checkDefsFromViewModels(msg.Checks)
		logrus.Infof("EdgeClient: config_sync — loading %d checks", len(defs))
		sched.ReplaceAll(defs)

	case "config_patch":
		var msg models.EdgeConfigPatch
		if err := json.Unmarshal(raw, &msg); err != nil {
			return fmt.Errorf("unmarshal config_patch: %w", err)
		}
		switch msg.Action {
		case "add", "update":
			if msg.Check == nil {
				return fmt.Errorf("config_patch %s: check is nil", msg.Action)
			}
			def := viewModelToCheckDef(*msg.Check)
			logrus.Infof("EdgeClient: config_patch %s check %s", msg.Action, def.UUID)
			sched.AddOrUpdate(def)
		case "delete":
			logrus.Infof("EdgeClient: config_patch delete check %s", msg.UUID)
			sched.Delete(msg.UUID)
		default:
			logrus.Warnf("EdgeClient: unknown config_patch action %q", msg.Action)
		}

	case "ping":
		// Respond with a pong.
		pong := models.EdgeMessage{Type: "pong"}
		data, _ := json.Marshal(pong)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logrus.Warnf("EdgeClient: failed to send pong: %v", err)
		}

	default:
		logrus.Debugf("EdgeClient: unhandled message type %q", base.Type)
	}

	return nil
}

// sendResults drains the results channel and sends each result over the WebSocket.
func (c *Client) sendResults(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-c.results:
			if !ok {
				return
			}
			msg := models.EdgeResult{
				Type:      "result",
				CheckUUID: r.CheckUUID,
				IsHealthy: r.IsHealthy,
				Message:   r.Message,
				Duration:  r.Duration,
				Timestamp: r.Timestamp,
			}
			data, err := json.Marshal(msg)
			if err != nil {
				logrus.Errorf("EdgeClient: marshal result: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logrus.Warnf("EdgeClient: send result failed: %v", err)
				return
			}
		}
	}
}

// sendHeartbeats sends a heartbeat message every heartbeatInterval.
func (c *Client) sendHeartbeats(ctx context.Context, conn *websocket.Conn, sched *EdgeScheduler) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := models.EdgeHeartbeat{
				Type:          "heartbeat",
				Version:       edgeClientVersion,
				Region:        c.cfg.Region,
				WorkerCount:   c.cfg.MaxWorkers,
				ActiveChecks:  sched.ActiveCount(),
				UptimeSeconds: int64(time.Since(c.startTime).Seconds()),
			}
			data, err := json.Marshal(hb)
			if err != nil {
				logrus.Errorf("EdgeClient: marshal heartbeat: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logrus.Warnf("EdgeClient: send heartbeat failed: %v", err)
				return
			}
			logrus.Debugf("EdgeClient: sent heartbeat (active_checks=%d, uptime=%ds)",
				hb.ActiveChecks, hb.UptimeSeconds)
		}
	}
}

// buildURL constructs the WebSocket URL with query parameters.
func buildURL(rawURL, apiKey, region string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("api_key", apiKey)
	q.Set("region", region)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// nextBackoff returns the next exponential backoff duration, capped at maxBackoff.
func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		next = maxBackoff
	}
	return next
}
