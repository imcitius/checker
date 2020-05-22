package project

import (
	"errors"
	"fmt"
	alerts "my/checker/alerts"
	checks "my/checker/checks"
	config "my/checker/config"
	"my/checker/telegram"
)

func GetName(p *config.Project) string {
	return p.Name
}

func GetMode(p *config.Project) string {
	return p.Parameters.Mode
}

func Alert(p *config.Project, alerttype string, e error) {
	config.Log.Debugf("Send non-critical alert for project: '%+v', with error '%+v'\n", p.Name, e)
	//config.Log.Printf("%+v", Config.Alerts)
	if config.Config.Defaults.Parameters.Mode == "loud" && p.Parameters.Mode == "loud" {
		if p.Parameters.Mode == "loud" {
			for _, alert := range config.Config.Alerts {
				//config.Log.Printf("%+v", alert)
				if alerts.GetAlertName(&alert) == p.Parameters.Alert {
					//config.Log.Printf("Alert details: %+v\n\n", alert)
					alerts.SendAlert(&alert, "noncrit", e)
				}
			}
		}
	}
}

func CritAlert(p *config.Project, alerttype string, e error) {
	config.Log.Printf("Send critical alert for project: %+v with error %+v\n\n", p, e)
	for _, alert := range config.Config.Alerts {
		//config.Log.Printf("%+v", alert)
		if alerts.GetAlertName(&alert) == p.Parameters.CritAlert {
			//config.Log.Printf("Alert details: %+v\n\n", alert)
			alerts.SendAlert(&alert, "crit", e)
		}
	}
}

func SendReport(p *config.Project) error {
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
		reportHeader = fmt.Sprintf("Project %s in %s state\n", GetName(p), GetMode(p))
		reportMessage = reportHeader + fmt.Sprintf("Ceased checks: %v\n", ceasedChecks)
	} else {
		if p.Parameters.Mode == "quiet" {
			reportMessage = fmt.Sprintf("Project %s in quiet state\n", GetName(p))
		}
	}

	if reportMessage != "" || p.Parameters.Mode == "quiet" {
		for _, alert := range config.Config.Alerts {
			if alert.Name == p.Parameters.Alert {
				alertName := alert.Type
				switch alertName {
				case "telegram":
					config.Log.Printf("Sending report for project %s\n", GetName(p))
					telegram.SendTgMessage("report", &alert, errors.New(reportMessage))
				default:
					errors.New("Alert method not implemented")
				}
			}
		}
	}
	return nil
}
