package checks

import (
	"github.com/teris-io/shortid"
	"math/rand"
	"my/checker/alerts"
	"my/checker/checks/getfile"
	"my/checker/checks/http"
	"my/checker/checks/icmp"
	"my/checker/checks/tcp"
	"my/checker/config"
	"time"
)

func (c TCommonCheck) GetSID() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	s, _ := sid.Generate()
	return s
}

func (c TCommonCheck) GetProject() string {
	return c.Project
}

func (c TCommonCheck) GetHealthcheck() string {
	return c.Healthcheck
}

func (c TCommonCheck) GetHost() string {
	if c.CheckConfig.Url != "" {
		return c.CheckConfig.Url
	} else {
		return c.CheckConfig.Host
	}
}

func (c TCommonCheck) GetType() string {
	return c.Type
}

func (c TCommonCheck) GetName() string {
	return c.Name
}

func (c TCommonCheck) GetResult() TCheckResult {
	return c.Result
}

func (c TCommonCheck) Execute() TCommonCheck {
	//logger.Infof("TCommonCheck started: %s", c.Name)

	d, e := c.RealCheck.RealExecute()

	c.Result = TCheckResult{
		Duration: d,
		Error:    e,
	}

	return c
}

func (c TCommonCheck) Alert(message string) TCommonCheck {
	if c.Alerter != nil {
		c.Alerter.Send(message)
	} else {
		logger.Errorf("Logger not set for check")
		logger.Errorf(message)
	}
	return c
}

func GetChecksByDuration(duration string) (TChecksCollection, error) {
	return findChecksByDuration(duration)
}

func findChecksByDuration(duration string) (TChecksCollection, error) {
	logger.Debugf("Look for checks with durations %s", duration)

	checksCollection := TChecksCollection{
		Checks: []TCheckWithDuration{},
	}

	checks := make([]TCheckWithDuration, 0)

	for _, p := range configurer.Projects {
		logger.Debugf(">>> Project %s", p.Name)

		for _, h := range p.Healthchecks {
			//logger.Debugf(">>> Healthcheck: %+v", h.Name)
			for _, check := range h.Checks {
				//logger.Debugf(">>> Check: %s, dur: %s", check.Name, check.Parameters.Duration)

				dConfig, err := time.ParseDuration(duration)
				if err != nil {
					logger.Errorf("Error parsing wanted duration: %s", err)
				}
				dCheck, _ := time.ParseDuration(check.Parameters.Duration)
				if err != nil {
					logger.Errorf("Error parsing check duration: %s", err)
				}

				if dConfig == dCheck {
					logger.Debugf(">>> Adding check: %s, dur: %s", check.Name, check.Parameters.Duration)
					checkToAdd := newCommonCheck(check, h, p)
					checks = append(checks, TCheckWithDuration{
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

func newCommonCheck(c config.TCheckConfig, h config.THealthcheck, p config.TProject) ICommonCheck {
	newCheck := TCommonCheck{
		CheckConfig: c,
		Name:        c.Name,
		Project:     p.Name,
		Healthcheck: h.Name,
		Type:        c.Type,
		Parameters:  c.Parameters,
		Result: TCheckResult{
			Duration: 0,
			Error:    nil,
		},
	}

	newCheck.Alerter = getAlerter(newCheck)
	newCheck.RealCheck = newSpecificCheck(newCheck)
	return newCheck
}

func getAlerter(c TCommonCheck) alerts.ICommonAlerter {
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

func newSpecificCheck(c TCommonCheck) ISpecificCheck {
	switch c.Type {
	case "http":
		return http.New(c.CheckConfig)
	case "icmp":
		return icmp.New(c.CheckConfig)
	case "tcp":
		return tcp.New(c.CheckConfig)
	case "getfile":
		return getfile.New(c.CheckConfig)
	}

	logger.Errorf("Error constructing specific check: unknown check type: '%s'", c.Type)
	return nil
}

func (c *TChecksCollection) Len() int {
	return len(c.Checks)
}
