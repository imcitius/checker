package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "github.com/spf13/viper/remote"
	"path/filepath"
	"reflect"
	"strings"
	"time"
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

func InitConfig() {

	logrus.Info("initConfig: load config file")
	logrus.Infof("Config flag: %s", CfgFile)

	logrus.Infof("%s %s", Viper.GetString("CONSUL_ADDR"), Viper.GetString("CONSUL_PATH"))

	switch {
	case CfgSrc == "" || CfgSrc == "file":
		if CfgFile == "" {
			// Use config file from the flag.
			Viper.SetConfigName("config")         // name of config file (without extension)
			Viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
			Viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
			Viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
			Viper.AddConfigPath(".")              // optionally look for config in the working directory

		} else {
			Viper.SetConfigName(filepath.Base(CfgFile)) // name of config file (without extension)
			if filepath.Ext(CfgFile) == "" {
				Viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
			} else {
				Viper.SetConfigType(filepath.Ext(CfgFile)[1:])
			}
			Viper.AddConfigPath(filepath.Dir(CfgFile)) // path to look for the config file in

		}
		Viper.WatchConfig()
		Viper.OnConfigChange(func(e fsnotify.Event) {
			Log.Info("Config file changed: ", e.Name)
			ConfigChangeSig <- true

		})

	case CfgSrc == "consul":
		if Viper.GetString("CONSUL_ADDR") != "" {
			if Viper.GetString("CONSUL_PATH") != "" {
				Viper.AddRemoteProvider("consul", Viper.GetString("CONSUL_ADDR"), Viper.GetString("CONSUL_PATH"))
				Viper.SetConfigType("json")
			} else {
				panic("Consul path not specified")
			}
		} else {
			panic("Consul URL not specified")
		}
	}

	Viper.AutomaticEnv()

	dl, err := logrus.ParseLevel(Viper.GetString("debugLevel"))
	if err != nil {
		Log.Panicf("Cannot parse debug level: %v", err)
	} else {
		Log.SetLevel(dl)
	}

}

func WatchConfig() {
	for {
		if period, err := time.ParseDuration(CfgWatchTimeout); err != nil {
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
				} else {
					Log.Debugf("Loaded config: %+v", Config)
				}
				ConfigChangeSig <- true
			}
		} else {
			Log.Infof("KV config seems to be broken: %+v", err)
		}

		//configWatchSig <- true
	}
}
