package main

import (
	"fmt"
	"regexp"

	tb "gopkg.in/tucnak/telebot.v2"
)

// TgMessage - new type to define own methods
type TgMessage struct {
	*tb.Message
}

func (p project) CeaseAlerts() error {
	Runtime.AlertFlags.ByProject[p.Name] = "quiet"
	return nil
}

func (p project) EnableAlerts() error {
	Runtime.AlertFlags.ByProject[p.Name] = "loud"
	return nil
}

func ceaseUUID(uuID string) error {
	Runtime.AlertFlags.ByUUID[uuID] = "quiet"
	return nil
}

func enableUUID(uuID string) error {
	Runtime.AlertFlags.ByUUID[uuID] = "loud"
	return nil
}

func (m *TgMessage) GetProject() string {
	var projectName string

	fmt.Printf("message: %v\n", m)
	pattern := regexp.MustCompile("Project: (.*)\n")
	result := pattern.FindStringSubmatch(m.ReplyTo.Text)
	if result == nil {
		fmt.Printf("Project extraction error.")
	} else {
		fmt.Printf("Project extracted: %v\n", result[1])
		projectName = result[1]
	}

	return projectName
}

func (m *TgMessage) GetUUID() string {
	var uuid string

	fmt.Printf("message: %v\n", m)
	pattern := regexp.MustCompile("UUID: (.*)")
	result := pattern.FindStringSubmatch(m.ReplyTo.Text)
	if result == nil {
		fmt.Printf("UUID extraction error.")
	} else {
		fmt.Printf("UUID extracted: %v\n", result[1])
		uuid = result[1]
	}

	return uuid

	// WIP test and write error handling
}
