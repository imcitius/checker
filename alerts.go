package main

import (
	"log"
)

// TgAlert - encapsulated into external structure to be able to receive new methods
type TgAlert struct {
	Message string
}

// SendNonCrit - method sends alert to non-critical channel, if project and check level configs permits
func (a TgAlert) SendNonCrit(project project, check UniversalCheck) error {
	uuID := check.UUID()

	if Config.Defaults.Parameters.Mode == "loud" && Runtime.AlertFlags.ByProject[project.Name] == "loud" {
		log.Printf("Project Loud mode,")
		if Runtime.AlertFlags.ByUUID[uuID] != "quiet" {
			log.Printf("Check Loud mode:\n%v\n", a)
			// log.Printf("Ask to send alert: channel %d, token %s, message %s", project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
			sendAlert(project.Parameters.ProjectChannel, project.Parameters.BotToken, a.Message)
		} else {
			log.Printf("Check Quiet mode:\n%v\n", a)
		}
	} else {
		log.Printf("Project Quiet mode:\n%v\n", a)
	}

	return nil
}

// SendCrit - method sends alert to critical channel
func (a TgAlert) SendCrit(project project) error {
	sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, a.Message)
	return nil
}
