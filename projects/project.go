package project

import (
	"fmt"
	"my/checker/alerts"
	config "my/checker/config"
	"my/checker/metrics"
	"my/checker/status"
)

type Project struct {
	config.Project
}

func GetName(p *config.Project) string {
	return p.Name
}

func (p *Project) GetMode() string {
	return status.Statuses.Projects[p.Name].Mode
}

func (p *Project) IsLoud() bool {
	if status.Statuses.Projects[p.Name].Mode != "" {
		if status.Statuses.Projects[p.Name].Mode == "loud" {
			return true
		} else {
			return false
		}
	} else {
		if config.Config.Defaults.Parameters.Mode == "loud" {
			return true
		} else {
			return false
		}
	}
}

func (p *Project) IsQuiet() bool {
	if status.Statuses.Projects[p.Name].Mode != "" {
		if status.Statuses.Projects[p.Name].Mode == "quiet" {
			return true
		} else {
			return false
		}
	} else {
		if config.Config.Defaults.Parameters.Mode == "quiet" {
			return true
		} else {
			return false
		}
	}
}

func (p *Project) Loud() {
	status.Statuses.Projects[p.Name].Mode = "loud"
}

func (p *Project) Quiet() {
	status.Statuses.Projects[p.Name].Mode = "quiet"
}

func (p *Project) GetCritChannel() *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.CritAlertChannel {
			return &a
		}
	}
	return nil
}

func (p *Project) GetProjectChannel() *config.AlertConfigs {
	for _, a := range config.Config.Alerts {
		if a.Name == p.Parameters.AlertChannel {
			return &a
		}
	}
	return nil
}

func (p *Project) Send(text string) {
	p.AddProjectMetricChatOpsAnswer()

	err := alerts.Alert(p.GetProjectChannel(), text, "alert")
	if err != nil {
		config.Log.Infof("Send alert error for project %s: %s", p.Name, err)
	}
}

func (p *Project) SendCrit(text string) {

	critChannel := p.GetCritChannel()
	p.AddProjectMetricCriticalAlert()
	metrics.AddAlertMetricCritical(critChannel)

	err := alerts.Alert(critChannel, text, "alert")
	if err != nil {
		config.Log.Infof("Send critical alert error for project %s: %s", p.Name, err)
	}
}

func (p *Project) ProjectAlert(e error) {
	message := e.Error()

	config.Log.Debugf("Send non-critical alert for project: '%+v', with error '%+v'\n", p.Name, e)
	//config.Log.Printf("%+v", Config.Alerts)

	if len(p.Parameters.Mentions) > 0 {
		message = "\n" + message
		for _, mention := range p.Parameters.Mentions {
			message = mention + " " + message
		}
	}
	if p.IsLoud() && status.MainStatus != "quiet" {
		p.Send(message)
	}
}

func (p *Project) ProjectCritAlert(e error) {
	message := e.Error()

	if len(p.Parameters.Mentions) > 0 {
		message = "\n" + message
		for _, mention := range p.Parameters.Mentions {
			message = mention + " " + message
		}
	}

	config.Log.Printf("Send critical alert for project: %+v with error %+v\n\n", p, e)
	p.SendCrit(message)
}

func (p *Project) ProjectSendReport() error {
	var (
		ceasedChecks                []string
		reportMessage, reportHeader string
	)

	for _, hc := range p.Healthchecks {
		for _, c := range hc.Checks {
			if status.GetCheckMode(&c) == "quiet" {
				ceasedChecks = append(ceasedChecks, c.UUid)
			}
		}
	}

	if p.IsQuiet() {
		reportMessage = fmt.Sprintf("Project %s in quiet state\n", p.Name)
	} else {
		if len(ceasedChecks) > 0 {
			reportHeader = fmt.Sprintf("Project %s in %s state\n", p.Name, p.GetMode())
			reportMessage = reportHeader + fmt.Sprintf("Ceased checks: %v\n", ceasedChecks)
		}
	}

	if reportMessage != "" {
		alerts.SendChatOps(reportMessage)
	}
	return nil
}

func (p *Project) AddProjectMetricCriticalAlert() {
	metrics.AlertsCount.WithLabelValues(p.Name, "Critical").Inc()
	metrics.ProjectAlerts.WithLabelValues(p.Name, "Critical").Inc()
}

func (p *Project) AddProjectMetricChatOpsAnswer() {
	metrics.AlertsCount.WithLabelValues(p.Name, "ChatOps_Message").Inc()
	metrics.ProjectAlerts.WithLabelValues(p.Name, "ChatOps_Message").Inc()
}

func GetProjectByName(name string) *Project {
	for _, project := range config.Config.Projects {
		if project.Name == name {
			return &Project{project}
		}
	}
	return nil
}
