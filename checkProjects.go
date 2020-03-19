package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func checkProjects(timeout int) {

	for _, project := range Config.Projects {

		if project.Parameters.RunEvery == timeout {

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

					if Config.Defaults.Parameters.Mode == "loud" && project.Parameters.Mode == "loud" {
						log.Printf("Loud mode:\n%v\n", message)
						postChannel(project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
					} else {
						log.Printf("Quiet mode:\n%v\n", message)
					}
				}
			}
		}
	}
}
