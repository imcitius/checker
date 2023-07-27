package alerts

import (
	"my/checker/alerts/log"
	telegram "my/checker/alerts/telegram"
	"my/checker/config"
)

func initAlerters() error {
	alerters = new(TAlertersCollection)
	alerters.Alerters = make(map[string]ICommonAlerter)

	if len(configurer.Alerts) > 0 {
		for k, a := range configurer.Alerts {
			alerter := newAlerter(a)
			if alerter == nil {
				logger.Fatalf("Alerter %s not found", a.Type)
			}
			alerter.Init()
			if alerter.IsBot() {
				go alerter.Start(wg)
			}
			alerters.Alerters[k] = alerter
		}
	}

	alerters.Alerters["log"] = newAlerter(config.TAlert{
		Type: "log",
	})

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

func newAlerter(a config.TAlert) ICommonAlerter {
	switch a.Type {
	case "telegram":
		return telegram.NewAlerter(a)
	case "log":
		return log.NewAlerter(logger)
	default:
		return nil
	}
}
