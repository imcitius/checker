package alerts

import (
	"encoding/json"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
	"regexp"
	"sync"
	"time"
)

var (
	TgSignalCh chan bool
)

func init() {
	TgSignalCh = make(chan bool)
}

type TgMessage struct {
	*tb.Message
}

func (m TgMessage) GetProject() string {
	var (
		result      []string
		projectName string
	)

	conf, _ := json.Marshal(m)
	config.Log.Printf("Message: %+v\n\n", string(conf))

	if m.IsReply() {
		// try to get from reply
		pattern := regexp.MustCompile("roject: (.*)\n")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		projectName = result[1]
	} else {
		projectName = m.Payload
	}

	if result == nil {
		fmt.Printf("Project extraction error.")
	} else {
		fmt.Printf("Project extracted: %v\n", projectName)
	}

	return projectName
}

func (m TgMessage) GetUUID() string {
	var (
		result []string
		uuid   string
	)
	fmt.Printf("message: %v\n", m.Text)

	if m.IsReply() {
		// try to get uuid from reply
		pattern := regexp.MustCompile("UUID: (.*)")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		uuid = result[1]
	} else {
		uuid = m.Payload
	}

	if result == nil {
		fmt.Printf("UUID extraction error.")
	} else {
		fmt.Printf("UUID extracted: %v\n", uuid)
	}

	return uuid

	// WIP test and write error handling
}


type Telegram struct {
	Alerter
}

func (t Telegram) Send(a *config.AlertConfigs, message string) error {
	config.Log.Debugf("Sending alert, text: '%s' (alert channel %+v)", message, a.Name)
	bot, err := tb.NewBot(tb.Settings{
		Token:  a.BotToken,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})
	if err != nil {
		config.Log.Fatal(err)
	}
	user := tb.Chat{ID: a.ProjectChannel}
	//config.Log.Debugf("Alert to user: %+v with token %s, error: %+v", user, a.BotToken, e)

	_, err = bot.Send(&user, message)
	if err != nil {
		config.Log.Warnf("SendTgMessage error: %v", err)
	} else {
		config.Log.Debugf("sendTgMessage success")
		metrics.AddAlertCounter(a, "noncrit")
	}

	return err

}


func (t Telegram) InitBot(ch chan bool, wg *sync.WaitGroup) {

	a := GetCommandChannel()

	defer wg.Done()

	bot, err := tb.NewBot(tb.Settings{
		Token:  a.BotToken,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})

	if err != nil {
		config.Log.Fatal(err)
	}

	bot.Handle("/pa", func(m *tb.Message) {
		config.Log.Infof("Bot request /pa")

		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		config.Config.Defaults.Parameters.Mode = "quiet"
		answer := "All messages ceased"
		bot.Send(m.Chat, answer)
	})

	bot.Handle("/ua", func(m *tb.Message) {
		config.Log.Infof("Bot request /ua")

		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		config.Config.Defaults.Parameters.Mode = "loud"
		answer := "All messages enabled"
		bot.Send(m.Chat, answer)
	})

	bot.Handle("/pu", func(m *tb.Message) {
		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /pu")

		uuID := tgMessage.GetUUID()
		config.Log.Printf("Pause req for UUID: %+v\n", uuID)
		for _, project := range config.Config.Projects {
			for _, healthcheck := range project.Healtchecks {
				for _, check := range healthcheck.Checks {
					if uuID == check.UUid {
						_ = checks.CeaseAlerts(&check)
					}
				}
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages ceased for UUID %v", uuID)
			bot.Send(m.Chat, answer)
		}
	})

	bot.Handle("/uu", func(m *tb.Message) {
		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /uu")

		uuID := tgMessage.GetUUID()
		config.Log.Printf("Resume req for UUID: %+v\n", uuID)
		for _, project := range config.Config.Projects {
			for _, healthcheck := range project.Healtchecks {
				for _, check := range healthcheck.Checks {
					if uuID == check.UUid {
						_ = checks.EnableAlerts(&check)
					}
				}
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages resumed for UUID %v", uuID)
			bot.Send(m.Chat, answer)
		}

	})

	bot.Handle("/pp", func(m *tb.Message) {
		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /pp")

		projectName := tgMessage.GetProject()
		config.Log.Printf("Pause req for project: %s\n", projectName)
		for _, project := range config.Config.Projects {
			if projectName == project.Name {
				_ = config.CeaseProjectAlerts(&project)
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages ceased for project %s", projectName)
			bot.Send(m.Chat, answer)
		}

	})

	bot.Handle("/up", func(m *tb.Message) {
		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++
		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /up")

		projectName := tgMessage.GetProject()
		config.Log.Printf("Resume req for project: %s\n", projectName)
		for _, project := range config.Config.Projects {
			if projectName == project.Name {
				_ = config.EnableProjectAlerts(&project)
			}
		}
		if err == nil {
			answer := fmt.Sprintf("Messages resumed for project %s", projectName)
			bot.Send(m.Chat, answer)
		}

	})

	bot.Handle("/stats", func(m *tb.Message) {
		metrics.Metrics.Alerts[GetCommandChannel().Name].CommandReqs++

		config.Log.Infof("Bot request /stats from %s", m.Sender.Username)

		answer := fmt.Sprintf("@" + m.Sender.Username + "\n\n" + metrics.GenRuntimeStats())
		bot.Send(m.Chat, answer)

	})

	go func() {
		config.Log.Infof("Start listening telegram bots routine")
		SendChatOps("Bot %s at your service" + fmt.Sprintf(" (%s, %s, %s)", config.Version, config.VersionSHA, config.VersionBuild))
		bot.Start()
	}()

	<-ch
	bot.Stop()
	config.Log.Infof("Exit listening telegram bots")
	return
}
