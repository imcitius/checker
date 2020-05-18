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

func addAlertCounter(a *AlertConfigs) {
	for _, counter := range AlertStats.Counters {
		if a.Name == counter.Name {
			counter.AlertCount++
		}
	}
}

func addAlertCounterNoncrit(a *AlertConfigs) {
	for _, counter := range AlertStats.Counters {
		if a.Name == counter.Name {
			counter.NonCritical++
		}
	}
}

func addAlertCounterCrit(a *AlertConfigs) {
	for _, counter := range AlertStats.Counters {
		if a.Name == counter.Name {
			counter.Critical++
		}
	}
}
