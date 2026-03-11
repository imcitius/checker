package slack

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

// SlackClient provides methods to interact with the Slack Web API using Block Kit.
type SlackClient struct {
	botToken         string
	signingSecret    string
	defaultChannelID string
	api              *slack.Client
}

// NewSlackClient creates a new SlackClient with the given bot token, signing secret, and default channel.
func NewSlackClient(botToken, signingSecret, defaultChannel string) *SlackClient {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	api := slack.New(botToken, slack.OptionHTTPClient(httpClient))

	return &SlackClient{
		botToken:         botToken,
		signingSecret:    signingSecret,
		defaultChannelID: defaultChannel,
		api:              api,
	}
}

// DefaultChannelID returns the configured default channel ID.
func (c *SlackClient) DefaultChannelID() string {
	return c.defaultChannelID
}

// SendAlert posts an alert message with Block Kit blocks to the specified channel.
// It returns the message timestamp (ts) for threading.
func (c *SlackClient) SendAlert(ctx context.Context, channelID string, info CheckAlertInfo) (string, error) {
	blocks := BuildAlertBlocks(info)
	fallback := fmt.Sprintf("%s %s: %s", severityEmoji(info), info.Name, info.Message)

	_, ts, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText(fallback, false),
	)
	if err != nil {
		return "", fmt.Errorf("SendAlert: %w", err)
	}

	return ts, nil
}

// SendResolve posts a resolution reply in the thread and updates the original message.
// It changes the header to green, removes action buttons, and posts a thread reply.
func (c *SlackClient) SendResolve(ctx context.Context, info CheckAlertInfo, originalThreadTS, channelID string) error {
	// 1. Post resolution reply in the thread
	resolveBlocks := BuildResolveBlocks(info)
	resolveFallback := fmt.Sprintf("🟢 %s recovered", info.Name)

	_, _, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionBlocks(resolveBlocks...),
		slack.MsgOptionText(resolveFallback, false),
		slack.MsgOptionTS(originalThreadTS),
	)
	if err != nil {
		return fmt.Errorf("SendResolve thread reply: %w", err)
	}

	// 2. Update the original message: change emoji to green, remove action buttons
	resolvedInfo := info
	resolvedInfo.IsHealthy = true
	resolvedInfo.Severity = "resolved"
	updatedBlocks := BuildResolvedOriginalBlocks(resolvedInfo)
	updatedFallback := fmt.Sprintf("🟢 RESOLVED: %s", info.Name)

	_, _, _, err = c.api.UpdateMessageContext(ctx, channelID, originalThreadTS,
		slack.MsgOptionBlocks(updatedBlocks...),
		slack.MsgOptionText(updatedFallback, false),
	)
	if err != nil {
		return fmt.Errorf("SendResolve update original: %w", err)
	}

	return nil
}

// SendSilenceConfirmation posts a thread reply confirming a silence was applied.
func (c *SlackClient) SendSilenceConfirmation(ctx context.Context, channelID, threadTS, scope, target, duration, user string) error {
	blocks := BuildSilenceConfirmationBlocks(scope, target, duration, user)
	fallback := fmt.Sprintf("🔇 Silence applied: %s %s for %s by <@%s>", scope, target, duration, user)

	_, _, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText(fallback, false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		return fmt.Errorf("SendSilenceConfirmation: %w", err)
	}

	return nil
}

// SendAlertReply posts an alert message as a thread reply to an existing message.
// It returns the reply message timestamp.
func (c *SlackClient) SendAlertReply(ctx context.Context, channelID, threadTS string, info CheckAlertInfo) (string, error) {
	blocks := BuildAlertBlocks(info)
	fallback := fmt.Sprintf("%s %s: %s", severityEmoji(info), info.Name, info.Message)

	_, ts, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText(fallback, false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		return "", fmt.Errorf("SendAlertReply: %w", err)
	}

	return ts, nil
}

// PostAlert is a backward-compatible wrapper around SendAlert.
func (c *SlackClient) PostAlert(ctx context.Context, channelID string, info CheckAlertInfo) (string, error) {
	return c.SendAlert(ctx, channelID, info)
}

// PostResolution is a backward-compatible wrapper for posting a resolution thread reply.
func (c *SlackClient) PostResolution(ctx context.Context, channelID, threadTs string, info CheckAlertInfo) error {
	return c.SendResolve(ctx, info, threadTs, channelID)
}

// UpdateMessageColor is a backward-compatible wrapper. With Block Kit, this is handled
// by SendResolve which updates the original message. This method is kept for callers
// that update the message independently.
func (c *SlackClient) UpdateMessageColor(ctx context.Context, channelID, messageTs string, color string, info CheckAlertInfo) error {
	resolvedInfo := info
	resolvedInfo.IsHealthy = true
	resolvedInfo.Severity = "resolved"
	updatedBlocks := BuildResolvedOriginalBlocks(resolvedInfo)
	updatedFallback := fmt.Sprintf("🟢 RESOLVED: %s", info.Name)

	_, _, _, err := c.api.UpdateMessageContext(ctx, channelID, messageTs,
		slack.MsgOptionBlocks(updatedBlocks...),
		slack.MsgOptionText(updatedFallback, false),
	)
	if err != nil {
		return fmt.Errorf("UpdateMessageColor: %w", err)
	}

	return nil
}

// SigningSecret returns the configured Slack signing secret.
func (c *SlackClient) SigningSecret() string {
	return c.signingSecret
}

// UpdateMessage updates an existing Slack message with new blocks and fallback text.
func (c *SlackClient) UpdateMessage(ctx context.Context, channelID, messageTs string, blocks []slack.Block, fallbackText string) error {
	_, _, _, err := c.api.UpdateMessageContext(ctx, channelID, messageTs,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText(fallbackText, false),
	)
	if err != nil {
		return fmt.Errorf("UpdateMessage: %w", err)
	}
	return nil
}

// PostThreadReply posts a text message as a thread reply and returns the reply timestamp.
func (c *SlackClient) PostThreadReply(ctx context.Context, channelID, threadTS, text string) (string, error) {
	_, ts, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		return "", fmt.Errorf("PostThreadReply: %w", err)
	}
	return ts, nil
}

// PostEphemeral posts an ephemeral message visible only to the specified user.
func (c *SlackClient) PostEphemeral(ctx context.Context, channelID, userID, text string) error {
	_, err := c.api.PostEphemeralContext(ctx, channelID, userID,
		slack.MsgOptionText(text, false),
	)
	if err != nil {
		return fmt.Errorf("PostEphemeral: %w", err)
	}

	return nil
}
