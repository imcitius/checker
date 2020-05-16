package main

import (
	"fmt"
	"regexp"
)

func (c *Check) Execute(p *Project) error {
	var err error

	switch c.Type {
	case "http":
		//log.Printf("http check execute: %+v\n", c.Host)
		err = runHTTPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "icmp":
		//log.Printf("icmp check execute %+v\n", c)
		err = runICMPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "tcp":
		//log.Printf("tcp check execute %+v\n", c)
		err = runTCPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "pgsql_query":
		//log.Printf("postgres_query check execute %+v\n", c)
		err = runPgsqlQueryCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "pgsql_query_unixtime":
		//log.Printf("postgres_query check execute %+v\n", c)
		err = runPgsqlUnixtimeCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "pgsql_replication":
		//log.Printf("postgres_query check execute %+v\n", c)
		err = runPgsqlReplicationCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "mysql_query":
		//log.Printf("mysql_query check execute %+v\n", c)
		err = runMysqlQueryCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "mysql_query_unixtime":
		//log.Printf("mysql_query check execute %+v\n", c)
		err = runMysqlUnixtimeCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "mysql_replication":
		//log.Printf("mysql_query check execute %+v\n", c)
		err = runMysqlReplicationCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "clickhouse_query":
		//log.Printf("clickhouse_query check execute %+v\n", c)
		err = runClickhouseQueryCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "clickhouse_query_unixtime":
		//log.Printf("clickhouse_query check execute %+v\n", c)
		err = runClickhouseUnixtimeCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	case "redis_pubsub":
		//log.Printf("redis_pubsub check execute %+v\n", c)
		err = runRedisPubSubCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
	default:
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
