package alerts

import (
	"encoding/json"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"my/checker/config"
	"my/checker/metrics"
	"regexp"
	"sync"
	"time"
)

var (
	TgSignalCh chan bool

	selectorAlert = &tb.ReplyMarkup{}
	selPU         = selectorAlert.Data("pu", "pu")
	selPP         = selectorAlert.Data("pp", "pp")

	menu     = &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	btnHelp  = menu.Text("ℹ️ Help")
	btnPA    = menu.Text("⏸️ Pause All")
	btnUA    = menu.Text("▶️ Unpause All")
	btnList  = menu.Text("🔭 List")
	btnStats = menu.Text("📊 Stats")
)

func init() {
	TgSignalCh = make(chan bool)
}

type TgMessage struct {
	*tb.Message
}

func (m TgMessage) GetProject() (string, error) {
	var (
		result      []string
		projectName string
		err         error
	)

	conf, _ := json.Marshal(m)
	config.Log.Debugf("Message: %+v\n\n", string(conf))

	if m.Payload != "" {
		projectName = m.Payload
	} else {
		// try to get project from message text
		pattern := regexp.MustCompile(".*project: (.*)\n")
		if m.IsReply() {
			result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		} else {
			result = pattern.FindStringSubmatch(m.Message.Text)
		}
		projectName = result[1]
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
	config.Log.Infof("message: %v\n", m.Text)

	if m.Payload != "" {
		uuid = m.Payload
	} else {
		// try to get uuid from reply
		pattern := regexp.MustCompile("UUID: (.*)")
		if m.IsReply() {
			result = pattern.FindStringSubmatch(m.ReplyTo.Text)
		} else {
			result = pattern.FindStringSubmatch(m.Message.Text)
		}
		uuid = result[1]
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
func (t Telegram) Send(a *config.AlertConfigs, message, messageType string) error {

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

	options := &tb.SendOptions{ParseMode: "MarkDownV2"}

	menu.Reply(
		menu.Row(btnHelp, btnList),
		menu.Row(btnPA, btnUA),
	)

	//config.Log.Debugf("Bot quoted answer: %s", QuoteMeta(message))

	switch messageType {
	case "alert":
		selectorAlert.Inline(selectorAlert.Row(selPP, selPU))
		_, err = bot.Send(&user, QuoteMeta(message), options, menu, selectorAlert)
	default:
		_, err = bot.Send(&user, QuoteMeta(message), options, menu)
	}

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

	bot.Handle("/pa", func(m *tb.Message) { paHandler() })
	bot.Handle("/ua", func(m *tb.Message) { uaHandler() })
	bot.Handle("/uu", func(m *tb.Message) { uuHandler(m, a) })
	bot.Handle("/pp", func(m *tb.Message) { ppHandler(m, a) })
	bot.Handle("/pu", func(m *tb.Message) { puHandler(m, a) })
	bot.Handle("/up", func(m *tb.Message) { upHandler(m, a) })
	bot.Handle("/stats", func(m *tb.Message) { statsHandler(m) })

	bot.Handle(&btnHelp, func(m *tb.Message) {
		config.Log.Infof("Help pressed")
		SendChatOps(fmt.Sprintf("@" + m.Sender.Username + "\n\n" + "that should be help"))
	})
	bot.Handle(&btnList, func(m *tb.Message) {
		config.Log.Infof("List pressed")
		SendChatOps(fmt.Sprintf("@" + m.Sender.Username + "\n\n" + config.ListElements()))
	})
	bot.Handle(&btnList, func(m *tb.Message) {
		config.Log.Infof("Stats pressed")
		statsHandler(m)
	})
	bot.Handle(&btnPA, func(m *tb.Message) {
		config.Log.Infof("PA pressed")
		paHandler()
	})
	bot.Handle(&btnUA, func(m *tb.Message) {
		config.Log.Infof("UA pressed")
		uaHandler()
	})

	// On inline button pressed (callback)
	bot.Handle(&selPU, func(c *tb.Callback) {
		puHandler(c.Message, a)
		// ...
		// Always respond!
		bot.Respond(c, &tb.CallbackResponse{Text: "trying"})
	})

	// On inline button pressed (callback)
	bot.Handle(&selPP, func(c *tb.Callback) {
		ppHandler(c.Message, a)
		// ...
		// Always respond!
		bot.Respond(c, &tb.CallbackResponse{Text: "trying"})
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
