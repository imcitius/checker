package main

import (
	"errors"
	"fmt"
	"log"
)

func (p *Project) AddError() error {
	p.ErrorsCount++
	return nil
}

func (p *Project) DecError() error {
	if p.ErrorsCount > 0 {
		p.ErrorsCount--
	}
	return nil

}

func (p *Project) AddFail() error {
	p.FailsCount++
	return nil
}

func (p *Project) DecFail() error {
	if p.FailsCount > 0 {
		p.FailsCount--
	}
	return nil

}

func (p *Project) CeaseAlerts() error {
	log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "quiet"
	log.Printf("New mode: %s", p.Parameters.Mode)
	return nil
}

func (p *Project) EnableAlerts() error {
	log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "loud"
	log.Printf("New mode: %s", p.Parameters.Mode)
	return nil
}

func (p *Project) SendReport() error {
	var (
		project                     CommonProject = p
		ceasedChecks                []string
		reportMessage, reportHeader string
	)
	for _, healthcheck := range p.Healtchecks {
		for _, check := range healthcheck.Checks {
			if check.Mode == "quiet" {
				ceasedChecks = append(ceasedChecks, check.UUID())
			}
		}
	}

	if len(ceasedChecks) > 0 {
		reportHeader = fmt.Sprintf("Project %s in %s state\n", project.GetName(), project.GetMode())
		reportMessage = reportHeader + fmt.Sprintf("Ceased checks: %v\n", ceasedChecks)
	} else {
		if p.Parameters.Mode == "quiet" {
			reportMessage = fmt.Sprintf("Project %s in quiet state\n", project.GetName())
		}
	}

	if reportMessage != "" || p.Parameters.Mode == "quiet" {
		for _, alert := range Config.Alerts {
			if alert.Name == p.Parameters.Alert {
				alertName := alert.Type
				switch alertName {
				case "telegram":
					log.Printf("Sending report for project %s\n", project.GetName())
					sendTgMessage(alert, errors.New(reportMessage))
				default:
					errors.New("Alert method not implemented")
				}
			}
		}
	}
	return nil
}

func (p *Project) GetName() string {
	return p.Name
}

func (p *Project) GetMode() string {
	return p.Parameters.Mode
}

func (p *Project) Alert(e error) {
	//log.Printf("Send non-critical alert for project: '%+v', with error '%+v'\n", p.Name, e)
	//log.Printf("%+v", Config.Alerts)
	if Config.Defaults.Parameters.Mode == "loud" && p.Parameters.Mode == "loud" {
		if p.Parameters.Mode == "loud" {
			for _, alert := range Config.Alerts {
				//log.Printf("%+v", alert)
				if alert.GetName() == p.Parameters.Alert {
					//log.Printf("Alert details: %+v\n\n", alert)
					alert.Send(e)
				}
			}
		}
	}
}

func (p *Project) CritAlert(e error) {
	log.Printf("Send critical alert for project: %+v with error %+v\n\n", p, e)
	for _, alert := range Config.Alerts {
		//log.Printf("%+v", alert)
		if alert.GetName() == p.Parameters.CritAlert {
			//log.Printf("Alert details: %+v\n\n", alert)
			alert.Send(e)
		}
	}
}
