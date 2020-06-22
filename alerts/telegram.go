package alerts

import (
	"encoding/json"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
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

func init() {
	AlerterCollection["telegram"] = new(Telegram)
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
		metrics.AddAlertMetricNonCritical(a)
	}

	return err

}

func (t Telegram) InitBot(ch chan bool, wg *sync.WaitGroup) {

	var verbosity bool

	a := GetCommandChannel()

	defer wg.Done()

	if config.Log.GetLevel().String() == "debug" {
		verbosity = true
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:   a.BotToken,
		Poller:  &tb.LongPoller{Timeout: 5 * time.Second},
		Verbose: verbosity,
	})

	if err != nil {
		config.Log.Fatal(err)
	}

	bot.Handle("/pa", func(m *tb.Message) {
		config.Log.Infof("Bot request /pa")

		metrics.AddAlertMetricChatOpsRequest(a)
		status.MainStatus = "quiet"
		answer := "All messages ceased"
		bot.Send(m.Chat, answer)
	})

	bot.Handle("/ua", func(m *tb.Message) {
		config.Log.Infof("Bot request /ua")

		status.MainStatus = "loud"
		answer := "All messages enabled"
		bot.Send(m.Chat, answer)
	})

	bot.Handle("/pu", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}
		uuID := tgMessage.GetUUID()

		config.Log.Infof("Bot request /pu")
		config.Log.Printf("Pause req for UUID: %+v\n", uuID)
		status.SetCheckMode(checks.GetCheckByUUID(uuID), "quiet")

		answer := fmt.Sprintf("Messages ceased for UUID %v", uuID)
		bot.Send(m.Chat, answer)
	})

	bot.Handle("/uu", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}
		uuID := tgMessage.GetUUID()
		config.Log.Infof("Bot request /uu")
		config.Log.Printf("Unpause req for UUID: %+v\n", uuID)
		status.SetCheckMode(checks.GetCheckByUUID(uuID), "loud")

		answer := fmt.Sprintf("Messages resumed for UUID %v", uuID)
		bot.Send(m.Chat, answer)

	})

	bot.Handle("/pp", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /pp")
		project := projects.GetProjectByName(tgMessage.GetProject())
		config.Log.Printf("Pause req for project: %s\n", tgMessage.GetProject())
		status.SetProjectMode(project, "loud")

		answer := fmt.Sprintf("Messages ceased for project %s", tgMessage.GetProject())
		bot.Send(m.Chat, answer)

	})

	bot.Handle("/up", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /up")

		projectName := projects.GetProjectByName(tgMessage.GetProject())
		config.Log.Printf("Resume req for project: %s\n", tgMessage.GetProject())
		status.SetProjectMode(projectName, "quiet")

		answer := fmt.Sprintf("Messages resumed for project %s", tgMessage.GetProject())
		bot.Send(m.Chat, answer)

	})

	bot.Handle("/stats", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		config.Log.Infof("Bot request /stats from %s", m.Sender.Username)

		answer := fmt.Sprintf("@" + m.Sender.Username + "\n\n" + metrics.GenTextRuntimeStats())
		bot.Send(m.Chat, answer)

	})

	go func() {
		var message string
		switch config.InternalStatus {
		case "start":
			message = fmt.Sprintf("Bot at your service (%s, %s, %s)", config.Version, config.VersionSHA, config.VersionBuild)
		case "reload":
			message = "Config reloaded"
		}
		config.Log.Infof("Start listening telegram bots routine")
		SendChatOps(message)
		bot.Start()
		SendChatOps("Bot is stopped")
	}()

	<-ch
	bot.Stop()
	// let bot to actually stop
	config.Log.Infof("Exit listening telegram bots")
	time.Sleep(5 * time.Second)
	return
}
