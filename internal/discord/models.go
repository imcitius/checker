package discord

// Discord API base URL.
const baseURL = "https://discord.com/api/v10"

// Discord embed colors.
const (
	ColorRed   = 0xED4245 // Alert / failure
	ColorGreen = 0x57F287 // Resolved / healthy
	ColorGray  = 0x95A5A6 // Informational
)

// Discord component types.
const (
	ComponentTypeActionRow = 1
	ComponentTypeButton    = 2
)

// Discord button styles.
const (
	ButtonStylePrimary   = 1
	ButtonStyleSecondary = 2
	ButtonStyleSuccess   = 3
	ButtonStyleDanger    = 4
	ButtonStyleLink      = 5
)

// Discord interaction response types.
const (
	InteractionResponseTypeMessage       = 4 // Respond with a message
	InteractionResponseTypeUpdateMessage = 7 // Update the original message
)

// MessagePayload is the request body for creating or editing a Discord message.
type MessagePayload struct {
	Content    string      `json:"content,omitempty"`
	Embeds     []Embed     `json:"embeds,omitempty"`
	Components []ActionRow `json:"components,omitempty"`
}

// Embed represents a Discord embed object.
type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
}

// EmbedField represents a field within a Discord embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// ActionRow is a container for interactive components (buttons, selects).
type ActionRow struct {
	Type       int         `json:"type"` // always ComponentTypeActionRow (1)
	Components []Component `json:"components"`
}

// Component represents an interactive component such as a button.
type Component struct {
	Type     int    `json:"type"`                // ComponentTypeButton (2)
	Label    string `json:"label"`
	Style    int    `json:"style"`               // 1=primary, 2=secondary, 3=success, 4=danger, 5=link
	CustomID string `json:"custom_id,omitempty"` // for non-link buttons
	URL      string `json:"url,omitempty"`       // for link buttons (style 5)
	Disabled bool   `json:"disabled,omitempty"`
}

// Message represents a Discord message returned from the API.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
}

// Channel represents a Discord channel or thread returned from the API.
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InteractionResponse is the payload sent to respond to a Discord interaction.
type InteractionResponse struct {
	Type int                      `json:"type"` // 4=message, 7=update message
	Data *InteractionCallbackData `json:"data,omitempty"`
}

// InteractionCallbackData contains the message data for an interaction response.
type InteractionCallbackData struct {
	Content    string      `json:"content,omitempty"`
	Embeds     []Embed     `json:"embeds,omitempty"`
	Components []ActionRow `json:"components,omitempty"`
}

// CheckAlertInfo contains the check information used when posting alerts to Discord.
// This mirrors the Slack version for consistency.
type CheckAlertInfo struct {
	UUID          string
	Name          string
	Project       string
	Group         string
	CheckType     string
	Frequency     string // e.g. "5m"
	Message       string // error message
	IsHealthy     bool
	Severity      string // "critical", "degraded", "resolved"
	Target        string // check target: URL for HTTP, host:port for TCP/DB, host for ICMP
	OriginalError string // original error message, used when resolving to preserve context
}

// typeEmoji returns the emoji for a check type.
func typeEmoji(checkType string) string {
	switch checkType {
	case "http":
		return "🌐"
	case "tcp":
		return "🔌"
	case "icmp":
		return "📡"
	case "pgsql", "postgresql":
		return "🐘"
	case "mysql":
		return "🐬"
	case "passive":
		return "⏳"
	default:
		return "🔍"
	}
}

// severityEmoji returns the header emoji based on severity/health status.
func severityEmoji(info CheckAlertInfo) string {
	if info.IsHealthy {
		return "🟢"
	}
	switch info.Severity {
	case "degraded":
		return "🟡"
	default:
		return "🔴"
	}
}

// statusText returns a human-readable status string with emoji.
func statusText(isHealthy bool) string {
	if isHealthy {
		return "🟢 Healthy"
	}
	return "🔴 Unhealthy"
}
