package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func checkHTTP(timeout int) {

	var (
		healthy      int
		checkNum     int
		failedChecks []string
	)

	for i, project := range Config.Projects {

		if project.Parameters.RunEvery == timeout {

			fmt.Printf("Time: %v\nTimeout: %v\nProject: %v\n\n", time.Now(), timeout, project.Name)
			// fmt.Printf("%v\n", project.URLChecks)

			for _, urlcheck := range project.URLChecks {
				fmt.Println("test: ", urlcheck.URL)
				_, err := url.Parse(urlcheck.URL)
				if err != nil {
					log.Fatal(err)
				}
				checkNum++

				client := &http.Client{}
				req, err := http.NewRequest("GET", urlcheck.URL, nil)

				// if custom headers requested
				if urlcheck.Headers != nil {
					for _, headers := range urlcheck.Headers {
						for header, value := range headers {
							req.Header.Add(header, value)
						}
					}
				}
				// log.Printf("http request: %v", req)
				response, err := client.Do(req)

				buf := new(bytes.Buffer)
				buf.ReadFrom(response.Body)

				// check that response code is correct
				code := urlcheck.Code == response.StatusCode

				answer, err := regexp.Match(urlcheck.Answer, buf.Bytes())

				// check answer_present condition
				answerGood := answer == urlcheck.AnswerPresent
				// log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, urlcheck.AnswerPresent, answerGood)

				if code && answerGood {
					healthy++
					continue
				} else {
					fmt.Printf("The HTTP request %v failed with error %d\n", urlcheck.URL, response.StatusCode)
					message := nonCritical(project.Name, urlcheck.URL, urlcheck.uuID, response.StatusCode)

					if Config.Defaults.Parameters.Mode == "loud" && project.Parameters.Mode == "loud" {
						log.Printf("Project Loud mode,")
						if urlcheck.Mode != "quiet" {
							log.Printf("Check Loud mode:\n%v\n", message)
							sendAlert(project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
						} else {
							log.Printf("Check Quiet mode:\n%v\n", message)
						}
					} else {
						log.Printf("Project Quiet mode:\n%v\n", message)
					}
					failedChecks = append(failedChecks, fmt.Sprintf("{Url: %s, code %d}\n", urlcheck.URL, response.StatusCode))
				}
			}
		}
		// fmt.Printf("Healthy %d of %d\n", healthy, project.Parameters.MinHealth)
		if healthy < project.Parameters.MinHealth {
			Config.Projects[i].Runtime.Fails++
			fmt.Printf("Critical fails: %d on project %s\n", Config.Projects[i].Runtime.Fails, project.Name)
		} else {
			Config.Projects[i].Runtime.Fails = 0
		}
		if Config.Projects[i].Runtime.Fails > project.Parameters.AllowFails {
			message := critical(project.Name, healthy, checkNum, project.Parameters.MinHealth, failedChecks)
			sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, message)
		}
	}
}
