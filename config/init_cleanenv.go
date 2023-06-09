package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

func initCleanenv(cfgFile string) {
	if cfgFile != "" {
		err := cleanenv.ReadConfig(cfgFile, &config)
		if err != nil {
			panic(fmt.Errorf("Error: uncaught error! %s", err))
		} else {
			logger.Infof("Using c file %s\n", cfgFile)
		}
	} else {
		panic(fmt.Errorf("config file not found at path %s\n", cfgFile))
	}
}
