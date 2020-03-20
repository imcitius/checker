package main

import (
	"fmt"
	"regexp"
)

func ceaseProject(projectName string) error {
	var done bool
	for i, project := range Config.Projects {
		if project.Name == projectName {
			fmt.Printf("Project: %v, Initial mode: %v,", project.Name, Config.Projects[i].Parameters.Mode)
			Config.Projects[i].Parameters.Mode = "quiet"
			fmt.Printf("Current mode: %v\n", Config.Projects[i].Parameters.Mode)
			done = true
		}
	}

	if !done {
		return fmt.Errorf("Project not found trying to cease: %v", projectName)

	}
	// WIP test and write error handling
	return nil
}

func enableProject(projectName string) error {
	var done bool
	for i, project := range Config.Projects {
		if project.Name == projectName {
			fmt.Printf("Project: %v, Initial mode: %v,", project.Name, Config.Projects[i].Parameters.Mode)
			Config.Projects[i].Parameters.Mode = "loud"
			fmt.Printf("Current mode: %v\n", Config.Projects[i].Parameters.Mode)
			done = true
		}
	}

	if !done {
		return fmt.Errorf("Project not found trying to enable: %v", projectName)
	}
	// WIP test and write error handling
	return nil
}

func ceaseUUID(uuID string) error {
	var done bool
	for i := range Config.Projects {
		for j := range Config.Projects[i].URLChecks {
			if Config.Projects[i].URLChecks[j].uuID == uuID {
				fmt.Printf("Project: %v, Initial mode: %v,", Config.Projects[i].Name, Config.Projects[i].URLChecks[j].Mode)
				Config.Projects[i].URLChecks[j].Mode = "quiet"
				fmt.Printf("Current mode: %v\n", Config.Projects[i].URLChecks[j].Mode)
				done = true
			}
		}
	}

	if !done {
		return fmt.Errorf("UUID not found trying to cease: %v", uuID)
	}
	// WIP test and write error handling
	return nil
}

func enableUUID(uuID string) error {
	var done bool
	for i := range Config.Projects {
		for j := range Config.Projects[i].URLChecks {
			if Config.Projects[i].URLChecks[j].uuID == uuID {
				fmt.Printf("Project: %v, Initial mode: %v,", Config.Projects[i].Name, Config.Projects[i].URLChecks[j].Mode)
				Config.Projects[i].URLChecks[j].Mode = "loud"
				fmt.Printf("Current mode: %v\n", Config.Projects[i].URLChecks[j].Mode)
				done = true
			}
		}
	}

	if !done {
		return fmt.Errorf("UUID not found trying to enable: %v", uuID)
	}
	// WIP test and write error handling
	return nil
}

func extractProject(message []byte) string {

	fmt.Printf("result: %v\n", string(message))

	pattern := regexp.MustCompile(`Project: (.*);.*`)
	template := []byte("$1")
	result := []byte{}

	for _, submatches := range pattern.FindAllSubmatchIndex(message, -1) {
		result = pattern.Expand(result, template, message, submatches)
	}
	fmt.Printf("result: %v\n", result)
	return string(result)

	// WIP test and write error handling

}

func extractUUID(message []byte) string {

	fmt.Printf("result: %v\n", string(message))

	pattern := regexp.MustCompile(`.*UUID: (.*);`)
	template := []byte("$1")
	result := []byte{}

	for _, submatches := range pattern.FindAllSubmatchIndex(message, -1) {
		result = pattern.Expand(result, template, message, submatches)
	}
	fmt.Printf("result: %v\n", result)
	return string(result)

	// WIP test and write error handling
}
