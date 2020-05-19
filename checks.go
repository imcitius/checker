package main

import (
	"fmt"
	"regexp"
)

func (c *Check) Execute(p *Project) error {
	var err error

	if _, ok := Checks[c.Type]; ok {
		err = Checks[c.Type](c, p)
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

func (c *Check) UUID() string {
	return c.uuID
}

func (c *Check) GetScheme() string {
	pattern := regexp.MustCompile("(.*)://")
	result := pattern.FindStringSubmatch(c.Host)
	return result[1]
}

func (c *Check) HostName() string {
	return c.Host
}

func (c *Check) CeaseAlerts() error {
	log.Printf("Old mode: %s", c.Mode)
	c.Mode = "quiet"
	log.Printf("New mode: %s", c.Mode)
	return nil
}

func (c *Check) EnableAlerts() error {
	log.Printf("Old mode: %s", c.Mode)
	c.Mode = "loud"
	log.Printf("New mode: %s", c.Mode)
	return nil
}

func (c *Check) AddError() error {
	c.ErrorsCount++
	return nil
}

func (c *Check) DecError() error {
	if c.ErrorsCount > 0 {
		c.ErrorsCount--
	}
	return nil
}

func (c *Check) GetErrors() int {
	return c.ErrorsCount
}

func (c *Check) AddFail() error {
	c.FailsCount++
	return nil
}

func (c *Check) DecFail() error {
	if c.FailsCount > 0 {
		c.FailsCount--
	}
	return nil
}

func (c *Check) GetFails() int {
	return c.FailsCount
}

func (h *Healtchecks) AddError() error {
	h.ErrorsCount++
	return nil
}

func (h *Healtchecks) DecError() error {
	if h.ErrorsCount > 0 {
		h.ErrorsCount--
	}
	return nil
}

func (h *Healtchecks) GetErrors() int {
	return h.ErrorsCount
}

func (h *Healtchecks) AddFail() error {
	h.FailsCount++
	return nil
}

func (h *Healtchecks) DecFail() error {
	if h.FailsCount > 0 {
		h.FailsCount--
	}
	return nil
}

func (h *Healtchecks) GetFails() int {
	return h.FailsCount
}
