package config

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

func LoadConfig() error {

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		Log.Panicf("Fatal error config file: %s \n", err)
	}

	dl, err := logrus.ParseLevel(viper.GetString("debugLevel"))
	if err != nil {
		Log.Panicf("Cannot parse debug level: %v", err)
	} else {
		Log.SetLevel(dl)
	}

	viper.Unmarshal(&Config)

	return nil
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

func FillDefaults() error {

	//config.Log.Printf("Loaded config %+v\n\n", Config.Projects)
	for i, project := range Config.Projects {
		if project.Parameters.RunEvery == "" {
			project.Parameters.RunEvery = Config.Defaults.Parameters.RunEvery
		}
		if project.Parameters.Mode == "" {
			project.Parameters.Mode = Config.Defaults.Parameters.Mode
		}
		if project.Parameters.AllowFails == 0 {
			project.Parameters.AllowFails = Config.Defaults.Parameters.AllowFails
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.MinHealth == 0 {
			project.Parameters.MinHealth = Config.Defaults.Parameters.MinHealth
		}
		if project.Parameters.Alert == "" {
			project.Parameters.Alert = Config.Defaults.Parameters.Alert
		}
		if project.Parameters.CritAlert == "" {
			project.Parameters.CritAlert = Config.Defaults.Parameters.Alert
		}
		if project.Parameters.PeriodicReport == "" {
			project.Parameters.PeriodicReport = Config.Defaults.Parameters.PeriodicReport
		}
		if project.Parameters.SSLExpirationPeriod == "" {
			project.Parameters.SSLExpirationPeriod = Config.Defaults.Parameters.SSLExpirationPeriod
		}
		Config.Projects[i] = project
	}

	return nil
}

func FillUUIDs() error {
	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	for i := range Config.Projects {
		for j := range Config.Projects[i].Healtchecks {
			for k := range Config.Projects[i].Healtchecks[j].Checks {
				u2 := uuid.NewSHA1(ns, []byte(Config.Projects[i].Healtchecks[j].Checks[k].Host))
				Config.Projects[i].Healtchecks[j].Checks[k].UUid = u2.String()
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

func FillTimeouts() error {
	Timeouts.Add(Config.Defaults.Parameters.RunEvery)

	for _, project := range Config.Projects {

		//config.Log.Debugf("Project name: %s", project.Name)
		//config.Log.Debugf("Parameters: %+v", project.Parameters)

		if project.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
			Timeouts.Add(project.Parameters.RunEvery)
		}
		for _, healthcheck := range project.Healtchecks {
			if healthcheck.Parameters.RunEvery != Config.Defaults.Parameters.RunEvery {
				Timeouts.Add(healthcheck.Parameters.RunEvery)
				project.Timeouts.Add(healthcheck.Parameters.RunEvery)
			}
			Log.Debugf("Project %s timeouts found: %+v\n", project.Name, project.Timeouts)
		}
	}
	Log.Debugf("Total timeouts found: %+v\n\n", Timeouts)

	return nil
}

func FillSecrets() error {

	for i, alert := range Config.Alerts {
		if strings.HasPrefix(alert.BotToken, "vault") {
			vault := strings.Split(alert.BotToken, ":")
			path := vault[1]
			field := vault[2]
			token, err := GetVaultSecret(path, field)
			if err == nil {
				Config.Alerts[i].BotToken = token
			} else {
				return fmt.Errorf("Error getting bot token from vault: %v", err)
			}
		}
	}
	return nil
}
