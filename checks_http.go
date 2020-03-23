package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

func runHTTPCheck(project project) {
	var (
		healthy       uint
		checkNum      uint
		failedChecks  []string
		answerPresent bool
	)
	// set default
	answerPresent = true
	projectFails := Runtime.Fails.HTTP[project.Name]

	// check project urls
	for _, check := range project.Checks.URLChecks {
		fmt.Println("test: ", check.URL)
		_, err := url.Parse(check.URL)
		if err != nil {
			log.Fatal(err)
		}
		checkNum++

		client := &http.Client{}
		req, err := http.NewRequest("GET", check.URL, nil)

		// if custom headers requested
		if check.Headers != nil {
			for _, headers := range check.Headers {
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
		code := check.Code == uint(response.StatusCode)

		answer, err := regexp.Match(check.Answer, buf.Bytes())

		// check answer_present condition
		if check.AnswerPresent == "absent" {
			answerPresent = false
		}
		answerGood := answer == answerPresent
		// log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, urlcheck.AnswerPresent, answerGood)

		if code && answerGood {
			healthy++
			continue
		} else {
			fmt.Printf("The HTTP request %v failed with error %d\n", check.URL, response.StatusCode)
			message := nonCriticalHTTP(project.Name, check.URL, check.uuID, response.StatusCode)

			if Config.Defaults.Parameters.Mode == "loud" && Runtime.Alerts.Project[project.Name] == "loud" && Runtime.Alerts.UUID[check.uuID] != "quiet" {
				log.Printf("Project Loud mode,")
				if check.Mode != "quiet" {
					log.Printf("Check Loud mode:\n%v\n", message)
					sendAlert(project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
				} else {
					log.Printf("Check Quiet mode:\n%v\n", message)
				}
			} else {
				log.Printf("Project Quiet mode:\n%v\n", message)
			}
			failedChecks = append(failedChecks, fmt.Sprintf("{Url: %s, code %d}\n", check.URL, response.StatusCode))
		}

		// fmt.Printf("Time: %v\nTimeout: %v\nProject: %v\n\n", time.Now(), timeout, project.Name)
		// fmt.Printf("%v\n", project.URLChecks)

		fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)
		if healthy >= project.Parameters.MinHealth {
			if projectFails > 0 {
				projectFails--
			}
			continue
		} else {
			if project.Parameters.AllowFails > projectFails {
				projectFails++
				continue
			} else {
				message := criticalHTTP(project.Name, healthy, checkNum, project.Parameters.MinHealth, failedChecks)
				sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, message)
			}
		}
	}
}
