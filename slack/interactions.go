package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"my/checker/config"
	"my/checker/status"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Silence represents an active alert silence rule.
type Silence struct {
	ID        string     `json:"id"`
	Scope     string     `json:"scope"`      // "all", "check", "project"
	Target    string     `json:"target"`      // check name/UUID or project name (empty for scope "all")
	CreatedBy string     `json:"created_by"`  // Slack user ID
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`  // nil means indefinite
}

// InteractionHandler handles Slack slash command interactions.
type InteractionHandler struct {
	signingSecret string

	mu       sync.RWMutex
	silences map[string]*Silence
}

// NewInteractionHandler creates a new handler for Slack interactions.
func NewInteractionHandler(signingSecret string) *InteractionHandler {
	return &InteractionHandler{
		signingSecret: signingSecret,
		silences:      make(map[string]*Silence),
	}
}

// slackResponse is the JSON response format for Slack slash commands.
type slackResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

// HandleSlashCommand handles incoming Slack slash command requests.
func (h *InteractionHandler) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		config.Log.Errorf("slack: failed to read request body: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Slack signature
	if !h.verifySlackSignature(r, body) {
		config.Log.Warn("slack: invalid request signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data from body
	// Slack sends application/x-www-form-urlencoded
	params, err := parseFormBody(body)
	if err != nil {
		config.Log.Errorf("slack: failed to parse form data: %s", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(params["text"])
	userID := params["user_id"]

	// Expire old silences before processing
	h.expireSilences()

	// Route based on command text
	var response string
	parts := strings.Fields(text)

	if len(parts) == 0 {
		response = h.helpText()
	} else {
		switch parts[0] {
		case "all":
			duration := ""
			if len(parts) > 1 {
				duration = parts[1]
			}
			response = h.silenceAll(userID, duration)
		case "check":
			if len(parts) < 2 {
				response = "Usage: /checker-silence check <name-or-uuid>"
			} else {
				response = h.silenceCheck(userID, parts[1])
			}
		case "project":
			if len(parts) < 2 {
				response = "Usage: /checker-silence project <project-name>"
			} else {
				response = h.silenceProject(userID, parts[1])
			}
		case "list":
			response = h.listSilences()
		case "clear":
			if len(parts) > 1 {
				response = h.clearSilence(parts[1])
			} else {
				response = h.clearAllSilences()
			}
		case "help":
			response = h.helpText()
		default:
			response = fmt.Sprintf("Unknown command: %s\n\n%s", parts[0], h.helpText())
		}
	}

	h.respondEphemeral(w, response)
}

// verifySlackSignature verifies the Slack request signature using HMAC-SHA256.
func (h *InteractionHandler) verifySlackSignature(r *http.Request, body []byte) bool {
	if h.signingSecret == "" {
		config.Log.Warn("slack: signing secret not configured, skipping verification")
		return true
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp is within 5 minutes to prevent replay attacks
	// Slack sends timestamp as Unix epoch seconds
	var tsInt int64
	if _, err := fmt.Sscanf(timestamp, "%d", &tsInt); err != nil {
		config.Log.Errorf("slack: invalid timestamp: %s", timestamp)
		return false
	}
	ts := time.Unix(tsInt, 0)
	if math.Abs(time.Since(ts).Minutes()) > 5 {
		config.Log.Warn("slack: request timestamp too old")
		return false
	}

	// Compute HMAC-SHA256 of "v0:{timestamp}:{body}"
	sigBasestring := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	mac.Write([]byte(sigBasestring))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// parseFormBody parses application/x-www-form-urlencoded body into a map.
func parseFormBody(body []byte) (map[string]string, error) {
	result := make(map[string]string)
	pairs := strings.Split(string(body), "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := queryUnescape(kv[0])
			value := queryUnescape(kv[1])
			result[key] = value
		}
	}
	return result, nil
}

// queryUnescape performs basic URL decoding.
func queryUnescape(s string) string {
	s = strings.ReplaceAll(s, "+", " ")
	// Simple percent-decoding for common characters
	result := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			b, err := hex.DecodeString(s[i+1 : i+3])
			if err == nil {
				result.Write(b)
				i += 2
				continue
			}
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

// silenceAll silences all alerts globally.
func (h *InteractionHandler) silenceAll(userID, durationStr string) string {
	var expiresAt *time.Time
	if durationStr != "" {
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Sprintf("Invalid duration: %s. Use formats like 1h, 30m, 2h30m", durationStr)
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	// Set global silence
	status.MainStatus = "quiet"

	silence := h.addSilence("all", "", userID, expiresAt)

	expiry := "indefinitely"
	if expiresAt != nil {
		expiry = fmt.Sprintf("until %s", expiresAt.Format(time.RFC822))
	}
	return fmt.Sprintf("All alerts silenced %s (ID: %s)", expiry, silence.ID)
}

// silenceCheck silences a specific check by name or UUID.
func (h *InteractionHandler) silenceCheck(userID, nameOrUUID string) string {
	// Try to find check by UUID first
	check := config.GetCheckByUUID(nameOrUUID)
	if check != nil {
		err := status.SetCheckMode(check, "quiet")
		if err != nil {
			return fmt.Sprintf("Error silencing check: %s", err)
		}
		silence := h.addSilence("check", nameOrUUID, userID, nil)
		return fmt.Sprintf("Check %s (%s) silenced (ID: %s)", check.Name, check.UUid, silence.ID)
	}

	// Try to find by name
	for _, project := range config.Config.Projects {
		for _, hc := range project.Healthchecks {
			for _, c := range hc.Checks {
				if strings.EqualFold(c.Name, nameOrUUID) {
					err := status.SetCheckMode(&c, "quiet")
					if err != nil {
						return fmt.Sprintf("Error silencing check: %s", err)
					}
					silence := h.addSilence("check", c.UUid, userID, nil)
					return fmt.Sprintf("Check %s (%s) silenced (ID: %s)", c.Name, c.UUid, silence.ID)
				}
			}
		}
	}

	return fmt.Sprintf("Check not found: %s", nameOrUUID)
}

// silenceProject silences all checks in a project.
func (h *InteractionHandler) silenceProject(userID, projectName string) string {
	for _, project := range config.Config.Projects {
		if strings.EqualFold(project.Name, projectName) {
			// Set project mode to quiet
			if ps, ok := status.Statuses.Projects[project.Name]; ok {
				ps.Mode = "quiet"
			}

			// Also silence individual checks in the project
			for _, hc := range project.Healthchecks {
				for _, c := range hc.Checks {
					_ = status.SetCheckMode(&c, "quiet")
				}
			}

			silence := h.addSilence("project", project.Name, userID, nil)
			return fmt.Sprintf("Project %s silenced (ID: %s)", project.Name, silence.ID)
		}
	}

	return fmt.Sprintf("Project not found: %s", projectName)
}

// listSilences returns a formatted list of all active silences.
func (h *InteractionHandler) listSilences() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.silences) == 0 {
		return "No active silences."
	}

	var sb strings.Builder
	sb.WriteString("*Active Silences:*\n\n")

	for _, s := range h.silences {
		sb.WriteString(fmt.Sprintf("• *ID:* `%s`\n", s.ID))
		sb.WriteString(fmt.Sprintf("  *Scope:* %s\n", s.Scope))
		if s.Target != "" {
			sb.WriteString(fmt.Sprintf("  *Target:* %s\n", s.Target))
		}
		sb.WriteString(fmt.Sprintf("  *Created by:* <@%s>\n", s.CreatedBy))
		sb.WriteString(fmt.Sprintf("  *Created at:* %s\n", s.CreatedAt.Format(time.RFC822)))
		if s.ExpiresAt != nil {
			sb.WriteString(fmt.Sprintf("  *Expires at:* %s\n", s.ExpiresAt.Format(time.RFC822)))
		} else {
			sb.WriteString("  *Expires at:* never\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// clearSilence removes a specific silence by ID.
func (h *InteractionHandler) clearSilence(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	silence, ok := h.silences[id]
	if !ok {
		return fmt.Sprintf("Silence not found: %s", id)
	}

	h.removeSilenceEffects(silence)
	delete(h.silences, id)

	return fmt.Sprintf("Silence %s cleared.", id)
}

// clearAllSilences removes all active silences and restores alert state.
func (h *InteractionHandler) clearAllSilences() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.silences) == 0 {
		return "No active silences to clear."
	}

	count := len(h.silences)
	for _, s := range h.silences {
		h.removeSilenceEffects(s)
	}
	h.silences = make(map[string]*Silence)

	return fmt.Sprintf("Cleared %d silence(s). All alerts restored.", count)
}

// removeSilenceEffects restores alert state for a silence.
func (h *InteractionHandler) removeSilenceEffects(s *Silence) {
	switch s.Scope {
	case "all":
		status.MainStatus = "loud"
	case "check":
		check := config.GetCheckByUUID(s.Target)
		if check != nil {
			_ = status.SetCheckMode(check, "loud")
		}
	case "project":
		if ps, ok := status.Statuses.Projects[s.Target]; ok {
			ps.Mode = "loud"
		}
		// Also restore individual checks
		for _, project := range config.Config.Projects {
			if project.Name == s.Target {
				for _, hc := range project.Healthchecks {
					for _, c := range hc.Checks {
						_ = status.SetCheckMode(&c, "loud")
					}
				}
				break
			}
		}
	}
}

// addSilence creates and stores a new silence.
func (h *InteractionHandler) addSilence(scope, target, userID string, expiresAt *time.Time) *Silence {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := config.GetRandomId()
	s := &Silence{
		ID:        id,
		Scope:     scope,
		Target:    target,
		CreatedBy: userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	h.silences[id] = s
	return s
}

// expireSilences removes silences that have passed their expiration time
// and restores alert state.
func (h *InteractionHandler) expireSilences() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, s := range h.silences {
		if s.ExpiresAt != nil && now.After(*s.ExpiresAt) {
			h.removeSilenceEffects(s)
			delete(h.silences, id)
			config.Log.Infof("slack: silence %s expired (scope: %s, target: %s)", id, s.Scope, s.Target)
		}
	}
}

// respondEphemeral writes an ephemeral JSON response back to Slack.
func (h *InteractionHandler) respondEphemeral(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	resp := slackResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		config.Log.Errorf("slack: failed to encode response: %s", err)
	}
}

// helpText returns the usage help message.
func (h *InteractionHandler) helpText() string {
	return "*Checker Silence Management*\n\n" +
		"Usage:\n" +
		"• `/checker-silence all [duration]` - Silence all alerts (e.g., 1h, 30m). Default: indefinite\n" +
		"• `/checker-silence check <name-or-uuid>` - Silence a specific check\n" +
		"• `/checker-silence project <project-name>` - Silence all checks in a project\n" +
		"• `/checker-silence list` - List all active silences\n" +
		"• `/checker-silence clear` - Remove all active silences\n" +
		"• `/checker-silence clear <id>` - Remove a specific silence by ID\n" +
		"• `/checker-silence help` - Show this help message"
}
