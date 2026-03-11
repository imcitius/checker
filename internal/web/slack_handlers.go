package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"checker/internal/db"
	"checker/internal/models"
	"checker/internal/slack"
)

// SlackInteractiveHandler handles Slack interactive message payloads (button clicks).
type SlackInteractiveHandler struct {
	signingSecret string
	slackClient   *slack.SlackClient
	repo          db.Repository
}

// NewSlackInteractiveHandler creates a new handler for Slack interactive payloads.
func NewSlackInteractiveHandler(signingSecret string, slackClient *slack.SlackClient, repo db.Repository) *SlackInteractiveHandler {
	return &SlackInteractiveHandler{
		signingSecret: signingSecret,
		slackClient:   slackClient,
		repo:          repo,
	}
}

// SlackInteractionPayload represents the JSON payload from Slack interactive messages.
type SlackInteractionPayload struct {
	Type    string `json:"type"`
	User    SlackUser    `json:"user"`
	Channel SlackChannel `json:"channel"`
	Message SlackMessage `json:"message"`
	Actions []SlackAction `json:"actions"`
	// Container holds metadata about where the interaction occurred.
	Container SlackContainer `json:"container"`
}

// SlackUser represents the user who triggered the interaction.
type SlackUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// SlackChannel represents the channel where the interaction occurred.
type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SlackMessage represents the original message that was interacted with.
type SlackMessage struct {
	Ts     string          `json:"ts"`
	Text   string          `json:"text"`
	Blocks json.RawMessage `json:"blocks"`
}

// SlackAction represents a single action (button click) from the interaction.
type SlackAction struct {
	ActionID string `json:"action_id"`
	BlockID  string `json:"block_id"`
	Value    string `json:"value"`
	Type     string `json:"type"`
}

// SlackContainer represents the container metadata for the interaction.
type SlackContainer struct {
	Type      string `json:"type"`
	MessageTs string `json:"message_ts"`
	ChannelID string `json:"channel_id"`
	IsEphemeral bool `json:"is_ephemeral"`
}

