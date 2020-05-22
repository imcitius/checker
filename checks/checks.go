package check

import (
	"fmt"
	"my/checker/config"
	"regexp"
)

func Execute(c *config.Check, p *config.Project) error {
	var err error

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

func GetCheckScheme(c *config.Check) string {
	pattern := regexp.MustCompile("(.*)://")
	result := pattern.FindStringSubmatch(c.Host)
	return result[1]
}

func HostName(c *config.Check) string {
	return c.Host
}

func CeaseAlerts(c *config.Check) error {
	config.Log.Printf("Old mode: %s", c.Mode)
	c.Mode = "quiet"
	config.Log.Printf("New mode: %s", c.Mode)
	return nil
}

func EnableAlerts(c *config.Check) error {
	config.Log.Printf("Old mode: %s", c.Mode)
	c.Mode = "loud"
	config.Log.Printf("New mode: %s", c.Mode)
	return nil
}
