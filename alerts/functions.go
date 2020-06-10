package alerts

import (
	"fmt"
	checks "my/checker/checks"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/status"
)

func ProjectAlert(p *config.Project, e error) {
	message := e.Error()

	config.Log.Debugf("Send non-critical alert for project: '%+v', with error '%+v'\n", p.Name, e)
	//config.Log.Printf("%+v", Config.Alerts)

	if len(p.Parameters.Mentions) > 0 {
		message = "\n" + message
		for _, mention := range p.Parameters.Mentions {
			message = mention + " " + message
		}
	}
	if projects.GetMode(p) != "quiet" && status.MainStatus != "quiet" {
		Send(p, message)
	}
}

func ProjectCritAlert(p *config.Project, e error) {
	message := e.Error()

	if len(p.Parameters.Mentions) > 0 {
		message = "\n" + message
		for _, mention := range p.Parameters.Mentions {
			message = mention + " " + message
		}
	}

	config.Log.Printf("Send critical alert for project: %+v with error %+v\n\n", p, e)
	SendCrit(p, message)
}

func ProjectSendReport(p *config.Project) error {
	var (
		ceasedChecks                []string
		reportMessage, reportHeader string
	)
	for _, healthcheck := range p.Healtchecks {
		for _, check := range healthcheck.Checks {
			if check.Mode == "quiet" {
				ceasedChecks = append(ceasedChecks, checks.UUID(&check))
			}
		}
	}

	if len(ceasedChecks) > 0 {
		reportHeader = fmt.Sprintf("Project %s in %s state\n", p.Name, projects.GetMode(p))
		reportMessage = reportHeader + fmt.Sprintf("Ceased checks: %v\n", ceasedChecks)
	} else {
		if p.Parameters.Mode == "quiet" {
			reportMessage = fmt.Sprintf("Project %s in quiet state\n", p.Name)
		}
	}

	if reportMessage != "" || p.Parameters.Mode == "quiet" {

		SendChatOps(reportMessage)
	}
	return nil
}