package web

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"checker/internal/db"
	"checker/internal/discord"
	"checker/internal/models"
)

// DiscordWebhookRegistrar registers Discord interaction routes.
type DiscordWebhookRegistrar struct {
	Client    *discord.DiscordClient
	PublicKey string // Discord app public key for Ed25519 signature verification
}

// RegisterRoutes implements the WebhookRegistrar interface.
func (d *DiscordWebhookRegistrar) RegisterRoutes(router *gin.Engine, repo db.Repository) {
	handler := NewDiscordInteractionHandler(d.PublicKey, d.Client, repo)
	router.POST("/api/discord/interactions", gin.WrapF(handler.HandleInteraction))
	logrus.Info("Discord interaction endpoint registered at /api/discord/interactions")
}

// DiscordInteractionHandler handles Discord interaction payloads (button clicks).
type DiscordInteractionHandler struct {
	publicKey     ed25519.PublicKey
	discordClient *discord.DiscordClient
	repo          db.Repository
}

// NewDiscordInteractionHandler creates a new handler for Discord interactions.
func NewDiscordInteractionHandler(publicKeyHex string, client *discord.DiscordClient, repo db.Repository) *DiscordInteractionHandler {
	var pubKey ed25519.PublicKey
	if publicKeyHex != "" {
		decoded, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			logrus.Errorf("discord interactions: invalid public key hex: %s", err)
		} else {
			pubKey = ed25519.PublicKey(decoded)
		}
	}
	return &DiscordInteractionHandler{
		publicKey:     pubKey,
		discordClient: client,
		repo:          repo,
	}
}

// Discord interaction types.
const (
	interactionTypePing             = 1
	interactionTypeMessageComponent = 3
)

// Discord interaction response types.
const (
	interactionCallbackPong          = 1
	interactionCallbackUpdateMessage = 7
)

// DiscordInteraction represents the incoming interaction payload from Discord.
type DiscordInteraction struct {
	Type    int                    `json:"type"`
	ID      string                 `json:"id"`
	Token   string                 `json:"token"`
	Data    *DiscordInteractionData `json:"data,omitempty"`
	Member  *DiscordMember         `json:"member,omitempty"`
	User    *DiscordUser           `json:"user,omitempty"`
	Message *DiscordInteractionMessage `json:"message,omitempty"`
}

// DiscordInteractionData contains the data for message component interactions.
type DiscordInteractionData struct {
	CustomID      string `json:"custom_id"`
	ComponentType int    `json:"component_type"`
}

// DiscordMember represents a guild member who triggered the interaction.
type DiscordMember struct {
	User *DiscordUser `json:"user,omitempty"`
	Nick string       `json:"nick,omitempty"`
}

// DiscordUser represents a Discord user.
type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// DiscordInteractionMessage represents the message the interaction is attached to.
type DiscordInteractionMessage struct {
	ID        string          `json:"id"`
	ChannelID string          `json:"channel_id"`
	Embeds    []discord.Embed `json:"embeds,omitempty"`
}

// HandleInteraction handles POST /api/discord/interactions requests.
func (h *DiscordInteractionHandler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("discord interactions: failed to read request body: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Ed25519 signature
	if !VerifyDiscordSignature(h.publicKey, r, body) {
		logrus.Warn("discord interactions: invalid request signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the interaction
	var interaction DiscordInteraction
	if err := json.Unmarshal(body, &interaction); err != nil {
		logrus.Errorf("discord interactions: failed to parse payload: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	switch interaction.Type {
	case interactionTypePing:
		// Discord PING verification — must respond with PONG
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"type": interactionCallbackPong})
		return

	case interactionTypeMessageComponent:
		// Must respond within 3 seconds
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		h.handleMessageComponent(ctx, w, interaction)
		return

	default:
		logrus.Infof("discord interactions: unhandled interaction type: %d", interaction.Type)
		w.WriteHeader(http.StatusOK)
	}
}

// handleMessageComponent routes button clicks based on custom_id.
func (h *DiscordInteractionHandler) handleMessageComponent(ctx context.Context, w http.ResponseWriter, interaction DiscordInteraction) {
	if interaction.Data == nil {
		logrus.Warn("discord interactions: missing data in message component interaction")
		w.WriteHeader(http.StatusOK)
		return
	}

	customID := interaction.Data.CustomID
	username := getInteractionUsername(interaction)

	switch {
	case strings.HasPrefix(customID, "checker_ack_"):
		checkUUID := strings.TrimPrefix(customID, "checker_ack_")
		h.handleAcknowledge(ctx, w, interaction, checkUUID, username)

	case strings.HasPrefix(customID, "checker_silence_"):
		h.handleSilence(ctx, w, interaction, customID, username)

	case strings.HasPrefix(customID, "checker_unsilence_"):
		checkUUID := strings.TrimPrefix(customID, "checker_unsilence_")
		h.handleUnsilence(ctx, w, interaction, checkUUID, username)

	default:
		logrus.Infof("discord interactions: unknown custom_id: %s", customID)
		w.WriteHeader(http.StatusOK)
	}
}

// handleAcknowledge handles the Acknowledge button click.
func (h *DiscordInteractionHandler) handleAcknowledge(ctx context.Context, w http.ResponseWriter, interaction DiscordInteraction, checkUUID, username string) {
	// Build updated embed showing "Acknowledged"
	embeds := buildAcknowledgedEmbeds(interaction, username)
	// Keep original buttons (acknowledge can still silence)
	components := buildAckButtons(checkUUID)

	// Respond with UPDATE_MESSAGE to update the original message
	resp := discord.InteractionResponse{
		Type: interactionCallbackUpdateMessage,
		Data: &discord.InteractionCallbackData{
			Embeds:     embeds,
			Components: components,
		},
	}

	respondJSON(w, resp)

	// Post thread reply asynchronously (best-effort, don't block the response)
	if h.discordClient != nil && interaction.Message != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Find the thread for this message
			thread, err := h.repo.GetUnresolvedDiscordThread(bgCtx, checkUUID)
			if err != nil {
				logrus.Debugf("discord interactions: no thread found for ack reply (check %s): %s", checkUUID, err)
				return
			}

			replyPayload := discord.MessagePayload{
				Content: fmt.Sprintf("👀 **%s** acknowledged this alert", username),
			}
			if _, err := h.discordClient.SendThreadReply(bgCtx, thread.ThreadID, replyPayload); err != nil {
				logrus.Errorf("discord interactions: failed to send ack thread reply: %s", err)
			}
		}()
	}
}

