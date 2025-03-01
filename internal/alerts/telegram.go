package alerts

import (
	"fmt"
	"net/http"
	"net/url"
)

// SendTelegramAlert sends a message to a Telegram channel using a bot token
func SendTelegramAlert(botToken, chatID, message string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", botToken)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram alert failed with status %d", resp.StatusCode)
	}
	return nil
}
