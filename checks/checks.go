package check

import (
	"fmt"
	"my/checker/config"
	projects "my/checker/projects"
	"regexp"
)

var (
	Checks = make(map[string]func(c *config.Check, p *projects.Project) error)
)

func Execute(p *projects.Project, c *config.Check) error {
	var err error

	if _, ok := Checks[c.Type]; ok {
		err = Checks[c.Type](c, p)
		if err == nil {
			c.LastResult = true
			return nil
		} else {
			c.LastResult = false
		}
	} else {
		err = fmt.Errorf("check %s not implemented", c.Type)
	}
	return err
}

func UUID(c *config.Check) string {
	return c.UUid
}

func GetCheckByUUID(uuid string) *config.Check {
	for _, project := range config.Config.Projects {
		for _, healthcheck := range project.Healthchecks {
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
