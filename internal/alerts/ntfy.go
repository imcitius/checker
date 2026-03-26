package alerts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ntfyConfig holds the configuration for an ntfy.sh alert channel.
type ntfyConfig struct {
	ServerURL string `json:"server_url"` // default "https://ntfy.sh"
	Topic     string `json:"topic"`      // required
	Token     string `json:"token"`      // optional token auth (Basic with token as username)
	Username  string `json:"username"`   // optional Basic auth
	Password  string `json:"password"`   // optional Basic auth
	Icon      string `json:"icon"`       // optional notification icon URL
	ClickURL  string `json:"click_url"`  // base URL of Checker UI, e.g. "https://checker.example.com"
	Insecure  bool   `json:"insecure"`   // skip TLS certificate verification
}

// ntfyAction represents an interactive button in an ntfy notification.
type ntfyAction struct {
	Action  string            `json:"action"`            // "view", "http", "broadcast", "copy"
	Label   string            `json:"label"`
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`  // for "http" action
	Headers map[string]string `json:"headers,omitempty"` // for "http" action
	Body    string            `json:"body,omitempty"`    // for "http" action
	Clear   bool              `json:"clear,omitempty"`
}

// ntfyPayload is the JSON body sent to the ntfy server.
type ntfyPayload struct {
	Topic    string       `json:"topic"`
	Title    string       `json:"title"`
	Message  string       `json:"message"`
	Priority int          `json:"priority"`
	Tags     []string     `json:"tags"`
	Markdown bool         `json:"markdown"`
	Icon     string       `json:"icon,omitempty"`
	Click    string       `json:"click,omitempty"`   // URL opened on notification tap
	Actions  []ntfyAction `json:"actions,omitempty"` // up to 3 action buttons
}

// NtfyAlerter implements the Alerter interface for ntfy.sh notifications.
type NtfyAlerter struct {
	config ntfyConfig
}

func (a *NtfyAlerter) Type() string { return "ntfy" }

func (a *NtfyAlerter) httpClient() *http.Client {
	if a.config.Insecure {
		return insecureHTTPClient
	}
	return defaultHTTPClient
}

func (a *NtfyAlerter) SendAlert(p AlertPayload) error {
	priority := ntfyPriority(p.Severity)
	msg := fmt.Sprintf("**Check:** %s\n**Project:** %s\n**Type:** %s\n**Error:** %s",
		p.CheckName, p.Project, p.CheckType, p.Message)

	payload := ntfyPayload{
		Topic:    a.config.Topic,
		Title:    fmt.Sprintf("%s is DOWN", p.CheckName),
		Message:  msg,
		Priority: priority,
		Tags:     []string{"rotating_light", p.CheckType},
		Markdown: true,
		Icon:     a.config.Icon,
	}

	if a.config.ClickURL != "" {
		checkURL := fmt.Sprintf("%s/checks/%s", strings.TrimRight(a.config.ClickURL, "/"), p.CheckUUID)
		payload.Click = checkURL
		payload.Actions = []ntfyAction{
			{
				Action: "view",
				Label:  "View in Checker",
				URL:    checkURL,
			},
		}
	}

	url := a.config.ServerURL
	headers := a.authHeaders()
	if err := postJSONWithClient(a.httpClient(), url, payload, headers); err != nil {
		return fmt.Errorf("ntfy alert: %w", err)
	}
	return nil
}

func (a *NtfyAlerter) SendRecovery(p RecoveryPayload) error {
	msg := fmt.Sprintf("**Check:** %s\n**Project:** %s\n**Type:** %s",
		p.CheckName, p.Project, p.CheckType)

	payload := ntfyPayload{
		Topic:    a.config.Topic,
		Title:    fmt.Sprintf("%s is RESOLVED", p.CheckName),
		Message:  msg,
		Priority: 3, // normal
		Tags:     []string{"white_check_mark", p.CheckType},
		Markdown: true,
		Icon:     a.config.Icon,
	}

	if a.config.ClickURL != "" {
		checkURL := fmt.Sprintf("%s/checks/%s", strings.TrimRight(a.config.ClickURL, "/"), p.CheckUUID)
		payload.Click = checkURL
		payload.Actions = []ntfyAction{
			{
				Action: "view",
				Label:  "View in Checker",
				URL:    checkURL,
			},
		}
	}

	url := a.config.ServerURL
	headers := a.authHeaders()
	if err := postJSONWithClient(a.httpClient(), url, payload, headers); err != nil {
		return fmt.Errorf("ntfy recovery: %w", err)
	}
	return nil
}

// ntfyPriority maps severity strings to ntfy priority levels.
func ntfyPriority(severity string) int {
	switch severity {
	case "critical":
		return 5 // urgent
	case "warning":
		return 4 // high
	default:
		return 3 // normal
	}
}

// authHeaders builds the authorization headers for the ntfy request.
func (a *NtfyAlerter) authHeaders() map[string]string {
	headers := map[string]string{}
	if a.config.Token != "" {
		// ntfy access tokens authenticate via Basic Auth with token as username
		// and empty password, per https://docs.ntfy.sh/config/#access-tokens
		creds := base64.StdEncoding.EncodeToString(
			[]byte(a.config.Token + ":"),
		)
		headers["Authorization"] = "Basic " + creds
	} else if a.config.Username != "" && a.config.Password != "" {
		creds := base64.StdEncoding.EncodeToString(
			[]byte(a.config.Username + ":" + a.config.Password),
		)
		headers["Authorization"] = "Basic " + creds
	}
	return headers
}

// validateNtfyURL checks that the given URL is a valid HTTP(S) URL with a host.
func validateNtfyURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("ntfy server_url is not a valid HTTP(S) URL: %q", rawURL)
	}
	return nil
}

// SendNtfyTest sends a test notification to an ntfy server.
func SendNtfyTest(serverURL, topic, token, username, password, message, clickURL string, insecure bool) error {
	if serverURL == "" {
		serverURL = "https://ntfy.sh"
	}
	if err := validateNtfyURL(serverURL); err != nil {
		return fmt.Errorf("invalid server URL: %q — must be a valid HTTP(S) URL (e.g. https://ntfy.example.com)", serverURL)
	}
	serverURL = strings.TrimRight(serverURL, "/")
	payload := ntfyPayload{
		Topic:    topic,
		Title:    "Checker Test Notification",
		Message:  message,
		Priority: 3,
		Tags:     []string{"test_tube"},
		Markdown: false,
	}
	if clickURL != "" {
		testURL := strings.TrimRight(clickURL, "/") + "/checks"
		payload.Click = testURL
		payload.Actions = []ntfyAction{
			{
				Action: "view",
				Label:  "View in Checker",
				URL:    testURL,
			},
		}
	}
	alerter := &NtfyAlerter{config: ntfyConfig{
		ServerURL: serverURL,
		Topic:     topic,
		Token:     token,
		Username:  username,
		Password:  password,
		Insecure:  insecure,
	}}
	var client *http.Client
	if insecure {
		client = insecureHTTPClient
	} else {
		client = defaultHTTPClient
	}
	return postJSONWithClient(client, serverURL, payload, alerter.authHeaders())
}

func newNtfyAlerter(raw json.RawMessage) (Alerter, error) {
	var cfg ntfyConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing ntfy config: %w", err)
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("ntfy requires topic")
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "https://ntfy.sh"
	}
	// Validate server URL
	if err := validateNtfyURL(cfg.ServerURL); err != nil {
		return nil, fmt.Errorf("ntfy server_url is not a valid HTTP(S) URL: %q", cfg.ServerURL)
	}
	// Normalize: strip trailing slash
	cfg.ServerURL = strings.TrimRight(cfg.ServerURL, "/")
	return &NtfyAlerter{config: cfg}, nil
}

func init() {
	RegisterAlerter("ntfy", newNtfyAlerter)
}
