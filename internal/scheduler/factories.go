package scheduler

import (
	"fmt"

	"checker/internal/actors"
	"checker/internal/checks"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

// CheckerFactory creates Checker instances based on the CheckDefinition.
// Delegates to checks.CheckerFactory — kept here for backward compatibility.
func CheckerFactory(checkDef models.CheckDefinition, logger *logrus.Entry) checks.Checker {
	return checks.CheckerFactory(checkDef, logger)
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
	case "webhook":
		logger.Debugf("Creating Webhook actor")
		config, ok := checkDef.ActorConfig.(*models.WebhookConfig)
		if !ok || config == nil {
			return nil, fmt.Errorf("webhook actor config is missing or invalid")
		}

		return &actors.WebhookActor{
			URL:     config.URL,
			Method:  config.Method,
			Payload: config.Payload,
			Headers: config.Headers,
			Logger:  logger,
		}, nil
	case "":
		logger.Debug("No actor type specified, skipping actor creation")
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown actor type: %s", checkDef.ActorType)
	}
}
