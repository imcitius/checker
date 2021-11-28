package alerts

import (
	"encoding/json"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v3"
	"my/checker/config"
	"my/checker/reports"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var (
	//TgSignalCh chan bool

	selectorAlert = &tb.ReplyMarkup{}
	selQU         = selectorAlert.Data("Quiet UUID", "pu")
	selQP         = selectorAlert.Data("Quiet project", "pp")

	//menu     = &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	menu     = &tb.ReplyMarkup{}
	btnHelp  = menu.Text("ℹ️ Help")
	btnQA    = menu.Text("⏸️ Quiet All")
	btnLA    = menu.Text("▶️ Loud All")
	btnList  = menu.Text("🔭 List")
	btnStats = menu.Text("📊 Stats")

	help = "This is checker version " + config.Version + "\n" +
		"Please use following commands: \n" +
		"/qa,/la - Pause/Unpause all alerts (or use main keyboard buttons)\n" +
		"/qp,/lp - Pause/Unpause specific project (or use message button)\n" +
		"/qu,/lu - Pause/Unpause specific check by UUID (or use message button)\n" +
		"/list - List all projects and checks with UUID's\n\n\n" +
		"/stats - Show app statistics\n" +
		"/version - Show app version\n" +
		"\n\nPlease find detailed documentation here:\n" +
		"https://github.com/imcitius/checker"
)

func init() {
	//TgSignalCh = make(chan bool)
}

type TgMessage struct {
	*tb.Message
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
		err = fmt.Errorf("project name extraction error.\nShould be reply to an alert message, or speficied as `/<command> project_name`")
	} else {
		config.Log.Debugf("project extracted: %v\n", projectName)
	}

	return projectName, err
}

func (m TgMessage) GetUUID() (string, error) {
	var (
		result []string
		uuid   string
		err    error
	)
	config.Log.Debugf("message: %v\n", m.Text)

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
		err = fmt.Errorf("UUID extraction error.\nShould be reply to an alert message, or speficied as `/<command> UUID`")
	} else {
		config.Log.Debugf("UUID extracted: %v\n", uuid)
	}

	return uuid, err

	// WIP test and write error handling
}

func (t Telegram) Send(a *AlertConfigs, message, messageType string) error {

	config.Log.Debugf("Sending alert type %s, text: '%s' (alert channel %+v)", messageType, message, a.Name)
	bot, err := tb.NewBot(tb.Settings{
		Token:  a.BotToken,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})
	if err != nil {
		config.Log.Fatal(err)
	}
	//config.Log.Debugf("Alert to user: %+v with token %s, error: %+v", user, a.BotToken, e)

	options := &tb.SendOptions{ParseMode: "MarkDownV2"}
	optionsChatops := &tb.SendOptions{ParseMode: "MarkDownV2", DisableNotification: true}

	menu.Reply(
		menu.Row(btnHelp, btnList),
		menu.Row(btnQA, btnLA),
	)

	//config.Log.Debugf("Bot quoted answer: %s", QuoteMeta(message))

	switch messageType {
	case "alert":
		config.Log.Infof("Sending 'alert' type message")
		id, err := strconv.Atoi(a.ProjectChannel)
		if err != nil {
			config.Log.Fatalf("Cannot parse project channel %s", a.ProjectChannel)
		}
		user := tb.Chat{ID: int64(id)}

		messageToSend := tb.Message{Text: message}
		selectorAlert.Inline(selectorAlert.Row(selQP, selQU))
		_, err = bot.Send(&user, QuoteMeta(messageToSend.Text), options, menu, selectorAlert)
	case "critalert":
		config.Log.Infof("Sending 'critalert' type message")
		id, err := strconv.Atoi(a.ProjectChannel)
		if err != nil {
			config.Log.Fatalf("Cannot parse project channel %s", a.ProjectChannel)
		}

		chat := tb.Chat{ID: int64(id)}
		messageToSend := tb.Message{Text: message}
		selectorAlert.Inline(selectorAlert.Row(selQP, selQU))
		_, err = bot.Send(&chat, messageToSend.Text)
	case "chatops":
		config.Log.Infof("Sending 'chatops' type message")
		id, err := strconv.Atoi(a.ProjectChannel)
		if err != nil {
			config.Log.Fatalf("Cannot parse project channel %s", a.ProjectChannel)
		}

		user := tb.Chat{ID: int64(id)}
		messageToSend := tb.Message{Text: message}
		_, err = bot.Send(&user, QuoteMeta(messageToSend.Text), optionsChatops, menu)
	default:
		config.Log.Infof("Sending 'default' type message")
		id, err := strconv.Atoi(a.ProjectChannel)
		if err != nil {
			config.Log.Fatalf("Cannot parse project channel %s", a.ProjectChannel)
		}

		user := tb.Chat{ID: int64(id)}
		messageToSend := tb.Message{Text: message}
		_, err = bot.Send(&user, messageToSend.Text, options, menu)
	}

	if err != nil {
		config.Log.Errorf("SendTgMessage error: %v", err)
	} else {
		config.Log.Debugf("sendTgMessage success")
		a.AddAlertMetricNonCritical()
	}
	return err
}

