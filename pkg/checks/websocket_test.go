package checks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// echoWSHandler upgrades to WebSocket and echoes messages back.
func echoWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.WriteMessage(mt, msg); err != nil {
			return
		}
	}
}

func TestWebSocketCheck_ConnectOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(echoWSHandler))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	check := &WebSocketCheck{
		URL:     wsURL,
		Timeout: "5s",
	}

	dur, err := check.Run()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if dur <= 0 {
		t.Error("expected positive duration")
	}
}

func TestWebSocketCheck_SendAndExpect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(echoWSHandler))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	check := &WebSocketCheck{
		URL:           wsURL,
		Timeout:       "5s",
		SendMessage:   "ping",
		ExpectMessage: "ping",
	}

	dur, err := check.Run()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if dur <= 0 {
		t.Error("expected positive duration")
	}
}

func TestWebSocketCheck_ExpectMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(echoWSHandler))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	check := &WebSocketCheck{
		URL:           wsURL,
		Timeout:       "5s",
		SendMessage:   "hello",
		ExpectMessage: "world",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestWebSocketCheck_EmptyURL(t *testing.T) {
	check := &WebSocketCheck{
		URL:     "",
		Timeout: "5s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestWebSocketCheck_InvalidTimeout(t *testing.T) {
	check := &WebSocketCheck{
		URL:     "ws://127.0.0.1:9999",
		Timeout: "bad",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestWebSocketCheck_ConnectionRefused(t *testing.T) {
	check := &WebSocketCheck{
		URL:     "ws://127.0.0.1:1/ws",
		Timeout: "1s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
