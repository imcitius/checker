package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

func (c *Check) Execute(p *Project) error {

	switch c.Type {
	case "http":
		//log.Printf("http check execute: %+v\n", c.Host)
		err := runHTTPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	case "icmp":
		//log.Printf("icmp check execute %+v\n", c)
		err := runICMPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	case "tcp":
		//log.Printf("tcp check execute %+v\n", c)
		err := runTCPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	default:
		return errors.New("check not implemented")
	}
}

func (c *Check) UUID() string {
	return c.uuID
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
