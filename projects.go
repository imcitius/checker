package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func checkProjects() {
	token := ""
	for _, project := range CheckData.Projects {
		if project.BotToken != "" {
			token = project.BotToken
		} else {
			token = CheckData.BotToken
		}
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
				fmt.Printf("The HTTP request failed with error %d\n", response.StatusCode)
				message := fmt.Sprintf("URL: %s\nError code: %d\n", uri, response.StatusCode)
				postChannel(project.ProjectChannel, token, message)
			}
		}
	}

}
