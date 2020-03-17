package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func checkProjects(token string) {
	for _, project := range CheckData.Projects {
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
				postChannel(project.ProjectChannel, CheckData.BotToken, message)
			}
		}
	}

}

func urlChecks(token string) {
	for {
		ticker := time.NewTicker(time.Duration(CheckData.Secs) * time.Second)
		done := make(chan bool)

		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)
				checkProjects(token)
			}
		}
	}
}
