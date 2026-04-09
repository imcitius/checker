// SPDX-License-Identifier: BUSL-1.1

package actors

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// WebhookActor allows sending a custom HTTP request as an action
type WebhookActor struct {
	URL     string
	Method  string
	Payload string
	Headers map[string]string
	Logger  *logrus.Entry
}

// Act sends the HTTP request
func (wa *WebhookActor) Act(msg string) error {
	if wa.Logger == nil {
		wa.Logger = logrus.WithField("actor", "webhook")
	}

	wa.Logger.Infof("Executing webhook action to %s", wa.URL)

	if wa.URL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	method := wa.Method
	if method == "" {
		method = "GET"
	}

	var req *http.Request
	var err error

	if wa.Payload != "" {
		req, err = http.NewRequest(method, wa.URL, bytes.NewBufferString(wa.Payload))
	} else {
		req, err = http.NewRequest(method, wa.URL, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Add headers
	if wa.Headers != nil {
		for k, v := range wa.Headers {
			req.Header.Set(k, v)
		}
	}

	// Default Content-Type if not set and payload exists
	if wa.Payload != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	wa.Logger.Infof("Webhook action completed with status: %d", resp.StatusCode)
	return nil
}
