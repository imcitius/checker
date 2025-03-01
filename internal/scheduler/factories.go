package scheduler

import (
	"checker/internal/actors"
	"checker/internal/checks"
	"checker/internal/config"
	"github.com/sirupsen/logrus"
)

// CheckerFactory creates Checker instances based on the CheckConfig.
func CheckerFactory(cfg config.CheckConfig, logger *logrus.Entry) checks.Checker {
	switch cfg.Type {
	case "http":
		return &checks.HTTPCheck{
			URL:                 cfg.URL,
			Timeout:             cfg.Timeout,
			Answer:              cfg.Answer,
			Code:                cfg.Code,
			Auth:                struct{ User, Password string }{cfg.Auth.User, cfg.Auth.Password},
			Headers:             cfg.Headers,
			Cookies:             cfg.Cookies,
			SkipCheckSSL:        cfg.SkipCheckSSL,
			SSLExpirationPeriod: cfg.SSLExpirationPeriod,
			StopFollowRedirects: cfg.StopFollowRedirects,
			Logger:              logger,
		}
	case "tcp":
		return &checks.TCPCheck{
			Host:    cfg.Host,
			Port:    cfg.Port,
			Timeout: cfg.Timeout,
		}
	case "ping":
		return &checks.PingCheck{
			Host:    cfg.Host,
			Count:   cfg.Count,
			Timeout: cfg.Timeout,
		}
	default:
		logrus.Warnf("Unknown check type: %s", cfg.Type)
		return nil
	}
}

func ActorFactory(cfg config.CheckConfig) actors.Actor {
	switch cfg.ActorType {
	case "log":
		return &actors.LogActor{}
	//case "alert":
	//	switch cfg.AlertType {
	//	case "telegram":
	//		return &actors.Telegram{}
	//	case "slack":
	//		return &actors.Slack{}
	//	}

	default:
		logrus.Warnf("Unknown actor type: %s", cfg.Type)
		return nil
	}
}
