package check

import (
	"fmt"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/status"
	"time"
)

var (
	Checks = make(map[string]func(c *config.Check, p *projects.Project) error)

	// SlackAppAlertFunc is set by the scheduler to send slack_app alerts.
	// Signature: func(project, healthcheck, check, errMessage)
	SlackAppAlertFunc func(p *projects.Project, hc *config.Healthcheck, c *config.Check, errMsg string)

	// SlackAppRecoveryFunc is set by the scheduler to handle slack_app recovery.
	// Signature: func(project, healthcheck, check)
	SlackAppRecoveryFunc func(p *projects.Project, hc *config.Healthcheck, c *config.Check)
)

func Execute(p *projects.Project, c *config.Check) (time.Duration, error) {
	var err error
	startTime := time.Now()

	if _, ok := Checks[c.Type]; ok {
		err = Checks[c.Type](c, p)
		status.Statuses.Checks[c.UUid].ExecuteCount++
		if err == nil {
			endTime := time.Now()
			return endTime.Sub(startTime), nil
		}
	} else {
		err = fmt.Errorf("check %s not implemented", c.Type)
	}
	endTime := time.Now()
	return endTime.Sub(startTime), err
}
