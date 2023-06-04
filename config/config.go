package config

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	Log = *logrus.New()
)

type Config struct {
	Test string `mapstructure:"test" default:"John Smith"`
}

func (config *Config) InitConfig(cfgFile string) func() {
	return func() {
		if err := defaults.Set(config); err != nil {
			panic(err)
		}

		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		}

		viper.AutomaticEnv()

		err := viper.ReadInConfig()
		if err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Printf("Config file not found at path %s\n", cfgFile)
			} else {
				panic(fmt.Errorf("Error: uncaught error! %s", err))
			}
		} else {
			Log.Infof("Using config file %s\n", viper.ConfigFileUsed())
		}
	}
}
