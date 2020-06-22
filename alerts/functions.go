package alerts

import (
	"fmt"
	checks "my/checker/checks"
	"my/checker/config"
	"my/checker/metrics"
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

func GetAlertProto(a *config.AlertConfigs) Alerter {
	return AlerterCollection[a.Type]
}

func GetCommandChannel() (*config.AlertConfigs, error) {
	for _, a := range config.Config.Alerts {
		if a.Name == config.Config.Defaults.Parameters.CommandChannel {
			config.Log.Debugf("Found command channel: %v", a.ProjectChannel)
			return &a, nil
		}
	}
	return nil, fmt.Errorf("ChatOpsChannel not found: %s", config.Config.Defaults.Parameters.CommandChannel)
}

func GetProjectChannel(p *config.Project) *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.AlertChannel {
			return &a
		}
	}
	return nil
}

func GetCritChannel(p *config.Project) *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.CritAlertChannel {
			return &a
		}
	}
	return nil
}

func Send(p *config.Project, text string) {
	metrics.AddProjectMetricChatOpsAnswer(p)

	err := Alert(GetProjectChannel(p), text)
	if err != nil {
		config.Log.Infof("Send alert error for project %s: %s", p.Name, err)
	}
}

func SendCrit(p *config.Project, text string) {
	metrics.AddProjectMetricCriticalAlert(p)
	metrics.AddAlertMetricCritical(GetCritChannel(p))

	err := Alert(GetCritChannel(p), text)
	if err != nil {
		config.Log.Infof("Send critical alert error for project %s: %s", p.Name, err)
	}
}

func SendChatOps(text string) {
	metrics.AddProjectMetricChatOpsAnswer(&config.Project{
		Name: "ChatOps"})
	commandChannel, err := GetCommandChannel()
	if err != nil {
		config.Log.Infof("GetCommandChannel error: %s", err)
	}

	err = Alert(commandChannel, text)
	if err != nil {
		config.Log.Infof("SendTgChatOpsMessage error: %s", err)
	}
}

func Alert(a *config.AlertConfigs, text string) error {
	metrics.AddAlertMetricNonCritical(a)

	err := GetAlertProto(a).Send(a, text)
	if err != nil {
		config.Log.Infof("Send error")
	}
	return err
}
