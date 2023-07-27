package go_telegram

import (
	"context"
	tele "gopkg.in/telebot.v3"
	"my/checker/config"
	"os"
	"os/signal"
	"strconv"
	"sync"
)

var (
	options tele.SendOptions
)

func NewAlerter(a config.TAlert) *TTelegramAlerter {
	channelID, _ := strconv.ParseInt(a.ProjectChannel, 10, 64)
	criticalChannelID, _ := strconv.ParseInt(a.CriticalChannel, 10, 64)

	return &TTelegramAlerter{
		Token:             a.BotToken,
		Log:               logger,
		channelID:         channelID,
		criticalChannelID: criticalChannelID,
	}
}

func (a *TTelegramAlerter) IsBot() bool {
	return true
}

func (a *TTelegramAlerter) Init() {
	//a.Log.Info("TTelegramAlerter Init")

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	a.context = &ctx

	a.settings = tele.Settings{
		Token:  a.Token,
		Poller: &tele.LongPoller{Timeout: 10},
	}

	err := error(nil)
	a.bot, err = tele.NewBot(a.settings)
	if err != nil {
		logger.Fatalf("Error creating bot: %v", err)
	}

	options = tele.SendOptions{ParseMode: "MarkDownV2"}
}

func (a *TTelegramAlerter) Start(wg *sync.WaitGroup) {
	//a.Log.Infof("TTelegramAlerter Start, %+v", a.Bot)
	if a.bot == nil {
		a.Init()
	}
	wg.Add(1)
	defer wg.Done()
	logger.Infof("Starting tg bot")
	a.bot.Start()
}

func (a *TTelegramAlerter) Stop(wg *sync.WaitGroup) {
	_, err := a.bot.Close()
	wg.Done()
	if err != nil {
		return
	}
}

func (a *TTelegramAlerter) Send(message string) {
	//a.Log.Info("TTelegramAlerter Send")
	if a.bot == nil {
		a.Init()
	}

	_, err := a.bot.Send(&tele.Chat{ID: a.channelID}, message)
	if err != nil {
		if err != nil {
			logger.Errorf("error sending message: %s", err)
		}
	}
}

func (a *TTelegramAlerter) SendCritical(message string) {
	//a.Log.Info("TTelegramAlerter Send")
	if a.bot == nil {
		a.Init()
	}

	_, err := a.bot.Send(&tele.Chat{ID: a.criticalChannelID}, message)
	if err != nil {
		if err != nil {
			logger.Errorf("error sending message: %s", err)
		}
	}
}
