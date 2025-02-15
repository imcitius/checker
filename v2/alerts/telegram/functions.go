package telegram

import (
	"context"
	"my/checker/models"
	"my/checker/store"
	"sync"

	tele "gopkg.in/telebot.v3"
)

var (
	replyMarkup = tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	commandOptions = tele.SendOptions{
		ParseMode:   "MarkdownV2",
		ReplyMarkup: &replyMarkup,
	}

	alertReplyMarkup = tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	alertOptions = tele.SendOptions{
		ParseMode:   "MarkdownV2",
		ReplyMarkup: &alertReplyMarkup,
	}

	// Reply buttons.
	//btnHelp     = replyMarkup.Text("ℹ Help")
	//btnSettings = replyMarkup.Text("⚙ Settings")

	// Inline buttons.
	//
	// Pressing it will cause the client to
	// send the bot a callback.
	//
	// Make sure Unique stays unique as per button kind
	// since it's required for callback routing to work.
	//

	qProject = alertReplyMarkup.Data("Quiet project", "qProject")
	qUUID    = alertReplyMarkup.Data("Quiet UUID", "qUUID")

	messagesContext = store.GetMessagesContextStorage()
)

func (a *TTelegramAlerter) IsBot() bool {
	return true
}

func (a *TTelegramAlerter) Init(ctx context.Context) {
	//a.Log.Info("TTelegramAlerter Init")

	a.settings = tele.Settings{
		Token:  a.Token,
		Poller: &tele.LongPoller{Timeout: 10},
	}
	//a.options = options

	err := error(nil)
	a.bot, err = tele.NewBot(a.settings)
	if err != nil {
		logger.Fatalf("Error creating bot: %v", err)
	}

	a.bot.Handle("/qa", qaHandler)
	//a.bot.Handle("/la", laHandler)
	a.bot.Handle("/qp", qpHandler)
	//a.bot.Handle("/lp", lpHandler)
	a.bot.Handle("/qu", quHandler)
	//a.bot.Handle("/lu", luHandler)
	//a.bot.Handle("/stats", statsHandler)
	//a.bot.Handle("/version", versionHandler)

	replyMarkup.Inline(
		replyMarkup.Row(qProject, qUUID),
	)

	a.bot.Handle("/start", func(c tele.Context) error {
		a.Reply(c, "Hello\\!")
		return nil
	})

	// On reply button pressed (message)
	//a.bot.Handle(&btnHelp, func(c tele.Context) error {
	//	return c.Alert("Here is some help: ...")
	//})

	a.bot.Handle(&qProject, qpHandler)
	a.bot.Handle(&qUUID, quHandler)
}

func (a *TTelegramAlerter) Start(ctx context.Context, wg *sync.WaitGroup) {
	if a.bot == nil {
		a.Init(ctx)
	}
	wg.Add(1)
	defer wg.Done()
	logger.Infof("Starting tg bot")

	a.Alert(ctx, models.TAlertDetails{Severity: "info", Message: "Starting tg bot"})
	a.bot.Start()
}

func (a *TTelegramAlerter) Stop(wg *sync.WaitGroup) {
	_, err := a.bot.Close()
	wg.Done()
	if err != nil {
		return
	}
}

func (a *TTelegramAlerter) Alert(ctx context.Context, alertDetails models.TAlertDetails) {
	m, err := a.Send(ctx, alertDetails)
	if err != nil {
		logger.Errorf("error sending noncritical message: %s", err)
		return
	}
	messagesContext.Update(m)
}

func (a *TTelegramAlerter) Send(ctx context.Context, alertDetails models.TAlertDetails) (*tele.Message, error) {
	if a.bot == nil {
		a.Init(ctx)
	}
	m, err := a.bot.Send(&tele.Chat{ID: a.channelID}, alertDetails.Message, &alertOptions)
	if err != nil {
		logger.Errorf("error sending message: %s", err)
		return nil, err
	}
	return m, err
}

func (a *TTelegramAlerter) AlertCritical(ctx context.Context, alertDetails models.TAlertDetails) {
	m, err := a.SendCritical(ctx, alertDetails)
	if err != nil {
		logger.Errorf("error sending critical message: %s", err)
		return
	}
	messagesContext.Update(m)
}

func (a *TTelegramAlerter) SendCritical(ctx context.Context, alertDetails models.TAlertDetails) (*tele.Message, error) {
	if a.bot == nil {
		a.Init(ctx)
	}
	m, err := a.bot.Send(&tele.Chat{ID: a.criticalChannelID}, alertDetails.Message, replyMarkup, alertOptions)
	if err != nil {
		logger.Errorf("error sending message: %s", err)
		return nil, err
	}
	return m, err
}

func (a *TTelegramAlerter) SendCommand(ctx context.Context, message string) {
	//a.Log.Info("TTelegramAlerter Alert")
	if a.bot == nil {
		a.Init(ctx)
	}

	_, err := a.bot.Send(&tele.Chat{ID: a.channelID}, message, &commandOptions)
	if err != nil {
		if err != nil {
			logger.Errorf("error sending message: %s", err)
		}
	}
}

func (a *TTelegramAlerter) Reply(c tele.Context, message string) {
	_, err := a.bot.Reply(c.Message(), message, &commandOptions)

	if err != nil {
		if err != nil {
			logger.Errorf("error sending message: %s", err)
		}
	}
}
