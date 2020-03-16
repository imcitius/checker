package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Secs     int    `json:"secs"`
	BotToken string `json:"bot_token"`
}

var config Config

func main() {

	configFile, err := os.Open("config.json")
	defer configFile.Close()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	go runListenBot(config.BotToken)
	urlChecks(config.BotToken)

}
