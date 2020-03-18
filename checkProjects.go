package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func checkProjects(timeout int) {

	for _, project := range CheckData.Projects {

		if project.RunEvery == timeout {

			fmt.Printf("Time: %v\nTimeout: %v\nProject: %v\n\n", time.Now(), timeout, project.Name)

			for _, uri := range project.Urlchecks {
				fmt.Println("test: ", uri)
				_, err := url.Parse(uri)
				if err != nil {
					log.Fatal(err)
				}

				response, err := http.Get(uri)

				switch code := response.StatusCode; {
				case code == 200:
					continue
				default:
					fmt.Printf("The HTTP request %v failed with error %d\n", uri, response.StatusCode)
					message := fmt.Sprintf("Project: %v;\nURL: %s\nError code: %d\n", project.Name, uri, response.StatusCode)

					if CheckData.Defaults.Mode == "loud" && project.Mode == "loud" {
						log.Printf("Loud mode: %v\n", message)
						postChannel(project.ProjectChannel, project.BotToken, message)
					} else {
						log.Printf("Quiet mode: %v\n", message)
					}
				}
			}
		}
	}
}
