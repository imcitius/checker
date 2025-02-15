package checks

import (
	"context"
	"math/rand"
	"my/checker/alerts"
	"my/checker/checks/getfile"
	"my/checker/checks/http"
	"my/checker/checks/icmp"
	"my/checker/checks/passive"
	"my/checker/checks/tcp"
	"my/checker/config"
	"my/checker/models"
	"time"

	"github.com/teris-io/shortid"
)

func (c models.TCommonCheck) GetSID() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	s, _ := sid.Generate()
	return s
}

func (c models.TCommonCheck) GetProject() string {
	return c.Project
}

func (c models.TCommonCheck) GetHealthcheck() string {
	return c.Healthcheck
}

func (c models.TCommonCheck) GetHost() string {
	if c.CheckConfig.Url != "" {
		return c.CheckConfig.Url
	} else {
		return c.CheckConfig.Host
	}
}

func (c models.TCommonCheck) GetUUID() string {
	return c.UUID
}

func (c models.TCommonCheck) GetCheckDetails() models.TCheckDetails {
	return c.GetCheckDetails()
}

func (c models.TCommonCheck) GetType() string {
	return c.Type
}

func (c models.TCommonCheck) GetName() string {
	return c.Name
}

func (c models.TCommonCheck) GetResult() models.TCheckResult {
	return c.Result
}

func (c models.TCommonCheck) SetStatus(status bool) {
	configurer.SetStatus(c.CheckConfig.UUID, status)
}

func (c models.TCommonCheck) Execute() models.TCommonCheck {
	//logger.Infof("models.TCommonCheck started: %s", c.Name)

	d, e := c.RealCheck.RealExecute()

	c.Result = models.TCheckResult{
		Duration: d,
		Error:    e,
	}

	return c
}

func (c models.TCommonCheck) Alert(ctx context.Context, message string) models.TCommonCheck {
	if c.Alerter != nil {
		c.Alerter.Alert(ctx, models.TAlertDetails{
			Severity:    "noncritical",
			Message:     message,
			UUID:        c.UUID,
			ProjectName: c.Project,
		})
	} else {
		logger.Errorf("Alerter not set %s (for check: %s)", message, c.UUID)
	}
	return c
}

func GetChecksByDuration(duration string) (models.TChecksCollection, error) {
	return findChecksByDuration(duration)
}

func findChecksByDuration(duration string) (models.TChecksCollection, error) {
	logger.Debugf("Look for checks with durations %s", duration)

	checksCollection := models.TChecksCollection{
		Checks: []models.TCheckWithDuration{},
	}

	checks := make([]models.TCheckWithDuration, 0)

	for _, p := range configurer.Projects {
		logger.Debugf(">>> Project %s", p.Name)

		for _, h := range p.Healthchecks {
			logger.Debugf(">>> Healthcheck: %+v", h.Name)
			for _, check := range h.Checks {
				logger.Debugf(">>> Check: %s, dur: %s", check.Name, check.Parameters.Duration)

				dConfig, err := time.ParseDuration(duration)
				if err != nil {
					logger.Errorf("Error parsing wanted duration: %s", err)
				}
				dCheck, _ := time.ParseDuration(check.Parameters.Duration)
				if err != nil {
					logger.Errorf("Error parsing check duration: %s", err)
				}

				if dConfig == dCheck {
					logger.Debugf(">>> Adding check: %s, dur: %s, enabled: %t", check.Name, check.Parameters.Duration, check.Enabled)
					checkToAdd := newCommonCheck(check, h, p)
					checks = append(checks, models.TCheckWithDuration{
						Check:    checkToAdd,
						Duration: duration,
					})
				}
			}
		}
	}

	checksCollection.Checks = checks

	//logger.Infof("Checks: %+v", checksCollection)
	//spew.Dump(checksCollection)
	//logger.Fatalf("%s, %d", duration, len(checksCollection.Checks[duration]))

	return checksCollection, nil
}

func newCommonCheck(c models.TCheckConfig, h models.THealthcheck, p models.TProject) ICommonCheck {
	newCheck := models.TCommonCheck{
		CheckConfig: c,
		Name:        c.Name,
		Project:     p.Name,
		Healthcheck: h.Name,
		Type:        c.Type,
		UUID:        c.UUID,
		Parameters:  c.Parameters,
		Result: models.TCheckResult{
			Duration: 0,
			Error:    nil,
		},
		Enabled: c.Enabled,
	}

	newCheck.Alerter = getAlerter(newCheck)
	newCheck.RealCheck = newSpecificCheck(newCheck)
	return newCheck
}

func getAlerter(c models.TCommonCheck) models.ICommonAlerter {
	alerter := alerts.GetAlerterByName(configurer.Defaults.AlertsChannel)

	if c.Parameters.AlertChannel != "" {
		return alerts.GetAlerterByName(c.Parameters.AlertChannel)
	} else {
		if proj, _ := config.GetProjectByName(c.Project); proj.Parameters.AlertChannel != "" {
			logger.Infof("bzdury: %s", proj.Parameters.AlertChannel)
			return alerts.GetAlerterByName(proj.Parameters.AlertChannel)
		}
	}

	return alerter
}

func newSpecificCheck(c models.TCommonCheck) models.ISpecificCheck {
	switch c.Type {
	case "http":
		return http.New(c.CheckConfig)
	case "icmp":
		return icmp.New(c.CheckConfig)
	case "tcp":
		return tcp.New(c.CheckConfig)
	case "getfile":
		return getfile.New(c.CheckConfig)
	case "passive":
		return passive.New(c.CheckConfig)
	}

	logger.Errorf("Error constructing specific check: unknown check type: '%s'", c.Type)
	return nil
}

func (c *models.TChecksCollection) Len() int {
	return len(c.Checks)
}

func (c models.TCommonCheck) GetEnabled() bool {
	return c.Enabled
}
