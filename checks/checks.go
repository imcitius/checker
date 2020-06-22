package check

import (
	"fmt"
	"my/checker/config"
	"my/checker/metrics"
	"regexp"
)

func Execute(p *config.Project, h *config.Healtchecks, c *config.Check) error {
	var err error

	err = metrics.AddCheckRunCount(p, h, c)
	if err != nil {
		config.Log.Errorf("Metric count error: %v", err)
	}

	if _, ok := config.Checks[c.Type]; ok {
		err = config.Checks[c.Type](c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	} else {
		err = fmt.Errorf("Check %s not implemented", c.Type)
	}
	return err
}

func UUID(c *config.Check) string {
	return c.UUid
}

func GetCheckByUUID(uuid string) *config.Check {
	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healtchecks {
			for _, check := range healthcheck.Checks {
				if uuid == check.UUid {
					return &check
				}
			}
		}
	}
	return nil
}
func GetCheckScheme(c *config.Check) string {
	pattern := regexp.MustCompile("(.*)://")
	result := pattern.FindStringSubmatch(c.Host)
	return result[1]
}
