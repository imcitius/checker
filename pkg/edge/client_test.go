package edge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// echoServer creates a test WebSocket server that reads and discards all messages.
// It returns the server URL (ws://...) and a closer function.
func echoServer(t *testing.T) (string, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Drain all incoming messages until the connection closes.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return wsURL, srv.Close
}

// TestWsConn_ConcurrentWrites verifies that wsConn serialises concurrent
// WriteMessage calls and does not panic or race under the -race detector.
func TestWsConn_ConcurrentWrites(t *testing.T) {
	wsURL, closeServer := echoServer(t)
	defer closeServer()

	rawConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	wc := &wsConn{Conn: rawConn}
	defer wc.Close()

	const goroutines = 20
	const messagesEach = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	payload := []byte(`{"type":"test"}`)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < messagesEach; j++ {
				if err := wc.WriteMessage(websocket.TextMessage, payload); err != nil {
					// Connection may close mid-test; that's fine — just stop.
					return
				}
			}
		}()
	}

	wg.Wait()
}
