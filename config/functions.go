package config

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/s3"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"time"
)

func LoadConfig() error {

	tempConfig, err := TestConfig()
	if err != nil {
		//Log.Infof("Using config file: %s", Koanf.All())
		Log.Panicf("Сonfig parsing error: %s", err)
	} else {
		Config = tempConfig
	}

	return nil
}

func TestConfig() (File, error) {

	var tempConfig File

	switch {
	case Koanf.String("config.source") == "" || Koanf.String("config.source") == "file":

		switch {
		case Koanf.String("config.format") == "json":
			f := file.Provider(Koanf.String("config.file"))
			if err := Koanf.Load(f, json.Parser()); err != nil {
				return tempConfig, err
			}

		case Koanf.String("config.format") == "yaml":
			f := file.Provider(Koanf.String("config.file"))
			if err := Koanf.Load(f, yaml.Parser()); err != nil {
				return tempConfig, err
			}

		case Koanf.String("config.format") == "toml":
			f := file.Provider(Koanf.String("config.file"))
			if err := Koanf.Load(f, toml.Parser()); err != nil {
				return tempConfig, err
			}

		case Koanf.String("config.format") == "hcl":
			f := file.Provider(Koanf.String("config.file"))
			if err := Koanf.Load(f, hcl.Parser(true)); err != nil {
				return tempConfig, err
			}
		}

	case Koanf.String("config.source") == "s3" || Koanf.String("config.source") == "S3":

		s3config := s3.Config{
			AccessKey: Koanf.String("aws.access.key.id"),
			SecretKey: Koanf.String("aws.secret.access.key"),
			Region:    Koanf.String("aws.region"),
			Bucket:    Koanf.String("aws.bucket"),
			ObjectKey: Koanf.String("aws.object.key"),
		}

		switch {

		case Koanf.String("config.format") == "yaml":

			// Load yaml config from s3.
			if err := Koanf.Load(s3.Provider(s3config), yaml.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}

		case Koanf.String("config.format") == "json":

			// Load json config from s3.
			if err := Koanf.Load(s3.Provider(s3config), json.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}
		}

	case Koanf.String("config.source") == "consul":

		consulconfig := ConsulConfig{
			&ConsulParam{
				KVPath: Koanf.String("consul.path"),
			},
			&api.Config{
				Address: Koanf.String("consul.addr"),
			}}

		switch {

		case Koanf.String("config.format") == "json":

			// Load json config from consul.
			if err := Koanf.Load(ConsulProvider(&consulconfig), json.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}

		case Koanf.String("config.format") == "yaml":

			// Load yaml config from consul.
			if err := Koanf.Load(ConsulProvider(&consulconfig), yaml.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}
		}

	case Koanf.String("config.source") == "env":

		// watch should be disabled
		//Koanf.

		switch {

		case Koanf.String("config.format") == "yaml":
			// Load yaml config from env
			if err := Koanf.Load(EnvProvider(Koanf.String("checker.config")), yaml.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}

		case Koanf.String("config.format") == "json":
			// Load json config from env variable
			if err := Koanf.Load(EnvProvider(Koanf.String("checker.config")), json.Parser()); err != nil {
				logrus.Fatalf("error loading config: %v", err)
			}
		}

	}

	dl, err := logrus.ParseLevel(Koanf.String("debug.level"))
	if err != nil {
		Log.Errorf("Cannot parse debug level: %v", err)
		return tempConfig, err
	} else {
		Log.SetLevel(dl)
		switch Koanf.String("log.format") {
		case "json":
			Log.SetFormatter(&logrus.JSONFormatter{})
		case "text":
			Log.SetFormatter(&logrus.TextFormatter{})
		}
		if Koanf.String("debug.level") == "debug" {
			// add file and line number
			Log.SetReportCaller(true)
		}
	}

	if err := Koanf.Unmarshal("", &tempConfig); err != nil {
		return tempConfig, err
	}

	err = tempConfig.FillSecrets()
	if err != nil {
		return tempConfig, err
	}
	err = tempConfig.FillDefaults()
	if err != nil {
		return tempConfig, err
	}
	err = tempConfig.FillUUIDs()
	if err != nil {
		return tempConfig, err
	}
	err = tempConfig.FillPeriods()
	if err != nil {
		return tempConfig, err
	}

	return tempConfig, nil
}

