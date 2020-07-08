package config

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/s3"
	"github.com/sirupsen/logrus"
	"my/checker/common"
	"reflect"
	"strings"
	"time"
)

func LoadConfig() error {

	tempConfig, err := TestConfig()
	if err != nil {
		Log.Infof("Using config file: %s", Koanf.All())
	} else {
		Config = tempConfig
	}

	return nil
}

func TestConfig() (ConfigFile, error) {

	var tempConfig ConfigFile

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
	}

	dl, err := logrus.ParseLevel(Koanf.String("debug.level"))
	if err != nil {
		//Log.Panicf("Cannot parse debug level: %v", err)
		return tempConfig, err
	} else {
		Log.SetLevel(dl)
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
	err = tempConfig.FillTimeouts()
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

func (c *ConfigFile) FillDefaults() error {

	//Log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, p := range c.Projects {
		if p.Parameters.RunEvery == "" {
			p.Parameters.RunEvery = c.Defaults.Parameters.RunEvery
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
		if p.Parameters.PeriodicReport == "" {
			p.Parameters.PeriodicReport = c.Defaults.Parameters.PeriodicReport
		}
		if p.Parameters.SSLExpirationPeriod == "" {
			p.Parameters.SSLExpirationPeriod = c.Defaults.Parameters.SSLExpirationPeriod
		}
		if len(p.Parameters.Mentions) == 0 {
			p.Parameters.Mentions = c.Defaults.Parameters.Mentions
		}
		c.Projects[i] = p
	}

	if len(c.Alerts) == 0 {
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

func (c *ConfigFile) FillUUIDs() error {
	var err error
	for i := range c.Projects {
		for j, h := range c.Projects[i].Healthchecks {
			for k, check := range c.Projects[i].Healthchecks[j].Checks {
				c.Projects[i].Healthchecks[j].Checks[k].UUid = common.GenUUID(h.Name + check.Name + check.Host)
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

func (c *ConfigFile) FillTimeouts() error {

	defRunEvery := Koanf.String("defaults.parameters.run_every")
	Timeouts.Add(defRunEvery)

	for _, p := range c.Projects {

		if p.Parameters.RunEvery != defRunEvery {
			Timeouts.Add(p.Parameters.RunEvery)
		}
		for _, h := range p.Healthchecks {
			if h.Parameters.RunEvery != defRunEvery {
				Timeouts.Add(h.Parameters.RunEvery)
				p.Timeouts.Add(h.Parameters.RunEvery)
			}
			//Log.Debugf("Project %s timeouts found: %+v\n", p.Name, p.Timeouts)
		}
	}
	Log.Debugf("Total timeouts found: %+v\n\n", Timeouts)

	return nil
}

func (c *ConfigFile) FillSecrets() error {

	for i, alert := range c.Alerts {
		if strings.HasPrefix(alert.BotToken, "vault") {
			token, err := GetVaultSecret(alert.BotToken)
			if err == nil {
				c.Alerts[i].BotToken = token
			} else {
				return fmt.Errorf("Error getting bot token from vault: %v", err)
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
						return fmt.Errorf("Error getting bot token from vault: %v", err)
					}
				}
				if strings.HasPrefix(check.SqlReplicationConfig.Password, "vault") {
					token, err := GetVaultSecret(check.SqlReplicationConfig.Password)
					if err == nil {
						c.Projects[i].Healthchecks[j].Checks[k].SqlReplicationConfig.Password = token
					} else {
						return fmt.Errorf("Error getting bot token from vault: %v", err)
					}
				}
			}
		}
	}

	return nil
}

func WatchConfig() {
	if period, err := time.ParseDuration(Koanf.String("config.watchtimeout")); err != nil {
		Log.Infof("KV watch timeout parser error: %+v, use 5s", err)
		time.Sleep(time.Second * 5) // default delay
	} else {
		time.Sleep(period)
	}
	tempConfig, err := TestConfig()
	if err == nil {
		if !reflect.DeepEqual(Config, tempConfig) {
			Log.Infof("KV config changed, reloading")
			err := LoadConfig()
			if err != nil {
				Log.Infof("Config load error: %s", err)
			}
			//else {
			//	Log.Debugf("Loaded config: %+v", Config)
			//}
			ConfigChangeSig <- true
		}
	} else {
		Log.Infof("KV config seems to be broken: %+v", err)
	}

	//configWatchSig <- true
}

func ListElements() string {

	list := ""
	for _, p := range Config.Projects {
		list = list + fmt.Sprintf("Project: %s\n", p.Name)
		for _, h := range p.Healthchecks {
			list = list + fmt.Sprintf("\tHealthcheck: %s\n", h.Name)
			for _, c := range h.Checks {
				list = list + fmt.Sprintf("\t\tUUID: %s\n", c.UUid)
			}
		}
	}

	return list
}

func List() {

	err := LoadConfig()
	if err != nil {
		Log.Infof("Config load error: %s", err)
	}

	fmt.Print(ListElements())
}
