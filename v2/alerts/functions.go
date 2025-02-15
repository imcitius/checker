package alerts

import (
	"context"
	"errors"
	"my/checker/alerts/log"
	telegram "my/checker/alerts/telegram"
	"my/checker/models"
)

func initAlerters(ctx context.Context) error {
	alerters = new(TAlertersCollection)
	alerters.Alerters = make(map[string]ICommonAlerter)

	alerts := configurer.GetAlerts()
	if len(alerts) > 0 {
		for _, a := range alerts {
			alerter, err := newAlerter(a)
			if err != nil {
				logger.Fatalf("Alerter %s not found", a.Type)
			}
			alerter.Init(ctx)
			if alerter.IsBot() {
				go alerter.Start(ctx, wg)
			}
			alerters.Alerters[a.Name] = alerter
		}
	}

	var err error
	alerters.Alerters["log"], err = newAlerter(models.TAlert{
		Type: "log",
	})
	if err != nil {
		logger.Fatalf("Alerter %s not found", "log")
	}

	return nil
}

//func GetAlerter(check config.TCheckConfig) ICommonAlerter {
//	project, err := config.GetProjectByName(check.Project)
//	if err != nil {
//		logger.Infof("Project %s not found", check.Project)
//	}
//	alerter, err := GetAlerterByName(project.Parameters.AlerterName)
//	return alerter
//}
//

func GetAlerterByName(name string) ICommonAlerter {
	return alerters.getByName(name)
}

func (c *TAlertersCollection) getByName(name string) ICommonAlerter {
	return c.Alerters[name]
}

func newAlerter(a models.TAlert) (ICommonAlerter, error) {
	switch a.Type {
	case "telegram":
		return telegram.NewAlerter(a), nil
	case "log":
		return log.NewAlerter(logger), nil
	default:
		return nil, errors.New("unknown alerter type or type not specified")
	}
}
