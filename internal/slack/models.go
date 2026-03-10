package slack

// CheckAlertInfo contains the check information used when posting alerts to Slack.
type CheckAlertInfo struct {
	UUID      string
	Name      string
	Project   string
	Group     string
	CheckType string
	Message   string // error message
	IsHealthy bool
}

// slackRequest is the base request body for Slack API calls.
type slackRequest struct {
	Channel     string        `json:"channel"`
	Text        string        `json:"text,omitempty"`
	Attachments []attachment  `json:"attachments,omitempty"`
	ThreadTS    string        `json:"thread_ts,omitempty"`
	User        string        `json:"user,omitempty"`
	TS          string        `json:"ts,omitempty"`
}

// attachment represents a Slack message attachment.
type attachment struct {
	Color    string            `json:"color"`
	Fallback string           `json:"fallback"`
	Fields   []attachmentField `json:"fields"`
}

// attachmentField represents a field within a Slack attachment.
type attachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// slackResponse is the base response from Slack API calls.
type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	TS    string `json:"ts,omitempty"`
}
