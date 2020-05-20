package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

var log = logrus.New()

type config struct {
}

func main() {

	debugLevel := flag.String("debug", "Debug", "Debug,Info,Warn,Error,Fatal,Panic")
	flag.Parse()

	//viper.SetDefault("ConfigFile", "config")
	//viper.SetDefault("Taxonomies", map[string]string{"tag": "tags", "category": "categories"})
	viper.SetDefault("HTTPPort", "80")

	viper.SetConfigName("config")         // name of config file (without extension)
	viper.SetConfigType("json")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	err := viper.ReadInConfig()           // Find and read the config file
	if err != nil {                       // Handle errors reading the config file
		log.Panicf("Fatal error config file: %s \n", err)
	}

	dl, err := logrus.ParseLevel(*debugLevel)
	if err != nil {
		log.Panicf("Cannot parse debug level: %v", err)
	} else {
		log.SetLevel(dl)
	}

	//log.Infof("Config: \n%+v\n\n\n", viper.AllSettings())
	//log.Debugf("%+v\n\n", viper.Get("alerts"))

	//projects := viper.Sub("projects")
	//log.Infof("%+v\n\n", projects)

	viper.Unmarshal(&Config)
	log.Infof("Config: \n%+v\n\n\n", Config)
	os.Exit(1)

	runScheduler()

}
