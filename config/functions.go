package config

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "github.com/spf13/viper/remote"
	"strings"
)

func LoadConfig() error {

	tempConfig, err := TestConfig()
	if err != nil {
		Log.Infof("Using config file: %s", Viper.ConfigFileUsed())
	} else {
		Config = tempConfig
	}

	return nil
}

func TestConfig() (ConfigFile, error) {

	var tempConfig ConfigFile

	switch {
	case CfgSrc == "" || CfgSrc == "file":
		err := Viper.ReadInConfig() // Find and read the config file
		if err != nil {             // Handle errors reading the config file
			//Log.Infof("Fatal error config file: %s \n", err)
			return tempConfig, err
		}

	case CfgSrc == "consul":
		err := Viper.ReadRemoteConfig() // Find and read the config file
		if err != nil {                 // Handle errors reading the config file
			//Log.Infof("Fatal error config file: %s \n", err)
			return tempConfig, err
		}
	}

	dl, err := logrus.ParseLevel(Viper.GetString("debugLevel"))
	if err != nil {
		//Log.Panicf("Cannot parse debug level: %v", err)
		return tempConfig, err
	} else {
		Log.SetLevel(dl)
	}

	err = Viper.Unmarshal(&tempConfig)
	if err != nil {
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
	} else {
		Log.Debug("Empty timeout not adding")
	}
}

func (c *ConfigFile) FillDefaults() error {

	//config.Log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range c.Projects {
		if project.Parameters.RunEvery == "" {
			project.Parameters.RunEvery = c.Defaults.Parameters.RunEvery
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = c.Defaults.Parameters.Mode
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = c.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = c.Defaults.Parameters.MinHealth
		}
		if project.Parameters.AlertChannel == "" {
			project.Parameters.AlertChannel = c.Defaults.Parameters.AlertChannel
		}
		if project.Parameters.CritAlertChannel == "" {
			project.Parameters.CritAlertChannel = c.Defaults.Parameters.AlertChannel
		}
		if project.Parameters.PeriodicReport == "" {
			project.Parameters.PeriodicReport = c.Defaults.Parameters.PeriodicReport
		}
		if project.Parameters.SSLExpirationPeriod == "" {
			project.Parameters.SSLExpirationPeriod = c.Defaults.Parameters.SSLExpirationPeriod
		}
		c.Projects[i] = project
	}

	return nil
}

func (c *ConfigFile) FillUUIDs() error {
	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	for i := range c.Projects {
		for j := range c.Projects[i].Healtchecks {
			for k := range c.Projects[i].Healtchecks[j].Checks {
				u2 := uuid.NewSHA1(ns, []byte(c.Projects[i].Healtchecks[j].Checks[k].Host))
				c.Projects[i].Healtchecks[j].Checks[k].UUid = u2.String()
			}
		}
	}
	return err
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
	} else {
		Log.Debug("Empty timeout not adding")
	}
}

func (c *ConfigFile) FillTimeouts() error {

	defRunEvery := Viper.GetString("defaults.parameters.run_every")
	Timeouts.Add(defRunEvery)

	for _, p := range c.Projects {

		if p.Parameters.RunEvery != defRunEvery {
			Timeouts.Add(p.Parameters.RunEvery)
		}
		for _, h := range p.Healtchecks {
			if h.Parameters.RunEvery != defRunEvery {
				Timeouts.Add(h.Parameters.RunEvery)
				p.Timeouts.Add(h.Parameters.RunEvery)
			}
			Log.Debugf("Project %s timeouts found: %+v\n", p.Name, p.Timeouts)
		}
	}
	Log.Debugf("Total timeouts found: %+v\n\n", Timeouts)

	return nil
}

func (c *ConfigFile) FillSecrets() error {

	for i, alert := range c.Alerts {
		if strings.HasPrefix(alert.BotToken, "vault") {
			vault := strings.Split(alert.BotToken, ":")
			path := vault[1]
			field := vault[2]
			token, err := GetVaultSecret(path, field)
			if err == nil {
				c.Alerts[i].BotToken = token
			} else {
				return fmt.Errorf("Error getting bot token from vault: %v", err)
			}
		}
	}
	return nil
}
