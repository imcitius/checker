package main

import (
	"errors"
	"fmt"
)

func (a Alerts) Send(e error) error {

	switch a.Type {
	case "telegram":
		err := sendTgMessage(a, e)
		return err
	default:
		err := errors.New(fmt.Sprintf("Not implemented bot type %s, name %s", a.Type, a.Name))
		return err
	}
}

func (a Alerts) GetName() string {
	return a.Name
}

func (a Alerts) GetType() string {
	return a.Type
}

func (a Alerts) GetCreds() string {
	return a.BotToken
}