func (p *TimeoutsCollection) Add(period string) {
	var found bool
	if period != "" {
		for _, item := range p.Periods {
			if item == period {
				found = true
			}
		}
		if !found {
			p.Periods = append(p.Periods, period)
		}
	}
}

func (c *File) FillDefaults() error {

	if c.Defaults.Parameters.ReportPeriod == "" {
		c.Defaults.Parameters.ReportPeriod = DefaultPeriodicReportPeriod
		Log.Debugf("ReportPeriod not found in config, use defaults: %s", c.Defaults.Parameters.ReportPeriod)
	}
	if c.Defaults.Parameters.Period == "" {
		c.Defaults.Parameters.Period = DefaultCheckPeriod
		Log.Debugf("DefaultCheckPeriod not found in config, use default: %s", DefaultCheckPeriod)
	}
	if c.Defaults.Parameters.SSLExpirationPeriod == "" {
		c.Defaults.Parameters.SSLExpirationPeriod = DefaultSSLExpiration
		Log.Debugf("SSLExpirationPeriod not found in config, use default: %s", DefaultSSLExpiration)
	}

	//Log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, p := range c.Projects {
		if p.Parameters.Period == "" {
			p.Parameters.Period = c.Defaults.Parameters.Period
		}
		if p.Parameters.Mode == "" {
			p.Parameters.Mode = c.Defaults.Parameters.Mode
		}
		if p.Parameters.AllowFails == 0 {
			p.Parameters.AllowFails = c.Defaults.Parameters.AllowFails
		}
		if p.Parameters.MinHealth == 0 {
			p.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if p.Parameters.MinHealth == 0 {
			p.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if p.Parameters.AlertChannel == "" {
			p.Parameters.AlertChannel = c.Defaults.Parameters.AlertChannel
		}
		if p.Parameters.CritAlertChannel == "" {
			p.Parameters.CritAlertChannel = c.Defaults.Parameters.AlertChannel
		}
		if p.Parameters.ReportPeriod == "" {
			p.Parameters.ReportPeriod = c.Defaults.Parameters.ReportPeriod
		}
		if p.Parameters.SSLExpirationPeriod == "" {
			p.Parameters.SSLExpirationPeriod = c.Defaults.Parameters.SSLExpirationPeriod
		}
		if len(p.Parameters.Mentions) == 0 {
			p.Parameters.Mentions = c.Defaults.Parameters.Mentions
		}
		c.Projects[i] = p
	}

	if c.Alerts == nil {
		var alert AlertConfigs
		alert.Name = "log"
		alert.Type = "log"
		c.Alerts = append(c.Alerts, alert)
		c.Defaults.Parameters.CommandChannel = "log"
		c.Defaults.Parameters.AlertChannel = "log"
		c.Defaults.Parameters.CritAlertChannel = "log"
	}
	return nil
}

func (c *File) FillUUIDs() error {
	var err error
	for i := range c.Projects {
		for j, h := range c.Projects[i].Healthchecks {
			for k, check := range c.Projects[i].Healthchecks[j].Checks {
				c.Projects[i].Healthchecks[j].Checks[k].UUid = GenUUID(h.Name, check.Name, check.Host)
				if c.Projects[i].Healthchecks[j].Checks[k].UUid == "" {
					return err
				}
			}
		}
	}
	return nil
}

func (p *TimeoutCollection) Add(period string) {
	var found bool
	if period != "" {
		for _, item := range p.periods {
			if item == period {
				found = true
			}
		}
		if !found {
			p.periods = append(p.periods, period)
		}
	}
}

func (c *File) FillPeriods() error {

	Timeouts.Add(Koanf.String("defaults.parameters.check_period"))
	Timeouts.Add(Koanf.String("defaults.parameters.report_period"))

	for _, p := range c.Projects {
		Timeouts.Add(p.Parameters.Period)
		for _, h := range p.Healthchecks {
			Timeouts.Add(h.Parameters.Period)
			p.Timeouts.Add(h.Parameters.Period)
		}
	}
	Log.Debugf("Total timeouts found: %+v\n\n", Timeouts)

	return nil
}

func (c *File) FillSecrets() error {

	for i, alert := range c.Alerts {
		if strings.HasPrefix(alert.BotToken, "vault") {
			token, err := GetVaultSecret(alert.BotToken)
			if err == nil {
				c.Alerts[i].BotToken = token
			} else {
				return fmt.Errorf("error getting bot token from vault: %v", err)
			}
		}
	}

	for i, project := range c.Projects {
		for j, hc := range project.Healthchecks {
			for k, check := range hc.Checks {
				if strings.HasPrefix(check.SqlQueryConfig.Password, "vault") {
					token, err := GetVaultSecret(check.SqlQueryConfig.Password)
					if err == nil {
						c.Projects[i].Healthchecks[j].Checks[k].SqlQueryConfig.Password = token
					} else {
						return fmt.Errorf("error getting SQL password from vault: %v", err)
					}
				}
				if strings.HasPrefix(check.SqlReplicationConfig.Password, "vault") {
					token, err := GetVaultSecret(check.SqlReplicationConfig.Password)
					if err == nil {
						c.Projects[i].Healthchecks[j].Checks[k].SqlReplicationConfig.Password = token
					} else {
						return fmt.Errorf("error getting SQL password from vault: %v", err)
					}
				}
				if strings.HasPrefix(check.Auth.Password, "vault") {
					token, err := GetVaultSecret(check.Auth.Password)
					if err == nil {
						c.Projects[i].Healthchecks[j].Checks[k].Auth.Password = token
					} else {
						return fmt.Errorf("error getting http password from vault: %v", err)
					}
				}
			}
		}
	}

	if strings.HasPrefix(string(c.Defaults.TokenEncryptionKey), "vault") {
		token, err := GetVaultSecret(string(c.Defaults.TokenEncryptionKey))
		if err == nil {
			TokenEncryptionKey = []byte(token)
		} else {
			return fmt.Errorf("error getting jwt encryption token from vault: %v", err)
		}
	} else {
		TokenEncryptionKey = c.Defaults.TokenEncryptionKey
	}

	return nil
}

func WatchConfig() {
	p, _ := time.ParseDuration(Koanf.String("config.watchtimeout"))
	ticker := time.NewTicker(p)
	for {
		select {
		case <-ticker.C:
			tempConfig, err := TestConfig()
			if err == nil {
				if !reflect.DeepEqual(Config, tempConfig) {
					Log.Infof("KV config changed, reloading")
					err := LoadConfig()
					if err != nil {
						Log.Infof("Config load error: %s", err)
					}
					ConfigChangeSig <- true
				}
			} else {
				Log.Infof("KV config seems to be broken: %+v", err)
			}

		}
	}
}

func (c *Check) IsCritical() bool {
	if c.Severity == "critical" || c.Severity == "crit" {
		return true
	}
	return false
}

func (c *Check) GetCheckScheme() string {
	pattern := regexp.MustCompile("(.*)://")
	result := pattern.FindStringSubmatch(c.Host)
	return result[1]
}

func GetCheckByUUID(uuID string) *Check {
	for _, project := range Config.Projects {
		for _, healthcheck := range project.Healthchecks {
			for _, check := range healthcheck.Checks {
				if uuID == check.UUid {
					return &check
				}
			}
		}
	}
	return nil
}

func GetRandomId() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	checkRuntimeId, _ := sid.Generate()
	return checkRuntimeId
}

func GenUUID(name ...string) string {
	var err error

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		return ""
	}

	u2 := uuid.NewSHA1(ns, []byte(strings.Join(name, ".")))
	return u2.String()
}
