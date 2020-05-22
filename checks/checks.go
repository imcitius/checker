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

func CheckAddError(c *config.Check) error {
	c.ErrorsCount++
	return nil
}

func CheckDecError(c *config.Check) error {
	if c.ErrorsCount > 0 {
		c.ErrorsCount--
	}
	return nil
}

func CheckGetErrors(c *config.Check) int {
	return c.ErrorsCount
}

func CheckAddFail(c *config.Check) error {
	c.FailsCount++
	return nil
}

func CheckDecFail(c *config.Check) error {
	if c.FailsCount > 0 {
		c.FailsCount--
	}
	return nil
}

func CheckGetFails(c *config.Check) int {
	return c.FailsCount
}

func HealtcheckAddError(h *config.Healtchecks) error {
	h.ErrorsCount++
	return nil
}

func HealtcheckDecError(h *config.Healtchecks) error {
	if h.ErrorsCount > 0 {
		h.ErrorsCount--
	}
	return nil
}

func HealtcheckGetErrors(h *config.Healtchecks) int {
	return h.ErrorsCount
}

func HealtcheckAddFail(h *config.Healtchecks) error {
	h.FailsCount++
	return nil
}

func HealtcheckDecFail(h *config.Healtchecks) error {
	if h.FailsCount > 0 {
		h.FailsCount--
	}
	return nil
}

func HealtcheckGetFails(h *config.Healtchecks) int {
	return h.FailsCount
}
