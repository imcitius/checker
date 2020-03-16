package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Config struct {
	Secs     int    `json:"secs"`
	BotToken string `json:"bot_token"`
}
type urlCheck struct {
	URL     string `json:"URL"`
	Channel int    `json:"channel"`
}

var config Config
var probes []urlCheck

func jsonLoad(fileName string, destination interface{}) error {
	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	err = json.Unmarshal(configFile, &destination)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func main() {
	jsonLoad("config.json", &config)
	jsonLoad("data.json", &probes)

	go runListenBot(config.BotToken)
	urlChecks(config.BotToken)
}
