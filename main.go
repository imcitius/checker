package main

import (
	"encoding/json"
	"flag"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

var log = logrus.New()

func main() {

	configPath := flag.String("config", "config.json", "Config file path")
	debugLevel := flag.String("debug", "Info", "Debug,Info,Warn,Error,Fatal,Panic")
	flag.Parse()

	dl, err := logrus.ParseLevel(*debugLevel)
	if err != nil {
		log.Panicf("Cannot parse debug level: %v", err)
	} else {
		log.SetLevel(dl)
	}

	log.Infof("Config file path: %s", *configPath)

	err = jsonLoad(*configPath, &Config)
	if err != nil {
		log.Panicf("Config load error: %v", err)
	} else {
		err = fillDefaults()
		if err != nil {
			panic(err)
		}
		fillUUIDs()
		if err != nil {
			panic(err)
		}
		fillTimeouts()
	}

	if *debugLevel == "Debug" {
		conf, _ := json.Marshal(Config)
		log.Debugf("Config: %+v\n\n", string(conf))
	}

	rand.Seed(time.Now().UnixNano())

	initBots()
	runScheduler()

}
