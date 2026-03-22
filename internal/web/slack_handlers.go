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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"checker/internal/db"
	"checker/internal/models"
	"checker/internal/slack"
)

// SlackWebhookRegistrar registers Slack interactive and command routes.
type SlackWebhookRegistrar struct {
	Client *slack.SlackClient
}

func (r *SlackWebhookRegistrar) RegisterRoutes(router *gin.Engine, repo db.Repository) {
	handler := NewSlackInteractiveHandler(r.Client.SigningSecret(), r.Client, repo)
	router.POST("/api/slack/interactive", gin.WrapF(handler.HandleInteraction))
	logrus.Info("Slack interactive endpoint registered at /api/slack/interactive")
	router.POST("/api/slack/commands", gin.WrapF(handler.HandleSlashCommand))
	logrus.Info("Slack slash command endpoint registered at /api/slack/commands")
}

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

// SlackAction represents a single action (button click or select) from the interaction.
type SlackAction struct {
	ActionID       string              `json:"action_id"`
	BlockID        string              `json:"block_id"`
	Value          string              `json:"value"`
	Type           string              `json:"type"`
	SelectedOption *SlackOptionObject  `json:"selected_option,omitempty"`
}

// SlackOptionObject represents a selected option from a static select menu.
type SlackOptionObject struct {
	Value string `json:"value"`
	Text  struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"text"`
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
	case "unsilence":
		h.handleUnsilence(ctx, w, payload, action)
	default:
		logrus.Infof("slack interactive: unknown action_id: %s", action.ActionID)
		w.WriteHeader(http.StatusOK)
	}
}

// parseSilenceDuration parses a duration string from a select option value.
// Format: "target|duration" where duration is "30m", "1h", "4h", "8h", "24h", or "indefinite".
func parseSilenceDuration(optionValue string) (target string, duration time.Duration, durationLabel string, indefinite bool) {
	parts := strings.SplitN(optionValue, "|", 2)
	if len(parts) != 2 {
		return optionValue, 1 * time.Hour, "1h", false
	}
	target = parts[0]
	durationStr := parts[1]

	switch durationStr {
	case "30m":
		return target, 30 * time.Minute, "30m", false
	case "1h":
		return target, 1 * time.Hour, "1h", false
	case "4h":
		return target, 4 * time.Hour, "4h", false
	case "8h":
		return target, 8 * time.Hour, "8h", false
	case "24h":
		return target, 24 * time.Hour, "24h", false
	case "indefinite":
		return target, 0, "indefinitely", true
	default:
		return target, 1 * time.Hour, "1h", false
	}
}

// handleSilenceCheck creates a check-scoped silence and updates the original message.
func (h *SlackInteractiveHandler) handleSilenceCheck(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts

	// Parse target and duration from selected option
	optionValue := ""
	if action.SelectedOption != nil {
		optionValue = action.SelectedOption.Value
	} else {
		optionValue = action.Value
	}
	checkUUID, duration, durationLabel, indefinite := parseSilenceDuration(optionValue)

	silence := models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: userID,
		Reason:     fmt.Sprintf("Silenced via Slack for %s", durationLabel),
		Active:     true,
	}
	if !indefinite {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
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
		if err := h.slackClient.SendSilenceConfirmation(ctx, channelID, messageTs, "check", checkUUID, durationLabel, userID); err != nil {
			logrus.Errorf("slack interactive: failed to send silence confirmation: %s", err)
		}

		// Update original message to show "Silenced" badge with un-silence button
		info := slack.CheckAlertInfo{
			UUID:    checkUUID,
			Name:    extractCheckNameFromMessage(payload),
			Project: extractProjectFromMessage(payload),
		}
		blocks := slack.BuildSilencedOriginalBlocks(info, userID, "check", checkUUID)
		fallback := fmt.Sprintf("🔇 SILENCED: %s", info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, blocks, fallback); err != nil {
			logrus.Errorf("slack interactive: failed to update original message: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleSilenceProject creates a project-scoped silence and updates the original message.
func (h *SlackInteractiveHandler) handleSilenceProject(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts

	// Parse target and duration from selected option
	optionValue := ""
	if action.SelectedOption != nil {
		optionValue = action.SelectedOption.Value
	} else {
		optionValue = action.Value
	}
	projectName, duration, durationLabel, indefinite := parseSilenceDuration(optionValue)

	silence := models.AlertSilence{
		Scope:      "project",
		Target:     projectName,
		SilencedBy: userID,
		Reason:     fmt.Sprintf("Silenced via Slack for %s", durationLabel),
		Active:     true,
	}
	if !indefinite {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
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
		if err := h.slackClient.SendSilenceConfirmation(ctx, channelID, messageTs, "project", projectName, durationLabel, userID); err != nil {
			logrus.Errorf("slack interactive: failed to send silence confirmation: %s", err)
		}

		// Update original message to show "Silenced" badge with un-silence button
		info := slack.CheckAlertInfo{
			Name:    extractCheckNameFromMessage(payload),
			Project: projectName,
		}
		blocks := slack.BuildSilencedOriginalBlocks(info, userID, "project", projectName)
		fallback := fmt.Sprintf("🔇 SILENCED: %s", info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, blocks, fallback); err != nil {
			logrus.Errorf("slack interactive: failed to update original message: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleUnsilence deactivates active silences and restores the original alert message with action buttons.
func (h *SlackInteractiveHandler) handleUnsilence(ctx context.Context, w http.ResponseWriter, payload SlackInteractionPayload, action SlackAction) {
	userID := payload.User.ID
	channelID := payload.Channel.ID
	messageTs := payload.Message.Ts

	// Parse scope and target from action value: "scope|target"
	parts := strings.SplitN(action.Value, "|", 2)
	if len(parts) != 2 {
		logrus.Errorf("slack interactive: invalid unsilence value: %s", action.Value)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	scope := parts[0]
	target := parts[1]

	if h.repo != nil {
		if err := h.repo.DeactivateSilence(ctx, scope, target); err != nil {
			logrus.Errorf("slack interactive: failed to deactivate silence: %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	if h.slackClient != nil {
		// Post un-silence confirmation in thread (rich blocks version)
		if err := h.slackClient.SendUnsilenceConfirmation(ctx, channelID, messageTs, scope, target, userID); err != nil {
			logrus.Errorf("slack interactive: failed to send unsilence confirmation: %s", err)
		}

		// Restore the original alert message with action buttons
		checkUUID := ""
		project := ""
		if scope == "check" {
			checkUUID = target
			project = extractProjectFromMessage(payload)
		} else {
			project = target
		}
		info := slack.CheckAlertInfo{
			UUID:    checkUUID,
			Name:    extractCheckNameFromMessage(payload),
			Project: project,
		}
		restoredBlocks := slack.BuildAlertBlocks(info)
		fallbackText := fmt.Sprintf("%s ALERT: %s", slack.SeverityEmoji(info), info.Name)
		if err := h.slackClient.UpdateMessage(ctx, channelID, messageTs, restoredBlocks, fallbackText); err != nil {
			logrus.Errorf("slack interactive: failed to restore original message: %s", err)
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

// parseSlashDuration parses a duration string from a slash command argument.
// Returns the parsed duration, a human-readable label, and whether it's indefinite.
// Returns ok=false if the duration string is not recognized.
func parseSlashDuration(s string) (duration time.Duration, label string, ok bool) {
	switch s {
	case "30m":
		return 30 * time.Minute, "30m", true
	case "1h":
		return 1 * time.Hour, "1h", true
	case "4h":
		return 4 * time.Hour, "4h", true
	case "8h":
		return 8 * time.Hour, "8h", true
	case "24h":
		return 24 * time.Hour, "24h", true
	case "indefinite":
		return 0, "indefinitely", true
	default:
		return 0, "", false
	}
}

// HandleSlashCommand handles POST /api/slack/commands requests from Slack slash commands.
func (h *SlackInteractiveHandler) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("slack command: failed to read request body: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Slack request signature
	if !VerifySlackSignature(h.signingSecret, r, body) {
		logrus.Warn("slack command: invalid request signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the form-encoded slash command
	formData := parseFormValues(string(body))
	text := strings.TrimSpace(formData["text"])
	userID := formData["user_id"]

	// Route subcommands
	args := strings.Fields(text)
	var subcommand string
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "status":
		h.handleSlashStatus(w, r)
	case "silence":
		h.handleSlashSilence(w, r, userID, args[1:])
	case "unsilence":
		h.handleSlashUnsilence(w, r, args[1:])
	case "help", "":
		h.handleSlashHelp(w)
	default:
		respondEphemeral(w, fmt.Sprintf("Unknown command: `%s`. Type `/checker help` for available commands.", subcommand))
	}
}

// respondEphemeral sends an ephemeral JSON response to Slack.
func respondEphemeral(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response_type": "ephemeral",
		"text":          text,
	})
}

// handleSlashStatus handles `/checker status`.
func (h *SlackInteractiveHandler) handleSlashStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks, err := h.repo.GetUnhealthyChecks(ctx)
	if err != nil {
		logrus.Errorf("slack command: failed to get unhealthy checks: %s", err)
		respondEphemeral(w, "Failed to retrieve check status. Please try again.")
		return
	}

	if len(checks) == 0 {
		respondEphemeral(w, "All checks healthy :white_check_mark:")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d unhealthy check(s):*\n", len(checks)))
	for _, c := range checks {
		sb.WriteString(fmt.Sprintf("\xf0\x9f\x94\xb4 *%s* (%s/%s) \u2014 %s\n", c.Name, c.Project, c.GroupName, c.LastMessage))
	}

	respondEphemeral(w, sb.String())
}

// handleSlashSilence handles `/checker silence ...` subcommands.
func (h *SlackInteractiveHandler) handleSlashSilence(w http.ResponseWriter, r *http.Request, userID string, args []string) {
	if len(args) == 0 {
		respondEphemeral(w, "Usage: `/checker silence list`, `/checker silence check <uuid> <duration>`, or `/checker silence project <name> <duration>`\nDurations: `30m`, `1h`, `4h`, `8h`, `24h`, `indefinite`")
		return
	}

	switch args[0] {
	case "list":
		h.handleSlashSilenceList(w, r)
	case "check":
		if len(args) < 3 {
			respondEphemeral(w, "Usage: `/checker silence check <uuid> <duration>`\nDurations: `30m`, `1h`, `4h`, `8h`, `24h`, `indefinite`")
			return
		}
		h.handleSlashSilenceCreate(w, r, userID, "check", args[1], args[2])
	case "project":
		if len(args) < 3 {
			respondEphemeral(w, "Usage: `/checker silence project <name> <duration>`\nDurations: `30m`, `1h`, `4h`, `8h`, `24h`, `indefinite`")
			return
		}
		h.handleSlashSilenceCreate(w, r, userID, "project", args[1], args[2])
	default:
		respondEphemeral(w, fmt.Sprintf("Unknown silence subcommand: `%s`. Use `list`, `check`, or `project`.", args[0]))
	}
}

// handleSlashSilenceList handles `/checker silence list`.
func (h *SlackInteractiveHandler) handleSlashSilenceList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	silences, err := h.repo.GetActiveSilences(ctx)
	if err != nil {
		logrus.Errorf("slack command: failed to get active silences: %s", err)
		respondEphemeral(w, "Failed to retrieve silences. Please try again.")
		return
	}

	if len(silences) == 0 {
		respondEphemeral(w, "No active silences.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d active silence(s):*\n", len(silences)))
	for _, s := range silences {
		expires := "never"
		if s.ExpiresAt != nil {
			expires = s.ExpiresAt.Format("2006-01-02 15:04 UTC")
		}
		sb.WriteString(fmt.Sprintf("\xf0\x9f\x94\x87 %s *%s* \u2014 by <@%s>, expires %s\n", s.Scope, s.Target, s.SilencedBy, expires))
	}

	respondEphemeral(w, sb.String())
}

// handleSlashSilenceCreate creates a silence for a check or project.
func (h *SlackInteractiveHandler) handleSlashSilenceCreate(w http.ResponseWriter, r *http.Request, userID, scope, target, durationStr string) {
	duration, durationLabel, ok := parseSlashDuration(durationStr)
	if !ok {
		respondEphemeral(w, fmt.Sprintf("Invalid duration: `%s`. Valid values: `30m`, `1h`, `4h`, `8h`, `24h`, `indefinite`.", durationStr))
		return
	}

	silence := models.AlertSilence{
		Scope:      scope,
		Target:     target,
		SilencedBy: userID,
		Reason:     fmt.Sprintf("Silenced via /checker command for %s", durationLabel),
		Active:     true,
	}
	if durationStr != "indefinite" {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("slack command: failed to create silence: %s", err)
		respondEphemeral(w, "Failed to create silence. Please try again.")
		return
	}

	respondEphemeral(w, fmt.Sprintf("\xf0\x9f\x94\x87 Silenced %s `%s` for %s.", scope, target, durationLabel))
}

// handleSlashUnsilence handles `/checker unsilence ...` subcommands.
func (h *SlackInteractiveHandler) handleSlashUnsilence(w http.ResponseWriter, r *http.Request, args []string) {
	if len(args) < 2 {
		respondEphemeral(w, "Usage: `/checker unsilence check <uuid>` or `/checker unsilence project <name>`")
		return
	}

	scope := args[0]
	target := args[1]

	if scope != "check" && scope != "project" {
		respondEphemeral(w, fmt.Sprintf("Invalid scope: `%s`. Use `check` or `project`.", scope))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.DeactivateSilence(ctx, scope, target); err != nil {
		logrus.Errorf("slack command: failed to deactivate silence: %s", err)
		respondEphemeral(w, "Failed to remove silence. Please try again.")
		return
	}

	respondEphemeral(w, fmt.Sprintf("\xf0\x9f\x94\x8a Removed silence for %s `%s`.", scope, target))
}

// handleSlashHelp handles `/checker help` or `/checker` with no arguments.
func (h *SlackInteractiveHandler) handleSlashHelp(w http.ResponseWriter) {
	help := "*`/checker` commands:*\n" +
		"\u2022 `/checker status` \u2014 Show unhealthy checks\n" +
		"\u2022 `/checker silence list` \u2014 Show active silences\n" +
		"\u2022 `/checker silence check <uuid> <duration>` \u2014 Silence a check\n" +
		"\u2022 `/checker silence project <name> <duration>` \u2014 Silence a project\n" +
		"\u2022 `/checker unsilence check <uuid>` \u2014 Remove silence for a check\n" +
		"\u2022 `/checker unsilence project <name>` \u2014 Remove silence for a project\n" +
		"\u2022 `/checker help` \u2014 Show this message\n" +
		"\n_Durations: `30m`, `1h`, `4h`, `8h`, `24h`, `indefinite`_"

	respondEphemeral(w, help)
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
