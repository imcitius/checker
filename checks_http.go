package main

import (
	"fmt"
	"net/http"
)

func runHTTPCheck(project project) {
	var (
		failedChecks []string
		// answerGood   bool
		healthy      uint
		projectFails uint = Runtime.Fails.HTTP[project.Name]
		alert        TgAlert
	)

	for _, check := range project.Checks.URLChecks {

		resp, err := check.Execute()
		if err != nil {
			Runtime.Fails.HTTP[check.UUID()]++
			response := resp.(*http.Response)
			// log.Printf("Error execute http check: %+v (response: %+v)", err, response)
			fmt.Printf("The HTTP request %v failed with error %d\n", check.URL, response.StatusCode)
			// log.Println(response)
			alert.Message = nonCriticalHTTP(project.Name, check.URL, check.uuID, response.StatusCode)
			alert.SendNonCrit(project, check)

			failedChecks = append(failedChecks, fmt.Sprintf("{Url: %s, code %d}\n", check.URL, response.StatusCode))

		} else {
			healthy++
			continue
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
				// message := criticalHTTP(project.Name, healthy, checkNum, project.Parameters.MinHealth, failedChecks)
				// sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, message)
			}
		}
	}
}
