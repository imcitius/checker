package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
	"github.com/imcitius/checker/internal/telegram"
)

// TelegramWebhookRegistrar registers the Telegram webhook route.
type TelegramWebhookRegistrar struct {
	Client *telegram.TelegramClient
}

func (r *TelegramWebhookRegistrar) RegisterRoutes(router *gin.Engine, repo db.Repository) {
	handler := NewTelegramWebhookHandler(r.Client.SecretToken(), r.Client, repo)
	router.POST("/api/telegram/webhook", gin.WrapF(handler.HandleWebhook))
	logrus.Info("Telegram webhook endpoint registered at /api/telegram/webhook")
}

// TelegramWebhookHandler handles Telegram webhook updates (callback queries and bot commands).
type TelegramWebhookHandler struct {
	secretToken    string
	telegramClient *telegram.TelegramClient
	repo           db.Repository
}

// NewTelegramWebhookHandler creates a new handler for Telegram webhook updates.
func NewTelegramWebhookHandler(secretToken string, client *telegram.TelegramClient, repo db.Repository) *TelegramWebhookHandler {
	return &TelegramWebhookHandler{
		secretToken:    secretToken,
		telegramClient: client,
		repo:           repo,
	}
}

// HandleWebhook handles POST /api/telegram/webhook requests.
// It verifies the secret token, parses the update, and routes to the appropriate handler.
// Always returns 200 OK to Telegram to prevent retries.
func (h *TelegramWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify secret token header
	token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if h.secretToken != "" && token != h.secretToken {
		logrus.Warn("telegram webhook: invalid secret token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read and parse body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("telegram webhook: failed to read body: %s", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	defer r.Body.Close()

	var update telegram.Update
	if err := json.Unmarshal(body, &update); err != nil {
		logrus.Errorf("telegram webhook: failed to parse update: %s", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route the update
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(w, r, update.CallbackQuery)
	} else if update.Message != nil && strings.HasPrefix(update.Message.Text, "/") {
		h.handleCommand(w, r, update.Message)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// handleCallbackQuery handles inline button presses from Telegram callback queries.
func (h *TelegramWebhookHandler) handleCallbackQuery(w http.ResponseWriter, r *http.Request, query *telegram.CallbackQuery) {
	defer func() { w.WriteHeader(http.StatusOK) }()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if query.Message == nil {
		logrus.Warn("telegram webhook: callback query has no message")
		return
	}

	chatID := strconv.FormatInt(query.Message.Chat.ID, 10)
	messageID := query.Message.MessageID
	userName := query.From.FirstName
	if query.From.Username != "" {
		userName = "@" + query.From.Username
	}

	// Look up thread to get check UUID
	thread, err := h.repo.GetTelegramThreadByMessage(ctx, chatID, messageID)
	if err != nil {
		logrus.Errorf("telegram webhook: failed to look up thread for chat=%s msg=%d: %s", chatID, messageID, err)
		h.answerCallback(ctx, query.ID, "Alert not found")
		return
	}

	checkUUID := thread.CheckUUID
	data := query.Data

	switch {
	case strings.HasPrefix(data, "s|"):
		// Silence check: s|<duration>
		durationStr := data[2:]
		h.handleSilenceCheckCallback(ctx, query, chatID, messageID, checkUUID, durationStr, userName)

	case strings.HasPrefix(data, "sp|"):
		// Silence project: sp|<duration>
		durationStr := data[3:]
		h.handleSilenceProjectCallback(ctx, query, chatID, messageID, checkUUID, durationStr, userName)

	case data == "ack":
		h.handleAckCallback(ctx, query, chatID, messageID, checkUUID, userName)

	default:
		logrus.Infof("telegram webhook: unknown callback data: %s", data)
		h.answerCallback(ctx, query.ID, "Unknown action")
	}
}

// handleSilenceCheckCallback creates a check-scoped silence and updates the original message.
func (h *TelegramWebhookHandler) handleSilenceCheckCallback(ctx context.Context, query *telegram.CallbackQuery, chatID string, messageID int, checkUUID, durationStr, userName string) {
	duration, durationLabel, indefinite := parseTelegramDuration(durationStr)

	silence := models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		SilencedBy: userName,
		Reason:     fmt.Sprintf("Silenced via Telegram for %s", durationLabel),
		Active:     true,
	}
	if !indefinite {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
	}

	if err := h.repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("telegram webhook: failed to create silence: %s", err)
		h.answerCallback(ctx, query.ID, "Failed to create silence")
		return
	}

	h.answerCallback(ctx, query.ID, "Silenced for "+durationLabel)

	// Edit original message to show silenced badge
	info := h.lookupCheckAlertInfo(ctx, checkUUID)
	silencedHTML := telegram.BuildSilencedAlertHTML(info)
	if err := h.telegramClient.EditMessageText(ctx, chatID, messageID, silencedHTML, "HTML", nil); err != nil {
		logrus.Errorf("telegram webhook: failed to edit message for silence badge: %s", err)
	}

	// Post silence confirmation reply
	if err := h.telegramClient.SendSilenceConfirmation(ctx, chatID, messageID, "check", checkUUID, durationLabel, userName); err != nil {
		logrus.Errorf("telegram webhook: failed to send silence confirmation: %s", err)
	}
}

// handleSilenceProjectCallback creates a project-scoped silence and updates the original message.
func (h *TelegramWebhookHandler) handleSilenceProjectCallback(ctx context.Context, query *telegram.CallbackQuery, chatID string, messageID int, checkUUID, durationStr, userName string) {
	// Look up project name from the check definition
	projectName := h.lookupProjectForCheck(ctx, checkUUID)
	if projectName == "" {
		h.answerCallback(ctx, query.ID, "Could not determine project")
		return
	}

	duration, durationLabel, indefinite := parseTelegramDuration(durationStr)

	silence := models.AlertSilence{
		Scope:      "project",
		Target:     projectName,
		SilencedBy: userName,
		Reason:     fmt.Sprintf("Silenced via Telegram for %s", durationLabel),
		Active:     true,
	}
	if !indefinite {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
	}

	if err := h.repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("telegram webhook: failed to create project silence: %s", err)
		h.answerCallback(ctx, query.ID, "Failed to create silence")
		return
	}

	h.answerCallback(ctx, query.ID, "Project silenced for "+durationLabel)

	// Edit original message to show silenced badge
	info := h.lookupCheckAlertInfo(ctx, checkUUID)
	silencedHTML := telegram.BuildSilencedAlertHTML(info)
	if err := h.telegramClient.EditMessageText(ctx, chatID, messageID, silencedHTML, "HTML", nil); err != nil {
		logrus.Errorf("telegram webhook: failed to edit message for project silence badge: %s", err)
	}

	// Post silence confirmation reply
	if err := h.telegramClient.SendSilenceConfirmation(ctx, chatID, messageID, "project", projectName, durationLabel, userName); err != nil {
		logrus.Errorf("telegram webhook: failed to send project silence confirmation: %s", err)
	}
}

// handleAckCallback posts an acknowledgment reply and updates the original message.
func (h *TelegramWebhookHandler) handleAckCallback(ctx context.Context, query *telegram.CallbackQuery, chatID string, messageID int, checkUUID, userName string) {
	h.answerCallback(ctx, query.ID, "Acknowledged")

	// Post ack reply
	ackText := fmt.Sprintf("👀 Acknowledged by %s", userName)
	_, err := h.telegramClient.SendMessage(ctx, chatID, ackText, "", nil, &messageID)
	if err != nil {
		logrus.Errorf("telegram webhook: failed to post ack reply: %s", err)
	}

	// Edit original message to show acknowledged badge
	info := h.lookupCheckAlertInfo(ctx, checkUUID)
	ackedHTML := telegram.BuildAcknowledgedAlertHTML(info)
	keyboard := telegram.BuildAcknowledgedAlertKeyboard(info)
	if err := h.telegramClient.EditMessageText(ctx, chatID, messageID, ackedHTML, "HTML", keyboard); err != nil {
		logrus.Errorf("telegram webhook: failed to edit message for ack badge: %s", err)
	}
}

// handleCommand handles bot commands (messages starting with /).
func (h *TelegramWebhookHandler) handleCommand(w http.ResponseWriter, r *http.Request, msg *telegram.IncomingMessage) {
	defer func() { w.WriteHeader(http.StatusOK) }()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	text := strings.TrimSpace(msg.Text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	// Strip the @botname suffix from the command if present (e.g., /status@mybot)
	command := parts[0]
	if idx := strings.Index(command, "@"); idx > 0 {
		command = command[:idx]
	}

	switch command {
	case "/status":
		h.handleStatusCommand(ctx, chatID, msg.MessageID)
	case "/silence":
		h.handleSilenceCommand(ctx, chatID, msg.MessageID, parts[1:])
	case "/unsilence":
		h.handleUnsilenceCommand(ctx, chatID, msg.MessageID, parts[1:])
	case "/help", "/start":
		h.handleHelpCommand(ctx, chatID, msg.MessageID)
	default:
		// Ignore unknown commands
	}
}

// handleStatusCommand lists unhealthy checks.
func (h *TelegramWebhookHandler) handleStatusCommand(ctx context.Context, chatID string, replyTo int) {
	checks, err := h.repo.GetUnhealthyChecks(ctx)
	if err != nil {
		logrus.Errorf("telegram command: failed to get unhealthy checks: %s", err)
		h.sendReply(ctx, chatID, replyTo, "Failed to retrieve check status. Please try again.")
		return
	}

	if len(checks) == 0 {
		h.sendReply(ctx, chatID, replyTo, "✅ All checks healthy")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%d unhealthy check(s):</b>\n\n", len(checks)))
	for _, c := range checks {
		sb.WriteString(fmt.Sprintf("🔴 <b>%s</b> (%s/%s)\n<pre>%s</pre>\n\n", c.Name, c.Project, c.GroupName, c.LastMessage))
	}

	h.sendReply(ctx, chatID, replyTo, sb.String())
}

// handleSilenceCommand handles /silence subcommands.
func (h *TelegramWebhookHandler) handleSilenceCommand(ctx context.Context, chatID string, replyTo int, args []string) {
	if len(args) == 0 {
		h.sendReply(ctx, chatID, replyTo,
			"<b>Usage:</b>\n"+
				"<code>/silence list</code> — List active silences\n"+
				"<code>/silence check &lt;uuid&gt; &lt;duration&gt;</code> — Silence a check\n"+
				"<code>/silence project &lt;name&gt; &lt;duration&gt;</code> — Silence a project\n\n"+
				"<i>Durations: 30m, 1h, 4h, 8h, 24h, indefinite</i>")
		return
	}

	switch args[0] {
	case "list":
		h.handleSilenceListCommand(ctx, chatID, replyTo)
	case "check":
		if len(args) < 3 {
			h.sendReply(ctx, chatID, replyTo, "Usage: <code>/silence check &lt;uuid&gt; &lt;duration&gt;</code>\nDurations: 30m, 1h, 4h, 8h, 24h, indefinite")
			return
		}
		h.handleSilenceCreateCommand(ctx, chatID, replyTo, "check", args[1], args[2])
	case "project":
		if len(args) < 3 {
			h.sendReply(ctx, chatID, replyTo, "Usage: <code>/silence project &lt;name&gt; &lt;duration&gt;</code>\nDurations: 30m, 1h, 4h, 8h, 24h, indefinite")
			return
		}
		h.handleSilenceCreateCommand(ctx, chatID, replyTo, "project", args[1], args[2])
	default:
		h.sendReply(ctx, chatID, replyTo, fmt.Sprintf("Unknown silence subcommand: <code>%s</code>. Use <code>list</code>, <code>check</code>, or <code>project</code>.", args[0]))
	}
}

// handleSilenceListCommand lists active silences.
func (h *TelegramWebhookHandler) handleSilenceListCommand(ctx context.Context, chatID string, replyTo int) {
	silences, err := h.repo.GetActiveSilences(ctx)
	if err != nil {
		logrus.Errorf("telegram command: failed to get active silences: %s", err)
		h.sendReply(ctx, chatID, replyTo, "Failed to retrieve silences. Please try again.")
		return
	}

	if len(silences) == 0 {
		h.sendReply(ctx, chatID, replyTo, "No active silences.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%d active silence(s):</b>\n\n", len(silences)))
	for _, s := range silences {
		expires := "never"
		if s.ExpiresAt != nil {
			expires = s.ExpiresAt.Format("2006-01-02 15:04 UTC")
		}
		sb.WriteString(fmt.Sprintf("🔇 %s <b>%s</b> — by %s, expires %s\n", s.Scope, s.Target, s.SilencedBy, expires))
	}

	h.sendReply(ctx, chatID, replyTo, sb.String())
}

// handleSilenceCreateCommand creates a silence for a check or project.
func (h *TelegramWebhookHandler) handleSilenceCreateCommand(ctx context.Context, chatID string, replyTo int, scope, target, durationStr string) {
	duration, durationLabel, ok := parseSlashDuration(durationStr)
	if !ok {
		h.sendReply(ctx, chatID, replyTo, fmt.Sprintf("Invalid duration: <code>%s</code>. Valid values: 30m, 1h, 4h, 8h, 24h, indefinite.", durationStr))
		return
	}

	silence := models.AlertSilence{
		Scope:      scope,
		Target:     target,
		SilencedBy: "telegram-user",
		Reason:     fmt.Sprintf("Silenced via Telegram /silence command for %s", durationLabel),
		Active:     true,
	}
	if durationStr != "indefinite" {
		expiresAt := time.Now().Add(duration)
		silence.ExpiresAt = &expiresAt
	}

	if err := h.repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("telegram command: failed to create silence: %s", err)
		h.sendReply(ctx, chatID, replyTo, "Failed to create silence. Please try again.")
		return
	}

	h.sendReply(ctx, chatID, replyTo, fmt.Sprintf("🔇 Silenced %s <code>%s</code> for %s.", scope, target, durationLabel))
}

// handleUnsilenceCommand handles /unsilence subcommands.
func (h *TelegramWebhookHandler) handleUnsilenceCommand(ctx context.Context, chatID string, replyTo int, args []string) {
	if len(args) < 2 {
		h.sendReply(ctx, chatID, replyTo, "Usage: <code>/unsilence check &lt;uuid&gt;</code> or <code>/unsilence project &lt;name&gt;</code>")
		return
	}

	scope := args[0]
	target := args[1]

	if scope != "check" && scope != "project" {
		h.sendReply(ctx, chatID, replyTo, fmt.Sprintf("Invalid scope: <code>%s</code>. Use <code>check</code> or <code>project</code>.", scope))
		return
	}

	if err := h.repo.DeactivateSilence(ctx, scope, target); err != nil {
		logrus.Errorf("telegram command: failed to deactivate silence: %s", err)
		h.sendReply(ctx, chatID, replyTo, "Failed to remove silence. Please try again.")
		return
	}

	h.sendReply(ctx, chatID, replyTo, fmt.Sprintf("🔊 Removed silence for %s <code>%s</code>.", scope, target))
}

// handleHelpCommand sends a formatted help message.
func (h *TelegramWebhookHandler) handleHelpCommand(ctx context.Context, chatID string, replyTo int) {
	help := "<b>Available commands:</b>\n\n" +
		"• /status — Show unhealthy checks\n" +
		"• /silence list — Show active silences\n" +
		"• /silence check &lt;uuid&gt; &lt;duration&gt; — Silence a check\n" +
		"• /silence project &lt;name&gt; &lt;duration&gt; — Silence a project\n" +
		"• /unsilence check &lt;uuid&gt; — Remove silence for a check\n" +
		"• /unsilence project &lt;name&gt; — Remove silence for a project\n" +
		"• /help — Show this message\n\n" +
		"<i>Durations: 30m, 1h, 4h, 8h, 24h, indefinite</i>"

	h.sendReply(ctx, chatID, replyTo, help)
}

// --- Helper methods ---

// answerCallback answers a callback query with a text notification.
func (h *TelegramWebhookHandler) answerCallback(ctx context.Context, callbackQueryID, text string) {
	if h.telegramClient == nil {
		return
	}
	if err := h.telegramClient.AnswerCallbackQuery(ctx, callbackQueryID, text, false); err != nil {
		logrus.Errorf("telegram webhook: failed to answer callback query: %s", err)
	}
}

// sendReply sends an HTML reply to a message.
func (h *TelegramWebhookHandler) sendReply(ctx context.Context, chatID string, replyToMsgID int, text string) {
	if h.telegramClient == nil {
		return
	}
	if _, err := h.telegramClient.SendMessage(ctx, chatID, text, "HTML", nil, &replyToMsgID); err != nil {
		logrus.Errorf("telegram webhook: failed to send reply: %s", err)
	}
}

// lookupCheckAlertInfo fetches check definition details for building alert HTML.
func (h *TelegramWebhookHandler) lookupCheckAlertInfo(ctx context.Context, checkUUID string) telegram.CheckAlertInfo {
	info := telegram.CheckAlertInfo{UUID: checkUUID}

	checkDef, err := h.repo.GetCheckDefinitionByUUID(ctx, checkUUID)
	if err != nil {
		logrus.Errorf("telegram webhook: failed to look up check %s: %s", checkUUID, err)
		return info
	}

	info.Name = checkDef.Name
	info.Project = checkDef.Project
	info.Group = checkDef.GroupName
	info.CheckType = checkDef.Type
	info.Frequency = checkDef.Duration
	info.Message = checkDef.LastMessage
	info.Severity = checkDef.Severity
	info.IsHealthy = checkDef.IsHealthy

	return info
}

// lookupProjectForCheck fetches the project name for a check UUID.
func (h *TelegramWebhookHandler) lookupProjectForCheck(ctx context.Context, checkUUID string) string {
	checkDef, err := h.repo.GetCheckDefinitionByUUID(ctx, checkUUID)
	if err != nil {
		logrus.Errorf("telegram webhook: failed to look up check %s for project: %s", checkUUID, err)
		return ""
	}
	return checkDef.Project
}

// parseTelegramDuration parses a duration string from a Telegram callback query.
func parseTelegramDuration(s string) (duration time.Duration, label string, indefinite bool) {
	switch s {
	case "30m":
		return 30 * time.Minute, "30m", false
	case "1h":
		return 1 * time.Hour, "1h", false
	case "4h":
		return 4 * time.Hour, "4h", false
	case "8h":
		return 8 * time.Hour, "8h", false
	case "24h":
		return 24 * time.Hour, "24h", false
	case "indef", "indefinite":
		return 0, "indefinitely", true
	default:
		return 1 * time.Hour, "1h", false
	}
}
