package checks

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	ErrWebSocketDial           = "websocket dial error: %s"
	ErrWebSocketSend           = "websocket send error: %s"
	ErrWebSocketRead           = "websocket read error: %s"
	ErrWebSocketExpectMismatch = "websocket expected message containing %q, got %q"
	ErrWebSocketEmptyURL       = "websocket url is empty"
)

// WebSocketCheck represents a WebSocket health check.
type WebSocketCheck struct {
	URL           string
	Timeout       string
	SendMessage   string
	ExpectMessage string
	Headers       http.Header
	Logger        *logrus.Entry
}

// Run executes the WebSocket health check.
func (check *WebSocketCheck) Run() (time.Duration, error) {
	start := time.Now()

	if check.URL == "" {
		return time.Since(start), fmt.Errorf(ErrWebSocketEmptyURL)
	}

	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "websocket")
	}

	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
	}

	var reqHeader http.Header
	if check.Headers != nil {
		reqHeader = check.Headers
	}

	conn, _, err := dialer.Dial(check.URL, reqHeader)
	if err != nil {
		check.Logger.WithError(err).Debugf("WebSocket dial failed for %s", check.URL)
		return time.Since(start), fmt.Errorf(ErrWebSocketDial, err)
	}
	defer func() {
		// Send a clean close message
		_ = conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		conn.Close()
	}()

	// Set read/write deadline based on timeout
	deadline := time.Now().Add(timeout)
	_ = conn.SetWriteDeadline(deadline)
	_ = conn.SetReadDeadline(deadline)

	// Send message if configured
	if check.SendMessage != "" {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(check.SendMessage)); err != nil {
			check.Logger.WithError(err).Debug("WebSocket send failed")
			return time.Since(start), fmt.Errorf(ErrWebSocketSend, err)
		}
	}

	// Read and verify message if configured
	if check.ExpectMessage != "" {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			check.Logger.WithError(err).Debug("WebSocket read failed")
			return time.Since(start), fmt.Errorf(ErrWebSocketRead, err)
		}

		received := string(msg)
		if !strings.Contains(received, check.ExpectMessage) {
			return time.Since(start), fmt.Errorf(ErrWebSocketExpectMismatch, check.ExpectMessage, received)
		}
	}

	return time.Since(start), nil
}
