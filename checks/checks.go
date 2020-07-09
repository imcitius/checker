package check

import (
	"fmt"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/status"
	"regexp"
)

var (
	Checks = make(map[string]func(c *config.Check, p *projects.Project) error)
)

func Execute(p *projects.Project, c *config.Check) error {
	var err error

	if _, ok := Checks[c.Type]; ok {
		err = Checks[c.Type](c, p)
		status.Statuses.Checks[c.UUid].ExecuteCount++
		if err == nil {
			return nil
		}
	} else {
		err = fmt.Errorf("check %s not implemented", c.Type)
	}
	return err
}

func GetCheckScheme(c *config.Check) string {
	pattern := regexp.MustCompile("(.*)://")
	result := pattern.FindStringSubmatch(c.Host)
	return result[1]
}
