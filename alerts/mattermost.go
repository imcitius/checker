package alerts

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"my/checker/config"
	"net/http"
	"sync"
)

type Mattermost struct {
	Alerter
}

type MmMessage struct {
	Text string `json:"text"`
}

func (m *Mattermost) Send(a *config.AlertConfigs, message, messageType string) error {
	config.Log.Debugf("Alert send: %s (alert details %+v)", message, a)

	mmMessage := &MmMessage{
		Text: message,
	}

	text, err := json.Marshal(mmMessage)
	if err != nil {
		config.Log.Errorf("Json marshal error: %s", message)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", a.MMWebHookURL, bytes.NewBuffer(text))
	if err != nil {
		config.Log.Errorf("Cannot generate http client for mattermost: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		config.Log.Errorf("Cannot post to mattermost: %s", err)
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		config.Log.Errorf("Mattermost response not empty: %s (%+v)", err, bodyText)
	}

	return err
}

func (t Mattermost) InitBot(ch chan bool, wg *sync.WaitGroup) {
	config.Log.Panic("Mattermost bot not implemented yet")
}
