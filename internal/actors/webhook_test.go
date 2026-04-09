// SPDX-License-Identifier: BUSL-1.1

package actors

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWebhookActor_Act(t *testing.T) {
	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.Header.Get("X-Custom-Header") != "TestValue" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		expectedBody := `{"key":"value"}`
		if string(body) != expectedBody {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Logger
	logger := logrus.NewEntry(logrus.New())

	t.Run("Successful Webhook", func(t *testing.T) {
		actor := &WebhookActor{
			URL:     server.URL,
			Method:  "POST",
			Payload: `{"key":"value"}`,
			Headers: map[string]string{
				"X-Custom-Header": "TestValue",
			},
			Logger: logger,
		}

		err := actor.Act("test message")
		assert.NoError(t, err)
	})

	t.Run("Empty URL", func(t *testing.T) {
		actor := &WebhookActor{
			URL:    "",
			Logger: logger,
		}

		err := actor.Act("test message")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook URL is empty")
	})

	t.Run("Server Error", func(t *testing.T) {
		// Server that returns 500
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer errorServer.Close()

		actor := &WebhookActor{
			URL:    errorServer.URL,
			Method: "GET",
			Logger: logger,
		}

		err := actor.Act("test message")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook returned error status: 500")
	})
}