// handleSilence handles the Silence button clicks.
func (h *DiscordInteractionHandler) handleSilence(ctx context.Context, w http.ResponseWriter, interaction DiscordInteraction, customID, username string) {
	// Parse custom_id: "checker_silence_{checkUUID}_{duration}"
	remainder := strings.TrimPrefix(customID, "checker_silence_")
	checkUUID, duration, durationLabel := parseDiscordSilenceID(remainder)

	if checkUUID == "" {
		logrus.Errorf("discord interactions: invalid silence custom_id: %s", customID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Create silence in DB
	silence := models.AlertSilence{
		Scope:      "check",
		Target:     checkUUID,
		Channel:    "discord",
		SilencedBy: username,
		Reason:     fmt.Sprintf("Silenced via Discord for %s", durationLabel),
		Active:     true,
	}
	expiresAt := time.Now().Add(duration)
	silence.ExpiresAt = &expiresAt

	if err := h.repo.CreateSilence(ctx, silence); err != nil {
		logrus.Errorf("discord interactions: failed to create silence: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build updated embed showing "Silenced"
	embeds := buildSilencedEmbeds(interaction, username, durationLabel)
	// Replace buttons with Unsilence
	components := []discord.ActionRow{
		{
			Type: discord.ComponentTypeActionRow,
			Components: []discord.Component{
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Unsilence",
					Style:    discord.ButtonStyleDanger,
					CustomID: fmt.Sprintf("checker_unsilence_%s", checkUUID),
				},
			},
		},
	}

	resp := discord.InteractionResponse{
		Type: interactionCallbackUpdateMessage,
		Data: &discord.InteractionCallbackData{
			Embeds:     embeds,
			Components: components,
		},
	}

	respondJSON(w, resp)
}

// handleUnsilence handles the Unsilence button click.
func (h *DiscordInteractionHandler) handleUnsilence(ctx context.Context, w http.ResponseWriter, interaction DiscordInteraction, checkUUID, username string) {
	// Deactivate silence in DB
	if err := h.repo.DeactivateSilence(ctx, "check", checkUUID); err != nil {
		logrus.Errorf("discord interactions: failed to deactivate silence: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Restore original alert embeds and buttons
	embeds := buildUnsiliencedEmbeds(interaction, username)
	components := buildAlertButtons(checkUUID)

	resp := discord.InteractionResponse{
		Type: interactionCallbackUpdateMessage,
		Data: &discord.InteractionCallbackData{
			Embeds:     embeds,
			Components: components,
		},
	}

	respondJSON(w, resp)
}

// VerifyDiscordSignature verifies the Ed25519 signature from Discord.
// Discord sends the signature as hex in X-Signature-Ed25519, and the timestamp
// in X-Signature-Timestamp. The signed content is timestamp + body.
func VerifyDiscordSignature(publicKey ed25519.PublicKey, r *http.Request, body []byte) bool {
	if len(publicKey) == 0 {
		return false
	}

	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")

	if signature == "" || timestamp == "" {
		return false
	}

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	// Discord signs: timestamp + body
	msg := []byte(timestamp + string(body))
	return ed25519.Verify(publicKey, msg, sigBytes)
}

// getInteractionUsername extracts a display name from the interaction.
func getInteractionUsername(interaction DiscordInteraction) string {
	if interaction.Member != nil {
		if interaction.Member.Nick != "" {
			return interaction.Member.Nick
		}
		if interaction.Member.User != nil {
			return interaction.Member.User.Username
		}
	}
	if interaction.User != nil {
		return interaction.User.Username
	}
	return "Unknown"
}

// parseDiscordSilenceID parses "{checkUUID}_{duration}" from the remainder after
// stripping "checker_silence_". UUIDs contain hyphens, so we split from the right.
func parseDiscordSilenceID(remainder string) (checkUUID string, duration time.Duration, label string) {
	// The duration is the last segment: "1h" or "24h"
	lastUnderscore := strings.LastIndex(remainder, "_")
	if lastUnderscore < 0 {
		return "", 0, ""
	}

	checkUUID = remainder[:lastUnderscore]
	durationStr := remainder[lastUnderscore+1:]

	switch durationStr {
	case "1h":
		return checkUUID, 1 * time.Hour, "1h"
	case "24h":
		return checkUUID, 24 * time.Hour, "24h"
	default:
		return checkUUID, 1 * time.Hour, durationStr
	}
}

// respondJSON writes a JSON response to the HTTP writer.
func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logrus.Errorf("discord interactions: failed to encode response: %s", err)
	}
}

// buildAcknowledgedEmbeds takes the original message embeds and adds an "Acknowledged" field.
func buildAcknowledgedEmbeds(interaction DiscordInteraction, username string) []discord.Embed {
	embeds := getOriginalEmbeds(interaction)
	if len(embeds) > 0 {
		embeds[0].Fields = appendOrReplaceField(embeds[0].Fields, discord.EmbedField{
			Name:   "Acknowledged",
			Value:  fmt.Sprintf("👀 %s", username),
			Inline: true,
		})
	}
	return embeds
}

// buildSilencedEmbeds takes the original message embeds and adds a "Silenced" field.
func buildSilencedEmbeds(interaction DiscordInteraction, username, duration string) []discord.Embed {
	embeds := getOriginalEmbeds(interaction)
	if len(embeds) > 0 {
		embeds[0].Color = discord.ColorGray
		embeds[0].Fields = appendOrReplaceField(embeds[0].Fields, discord.EmbedField{
			Name:   "Silenced",
			Value:  fmt.Sprintf("🔇 %s for %s", username, duration),
			Inline: true,
		})
	}
	return embeds
}

// buildUnsiliencedEmbeds takes the original message embeds and removes the "Silenced" field,
// restoring the alert color.
func buildUnsiliencedEmbeds(interaction DiscordInteraction, username string) []discord.Embed {
	embeds := getOriginalEmbeds(interaction)
	if len(embeds) > 0 {
		embeds[0].Color = discord.ColorRed
		// Remove "Silenced" field
		filtered := make([]discord.EmbedField, 0, len(embeds[0].Fields))
		for _, f := range embeds[0].Fields {
			if f.Name != "Silenced" {
				filtered = append(filtered, f)
			}
		}
		embeds[0].Fields = filtered
	}
	return embeds
}

// getOriginalEmbeds returns the embeds from the interaction's original message.
// Returns a copy so modifications don't affect the original.
func getOriginalEmbeds(interaction DiscordInteraction) []discord.Embed {
	if interaction.Message == nil || len(interaction.Message.Embeds) == 0 {
		return []discord.Embed{{Title: "Alert", Color: discord.ColorRed}}
	}
	// Deep copy
	embeds := make([]discord.Embed, len(interaction.Message.Embeds))
	for i, e := range interaction.Message.Embeds {
		embeds[i] = e
		embeds[i].Fields = make([]discord.EmbedField, len(e.Fields))
		copy(embeds[i].Fields, e.Fields)
	}
	return embeds
}

// appendOrReplaceField adds or replaces a field with the given name in the embed fields.
func appendOrReplaceField(fields []discord.EmbedField, field discord.EmbedField) []discord.EmbedField {
	for i, f := range fields {
		if f.Name == field.Name {
			fields[i] = field
			return fields
		}
	}
	return append(fields, field)
}

// buildAlertButtons returns the standard alert action buttons.
func buildAlertButtons(checkUUID string) []discord.ActionRow {
	return []discord.ActionRow{
		{
			Type: discord.ComponentTypeActionRow,
			Components: []discord.Component{
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Acknowledge",
					Style:    discord.ButtonStylePrimary,
					CustomID: fmt.Sprintf("checker_ack_%s", checkUUID),
				},
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Silence 1h",
					Style:    discord.ButtonStyleSecondary,
					CustomID: fmt.Sprintf("checker_silence_%s_1h", checkUUID),
				},
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Silence 24h",
					Style:    discord.ButtonStyleSecondary,
					CustomID: fmt.Sprintf("checker_silence_%s_24h", checkUUID),
				},
			},
		},
	}
}

// buildAckButtons returns buttons after acknowledgment (still allows silence).
func buildAckButtons(checkUUID string) []discord.ActionRow {
	return []discord.ActionRow{
		{
			Type: discord.ComponentTypeActionRow,
			Components: []discord.Component{
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Silence 1h",
					Style:    discord.ButtonStyleSecondary,
					CustomID: fmt.Sprintf("checker_silence_%s_1h", checkUUID),
				},
				{
					Type:     discord.ComponentTypeButton,
					Label:    "Silence 24h",
					Style:    discord.ButtonStyleSecondary,
					CustomID: fmt.Sprintf("checker_silence_%s_24h", checkUUID),
				},
			},
		},
	}
}