func (t Telegram) InitBot(ch chan bool, wg *sync.WaitGroup) {

	var verbosity bool

	a, err := GetCommandChannel()
	if err != nil {
		config.Log.Errorf("GetCommandChannel error: %s", err)
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

	bot.Handle("/qa", qaHandler)
	bot.Handle("/la", laHandler)
	bot.Handle("/qp", qpHandler)
	bot.Handle("/lp", lpHandler)
	bot.Handle("/qu", quHandler)
	bot.Handle("/lu", luHandler)
	bot.Handle("/stats", statsHandler)
	bot.Handle("/version", versionHandler)

	bot.Handle(&btnHelp, func(c tb.Context) error {
		config.Log.Infof("Help pressed")
		SendChatOps(fmt.Sprintf("@" + c.Sender().Username + "\n\n" + help))
		return nil
	})
	bot.Handle(&btnList, func(c tb.Context) error {
		config.Log.Infof("List pressed")
		list := reports.List()
		if len(list) > 350 {
			list = "List is too long for message, use CLI/Web"
		} else {
			SendChatOps(fmt.Sprintf("@" + c.Sender().Username + "\n\n" + list))
		}
		return nil
	})
	bot.Handle(&btnStats, func(c tb.Context) error {
		config.Log.Infof("Stats pressed")
		err = statsHandler(c)
		if err != nil {
			config.Log.Errorf("/stats button handler error: %s", err.Error())
		}
		return err
	})
	bot.Handle(&btnQA, func(c tb.Context) error {
		config.Log.Infof("QA pressed")
		err = qaHandler(c)
		if err != nil {
			config.Log.Errorf("/qa button handler error: %s", err.Error())
		}
		return err
	})
	bot.Handle(&btnLA, func(c tb.Context) error {
		config.Log.Infof("LA pressed")
		err = laHandler(c)
		if err != nil {
			config.Log.Errorf("/la button handler error: %s", err.Error())
		}
		return err
	})

	//On inline button pressed (callback)
	bot.Handle(&selQU, func(c tb.Context) error {
		config.Log.Infof("PU pressed")
		err = quHandler(c)
		err = bot.Respond(c.Callback(), &tb.CallbackResponse{Text: "trying"})
		return nil
	})

	// On inline button pressed (callback)
	bot.Handle(&selQP, func(c tb.Context) error {
		config.Log.Infof("PP pressed")
		err = qpHandler(c)
		err = bot.Respond(c.Callback(), &tb.CallbackResponse{Text: "trying"})
		return nil
	})

	go func() {
		var message string
		config.Log.Debugf("Internal status is: %s", config.InternalStatus)
		switch config.InternalStatus {
		case "reload":
			message = "Bot config reloaded"
			config.Log.Warn(message)
		default:
			if config.Config.Defaults.BotGreetingEnabled {
				message = fmt.Sprintf("Bot at your service (%s, %s, %s)", config.Version, config.VersionSHA, config.VersionBuild)
				config.Log.Warn(message)
				SendChatOps(message)
			}
		}
		config.Log.Infof("Start listening telegram bots routine")
		bot.Start()
		if config.InternalStatus == "stop" {
			SendChatOps("Bot is stopped")
		}
	}()

	<-ch
	bot.Stop()
	config.Log.Infof("Exit listening telegram bots")
	// let bot to actually stop
	time.Sleep(5 * time.Second)
}
