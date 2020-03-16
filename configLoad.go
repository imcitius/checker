package main

import (
	"encoding/json"
	"io/ioutil"
)

type configFile struct {
	Secs     int    `json:"secs"`
	BotToken string `json:"bot_token"`
}
type urlCheck struct {
	URL     string `json:"URL"`
	Channel int    `json:"channel"`
}

var config configFile
var probes []urlCheck

func jsonLoad(fileName string, destination interface{}) error {
	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFile, &destination)
	if err != nil {
		return err
	}
	return nil
}
