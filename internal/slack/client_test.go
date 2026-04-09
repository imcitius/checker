// SPDX-License-Identifier: BUSL-1.1

package slack

import (
	"testing"
)

func TestNewSlackClient(t *testing.T) {
	client := NewSlackClient("xoxb-test-token", "signing-secret-123", "C12345678")

	if client == nil {
		t.Fatal("NewSlackClient returned nil")
	}
	if client.botToken != "xoxb-test-token" {
		t.Errorf("botToken = %q, want %q", client.botToken, "xoxb-test-token")
	}
	if client.signingSecret != "signing-secret-123" {
		t.Errorf("signingSecret = %q, want %q", client.signingSecret, "signing-secret-123")
	}
	if client.defaultChannelID != "C12345678" {
		t.Errorf("defaultChannelID = %q, want %q", client.defaultChannelID, "C12345678")
	}
	if client.api == nil {
		t.Error("slack API client should not be nil")
	}
}

func TestDefaultChannelID(t *testing.T) {
	client := NewSlackClient("token", "secret", "C99999")
	if got := client.DefaultChannelID(); got != "C99999" {
		t.Errorf("DefaultChannelID() = %q, want %q", got, "C99999")
	}
}
