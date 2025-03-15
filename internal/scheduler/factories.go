package scheduler

import (
	"fmt"

	"checker/internal/actors"
	"checker/internal/checks"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

// CheckerFactory creates Checker instances based on the CheckDefinition.
// Returns nil and logs a warning if the check type is unknown.
func CheckerFactory(checkDef models.CheckDefinition, logger *logrus.Entry) checks.Checker {
	if logger == nil {
		logger = logrus.WithField("function", "CheckerFactory")
	}

	switch checkDef.Type {
	case "http":
		logger.Debugf("Creating HTTP check for URL: %s", checkDef.URL)
		return &checks.HTTPCheck{
			URL:                 checkDef.URL,
			Timeout:             checkDef.Timeout,
			Answer:              checkDef.Answer,
			Code:                checkDef.Code,
			Auth:                struct{ User, Password string }{checkDef.Auth.User, checkDef.Auth.Password},
			Headers:             checkDef.Headers,
			SkipCheckSSL:        checkDef.SkipCheckSSL,
			SSLExpirationPeriod: checkDef.SSLExpirationPeriod,
			StopFollowRedirects: checkDef.StopFollowRedirects,
			Logger:              logger,
		}
	case "tcp":
		logger.Debugf("Creating TCP check for host: %s, port: %d", checkDef.Host, checkDef.Port)
		return &checks.TCPCheck{
			Host:    checkDef.Host,
			Port:    checkDef.Port,
			Timeout: checkDef.Timeout,
		}
	case "icmp":
		logger.Debugf("Creating ICMP check for host: %s", checkDef.Host)
		return &checks.ICMPCheck{
			Host:    checkDef.Host,
			Count:   checkDef.Count,
			Timeout: checkDef.Timeout,
		}
	case "passive":
		logger.Debugf("Creating Passive check for host: %s", checkDef.Host)
		return &checks.PassiveCheck{
			Timeout: checkDef.Timeout,
		}

	default:
		logger.Warnf("Unknown check type: %s", checkDef.Type)
		return nil
	}
}

// ActorFactory creates Actor instances based on the CheckDefinition.
// Returns nil and logs a warning if the actor type is unknown.
func ActorFactory(checkDef models.CheckDefinition) (actors.Actor, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function":  "ActorFactory",
		"actorType": checkDef.ActorType,
		"uuid":      checkDef.UUID,
	})

	switch checkDef.ActorType {
	case "log":
		logger.Debugf("Creating Log actor")
		return &actors.LogActor{}, nil
	case "alert":
		switch checkDef.AlertType {
		case "telegram":
			logger.Debugf("Creating Telegram actor")
			// Ensure Telegram is properly implemented in the actors package
			// return &actors.Telegram{
			//     Destination: checkDef.AlertDestination,
			// }, nil
			return nil, fmt.Errorf("telegram actor not yet implemented")
		case "slack":
			logger.Debugf("Creating Slack actor")
			// Ensure Slack is properly implemented in the actors package
			// return &actors.Slack{
			//     Webhook: checkDef.AlertDestination,
			// }, nil
			return nil, fmt.Errorf("slack actor not yet implemented")
		default:
			return nil, fmt.Errorf("unknown alert type: %s", checkDef.AlertType)
		}
	case "":
		logger.Debug("No actor type specified, skipping actor creation")
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown actor type: %s", checkDef.ActorType)
	}
}
