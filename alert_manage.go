package main

import (
	"fmt"
	"regexp"
)

func ceaseProject(projectName string) error {
	Runtime.Alerts.Project[projectName] = "quiet"
	return nil
}

func enableProject(projectName string) error {
	Runtime.Alerts.Project[projectName] = "loud"
	return nil
}

func ceaseUUID(uuID string) error {
	Runtime.Alerts.UUID[uuID] = "quiet"
	return nil
}

func enableUUID(uuID string) error {
	Runtime.Alerts.UUID[uuID] = "loud"
	return nil
}

func extractProject(message string) string {
	var projectName string

	fmt.Printf("message: %v\n", message)
	pattern := regexp.MustCompile("Project: (.*)\n")
	result := pattern.FindStringSubmatch(message)
	if result == nil {
		fmt.Printf("Project extraction error.")
	} else {
		fmt.Printf("Project extracted: %v\n", result[1])
		projectName = result[1]
	}

	return projectName
}

func extractUUID(message string) string {
	var uuid string

	fmt.Printf("message: %v\n", message)
	pattern := regexp.MustCompile("UUID: (.*)")
	result := pattern.FindStringSubmatch(message)
	if result == nil {
		fmt.Printf("UUID extraction error.")
	} else {
		fmt.Printf("UUID extracted: %v\n", result[1])
		uuid = result[1]
	}

	return uuid

	// WIP test and write error handling
}
