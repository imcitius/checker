package main

import (
	"fmt"
	"regexp"
)

func ceaseProject(projectName string) error {
	var done bool
	for i, project := range CheckData.Projects {
		if project.Name == projectName {
			fmt.Printf("Project: %v, Initial mode: %v,", project.Name, CheckData.Projects[i].Mode)
			CheckData.Projects[i].Mode = "quiet"
			fmt.Printf("Current mode: %v\n", CheckData.Projects[i].Mode)
			done = true
		}
	}

	if !done {
		return fmt.Errorf("Project not found trying to cease: %v", projectName)

	}
	return nil
}

func enableProject(projectName string) error {
	var done bool
	for i, project := range CheckData.Projects {
		if project.Name == projectName {
			CheckData.Projects[i].Mode = "loud"
			done = true
			fmt.Printf("Project: %v, Current mode: %v\n", project.Name, CheckData.Projects[i].Mode)
		}
	}

	if !done {
		return fmt.Errorf("Project not found trying to enable: %v", projectName)

	}
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
}