// HandleInteraction handles POST /api/slack/interactive requests.
func (h *SlackInteractiveHandler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("slack interactive: failed to read request body: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Slack request signature
	if !VerifySlackSignature(h.signingSecret, r, body) {
		logrus.Warn("slack interactive: invalid request signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the interaction payload - Slack sends it as form-encoded "payload" field
	payload, err := ParseInteractionPayload(body)
	if err != nil {
		logrus.Errorf("slack interactive: failed to parse payload: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Must respond within 3 seconds - process and respond
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if len(payload.Actions) == 0 {
		logrus.Warn("slack interactive: no actions in payload")
		w.WriteHeader(http.StatusOK)
		return
	}

	action := payload.Actions[0]
	switch action.ActionID {
	case "silence_check":
		h.handleSilenceCheck(ctx, w, payload, action)
	case "silence_project":
		h.handleSilenceProject(ctx, w, payload, action)
	case "ack_alert":
		h.handleAckAlert(ctx, w, payload, action)
	default:
		logrus.Infof("slack interactive: unknown action_id: %s", action.ActionID)
		w.WriteHeader(http.StatusOK)
	}
}

// handleSilenceCheck creates a check-scoped silence and updates the original message.
func (h *SlackInteractiveHandler) handleSilenceCheck(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	checkUUID := action.Value
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts
	duration := 1 * time.Hour

	expiresAt := time.Now().Add(duration)
	silence := models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: userID,
		ExpiresAt:  &expiresAt,
		Reason:     "Silenced via Slack button",
		Active:     true,
	}

	if h.repo != nil {
		if err := h.repo.CreateSilence(ctx, silence); err != nil {
			logrus.Errorf("slack interactive: failed to create silence: %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Post silence confirmation in thread
	if h.slackClient != nil {
		if err := h.slackClient.SendSilenceConfirmation(ctx, channelID, messageTs, "check", checkUUID, "1h", userID); err != nil {
			logrus.Errorf("slack interactive: failed to send silence confirmation: %s", err)
		}

		// Update original message to show "Silenced" badge, remove buttons
		info := slack.CheckAlertInfo{
			UUID:    checkUUID,
			Name:    extractCheckNameFromMessage(payload),
			Project: extractProjectFromMessage(payload),
		}
		blocks := slack.BuildSilencedOriginalBlocks(info, userID)
		fallback := fmt.Sprintf("🔇 SILENCED: %s", info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, blocks, fallback); err != nil {
			logrus.Errorf("slack interactive: failed to update original message: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleSilenceProject creates a project-scoped silence and updates the original message.
func (h *SlackInteractiveHandler) handleSilenceProject(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	projectName := action.Value
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts
	duration := 1 * time.Hour

	expiresAt := time.Now().Add(duration)
	silence := models.AlertSilence{
		Scope:      "project",
		Target:     projectName,
		SilencedBy: userID,
		ExpiresAt:  &expiresAt,
		Reason:     "Silenced via Slack button",
		Active:     true,
	}

	if h.repo != nil {
		if err := h.repo.CreateSilence(ctx, silence); err != nil {
			logrus.Errorf("slack interactive: failed to create project silence: %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Post silence confirmation in thread
	if h.slackClient != nil {
		if err := h.slackClient.SendSilenceConfirmation(ctx, channelID, messageTs, "project", projectName, "1h", userID); err != nil {
			logrus.Errorf("slack interactive: failed to send silence confirmation: %s", err)
		}

		// Update original message to show "Silenced" badge, remove buttons
		info := slack.CheckAlertInfo{
			Name:    extractCheckNameFromMessage(payload),
			Project: projectName,
		}
		blocks := slack.BuildSilencedOriginalBlocks(info, userID)
		fallback := fmt.Sprintf("🔇 SILENCED: %s", info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, blocks, fallback); err != nil {
			logrus.Errorf("slack interactive: failed to update original message: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleAckAlert posts a thread reply acknowledging the alert and updates the original message.
func (h *SlackInteractiveHandler) handleAckAlert(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts
	checkUUID := action.Value

	if h.slackClient != nil {
		// Post thread reply with acknowledgment
		ackText := fmt.Sprintf("👀 Acknowledged by <@%s>", userID)
		if _, err := h.slackClient.PostThreadReply(ctx, channelID, messageTs, ackText); err != nil {
			logrus.Errorf("slack interactive: failed to post ack reply: %s", err)
		}

		// Update original message to show "Acknowledged" badge
		info := slack.CheckAlertInfo{
			UUID:    checkUUID,
			Name:    extractCheckNameFromMessage(payload),
			Project: extractProjectFromMessage(payload),
		}
		blocks := slack.BuildAcknowledgedOriginalBlocks(info, userID)
		fallback := fmt.Sprintf("👀 ACKNOWLEDGED: %s", info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, blocks, fallback); err != nil {
			logrus.Errorf("slack interactive: failed to update original message: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// VerifySlackSignature verifies the Slack request signature using HMAC-SHA256.
// It checks that the timestamp is within 5 minutes and the signature matches.
func VerifySlackSignature(signingSecret string, r *http.Request, body []byte) bool {
	if signingSecret == "" {
		return false
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp is within 5 minutes to prevent replay attacks
	var tsInt int64
	if _, err := fmt.Sscanf(timestamp, "%d", &tsInt); err != nil {
		return false
	}
	ts := time.Unix(tsInt, 0)
	if math.Abs(time.Since(ts).Minutes()) > 5 {
		return false
	}

	// Compute HMAC-SHA256 of "v0:{timestamp}:{body}"
	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBasestring))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Timing-safe comparison
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// ParseInteractionPayload parses the Slack interaction payload from form-encoded body.
// Slack sends interactive payloads as application/x-www-form-urlencoded with a "payload" field
// containing URL-encoded JSON.
func ParseInteractionPayload(body []byte) (SlackInteractionPayload, error) {
	var payload SlackInteractionPayload

	// Parse form data to extract "payload" field
	formData := parseFormValues(string(body))
	payloadJSON, ok := formData["payload"]
	if !ok || payloadJSON == "" {
		return payload, fmt.Errorf("missing payload field in request body")
	}

	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return payload, fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	return payload, nil
}

// parseFormValues parses application/x-www-form-urlencoded data into a map.
func parseFormValues(body string) map[string]string {
	result := make(map[string]string)
	if body == "" {
		return result
	}

	pairs := splitFormPairs(body)
	for _, pair := range pairs {
		idx := indexByte(pair, '=')
		if idx < 0 {
			continue
		}
		key := formUnescape(pair[:idx])
		value := formUnescape(pair[idx+1:])
		result[key] = value
	}
	return result
}

// splitFormPairs splits a form-encoded string by "&".
func splitFormPairs(s string) []string {
	var pairs []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			pairs = append(pairs, s[start:i])
			start = i + 1
		}
	}
	pairs = append(pairs, s[start:])
	return pairs
}

// indexByte returns the index of the first occurrence of c in s, or -1.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// formUnescape performs URL decoding for form values.
func formUnescape(s string) string {
	// Replace + with space first
	var result []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '+':
			result = append(result, ' ')
		case '%':
			if i+2 < len(s) {
				b, err := hex.DecodeString(s[i+1 : i+3])
				if err == nil {
					result = append(result, b...)
					i += 2
					continue
				}
			}
			result = append(result, s[i])
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// extractCheckNameFromMessage attempts to extract the check name from the original message text.
// The fallback text format is "{emoji} ALERT: {name}: {message}".
func extractCheckNameFromMessage(payload SlackInteractionPayload) string {
	text := payload.Message.Text
	if text == "" {
		return "Unknown Check"
	}
	// Try to extract name from fallback text: "{emoji} {name}: {message}" or "{emoji} ALERT: {name}: {message}"
	// Simple heuristic: find text after the first emoji+space pattern
	for i := 0; i < len(text); i++ {
		// Skip until we find a space after possible emoji bytes
		if text[i] == ' ' {
			remaining := text[i+1:]
			// Check for common prefixes
			for _, prefix := range []string{"ALERT: ", "SILENCED: ", "ACKNOWLEDGED: "} {
				if len(remaining) > len(prefix) && remaining[:len(prefix)] == prefix {
					remaining = remaining[len(prefix):]
					break
				}
			}
			// Take everything up to the next ":" as the name
			for j := 0; j < len(remaining); j++ {
				if remaining[j] == ':' {
					return remaining[:j]
				}
			}
			return remaining
		}
	}
	return "Unknown Check"
}

// extractProjectFromMessage attempts to extract the project name from the original message.
// Since we can't easily parse Block Kit from the raw message, we use the action value
// when available. This is a best-effort extraction.
func extractProjectFromMessage(payload SlackInteractionPayload) string {
	// Try to find the silence_project action value as a fallback for project name
	for _, action := range payload.Actions {
		if action.ActionID == "silence_project" {
			return action.Value
		}
	}
	return ""
}
