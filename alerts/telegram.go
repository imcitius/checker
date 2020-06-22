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

func (m TgMessage) GetProject() (string, error) {
	var (
		result      []string
		projectName string
		err         error
	)

	conf, _ := json.Marshal(m)
	config.Log.Debugf("Message: %+v\n\n", string(conf))

	if m.IsReply() {
		// try to get from reply
		pattern := regexp.MustCompile("Project: (.*)\n")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		projectName = result[1]
	} else {
		if m.Payload != "" {
			projectName = m.Payload
		}
	}

	if projectName == "" {
		err = fmt.Errorf("Project name extraction error.\nShould be reply to an alert message, or speficied as `/<command> project_name`.")
	} else {
		config.Log.Debugf("Project extracted: %v\n", projectName)
	}

	return projectName, err
}

func (m TgMessage) GetUUID() (string, error) {
	var (
		result []string
		uuid   string
		err    error
	)
	fmt.Printf("message: %v\n", m.Text)

	if m.IsReply() {
		// try to get uuid from reply
		pattern := regexp.MustCompile("UUID: (.*)")
		result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		uuid = result[1]
	} else {
		if m.Payload != "" {
			uuid = m.Payload
		}
	}

	if uuid == "" {
		err = fmt.Errorf("UUID extraction error.\nShould be reply to an alert message, or speficied as `/<command> UUID`.")
	} else {
		config.Log.Debugf("UUID extracted: %v\n", uuid)
	}

	return uuid, err

	// WIP test and write error handling
}

type Telegram struct {
	Alerter
}

func special(b byte) bool {

	specials := []byte{'_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!'}

	for _, s := range specials {
		if s == b {
			return true
		}
	}
	return false
}

func QuoteMeta(s string) string {
	// A byte loop is correct because all metacharacters are ASCII.
	var i int
	for i = 0; i < len(s); i++ {
		if special(s[i]) {
			break
		}
	}
	// No meta characters found, so return original string.
	if i >= len(s) {
		return s
	}

	b := make([]byte, 2*len(s)-i)
	copy(b, s[:i])
	j := i
	for ; i < len(s); i++ {
		if special(s[i]) {
			b[j] = '\\'
			j++
		}
		b[j] = s[i]
		j++
	}
	return string(b[:j])
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

	options := new(tb.SendOptions)
	options.ParseMode = "MarkDownV2"

	config.Log.Infof("quoted message: %s", QuoteMeta(message))

	_, err = bot.Send(&user, QuoteMeta(message), options)
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

	a, err := GetCommandChannel()
	if err != nil {
		config.Log.Infof("GetCommandChannel error: %s", err)
	}

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
		SendChatOps("All messages ceased")
	})

	bot.Handle("/ua", func(m *tb.Message) {
		config.Log.Infof("Bot request /ua")

		status.MainStatus = "loud"
		SendChatOps("All messages enabled")
	})

	bot.Handle("/pu", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}
		uuID, err := tgMessage.GetUUID()
		if err != nil {
			SendChatOps(fmt.Sprintf("%s", err))
			return
		}

		config.Log.Infof("Bot request /pu")
		config.Log.Printf("Pause req for UUID: %+v\n", uuID)
		status.SetCheckMode(checks.GetCheckByUUID(uuID), "quiet")

		SendChatOps(fmt.Sprintf("Messages ceased for UUID %v", uuID))
	})

	bot.Handle("/uu", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}
		uuID, err := tgMessage.GetUUID()
		if err != nil {
			SendChatOps(fmt.Sprintf("%s", err))
			return
		}
		config.Log.Infof("Bot request /uu")
		config.Log.Printf("Unpause req for UUID: %+v\n", uuID)
		status.SetCheckMode(checks.GetCheckByUUID(uuID), "loud")

		SendChatOps(fmt.Sprintf("Messages resumed for UUID %v", uuID))
	})

	bot.Handle("/pp", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /pp")
		projectName, err := tgMessage.GetProject()
		if err != nil {
			SendChatOps(fmt.Sprintf("%s", err))
			return
		}

		project := projects.GetProjectByName(projectName)
		config.Log.Printf("Pause req for project: %s\n", projectName)
		status.SetProjectMode(project, "loud")

		SendChatOps(fmt.Sprintf("Messages ceased for project %s", projectName))
	})

	bot.Handle("/up", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		var tgMessage config.IncomingChatMessage
		tgMessage = TgMessage{m}

		config.Log.Infof("Bot request /up")

		projectName, err := tgMessage.GetProject()
		if err != nil {
			SendChatOps(fmt.Sprintf("%s", err))
			return
		}

		project := projects.GetProjectByName(projectName)
		config.Log.Printf("Resume req for project: %s\n", projectName)
		status.SetProjectMode(project, "quiet")

		SendChatOps(fmt.Sprintf("Messages resumed for project %s", projectName))
	})

	bot.Handle("/stats", func(m *tb.Message) {
		metrics.AddAlertMetricChatOpsRequest(a)

		config.Log.Infof("Bot request /stats from %s", m.Sender.Username)

		SendChatOps(fmt.Sprintf("@" + m.Sender.Username + "\n\n" + metrics.GenTextRuntimeStats()))
	})

	go func() {
		var message string
		config.Log.Debugf("Internal status is: %s", config.InternalStatus)
		switch config.InternalStatus {
		case "reload":
			message = "Config reloaded"
		default:
			message = fmt.Sprintf("Bot at your service (%s, %s, %s)", config.Version, config.VersionSHA, config.VersionBuild)
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
