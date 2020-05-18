package main

import (
	"errors"
	"fmt"
)

func (a *AlertConfigs) Send(alerttype string, e error) error {

	switch a.Type {
	case "telegram":
		err := sendTgMessage(alerttype, a, e)
		return err
	default:
		err := errors.New(fmt.Sprintf("Not implemented bot type %s, name %s", a.Type, a.Name))
		return err
	}
}

func (a *AlertConfigs) GetName() string {
	return a.Name
}

func (a *AlertConfigs) GetType() string {
	return a.Type
}

func (a *AlertConfigs) GetCreds() string {
	return a.BotToken
}

func addAlertCounter(alerttype string, a *AlertConfigs) {
	log.Debugf("increase alert counter")

	for _, counter := range Config.Alerts {
		log.Debugf("%s .... %s", a.Name, counter.Name)
		if a.Name == counter.Name {
			log.Debugf("Increase total %s counter %s", alerttype, counter.Name)
			switch alerttype {
			case "crit":
				counter.Critical++
			case "noncrit":
				counter.NonCritical++
			case "report":
				counter.NonCritical++
			default:
				log.Errorf("Undefined alert type")
			}
			counter.AlertCount++
		}
	}
}
