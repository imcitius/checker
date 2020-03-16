package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

type urlCheck struct {
	URL     string `json:"URL"`
	Channel int    `json:"channel"`
}

func urlChecks(token string) {
	probes := []urlCheck{}
	for {
		<-time.After(time.Duration(config.Secs) * time.Second)

		data, err := ioutil.ReadFile("data.json")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = json.Unmarshal(data, &probes)
		if err != nil {
			fmt.Println(err)
			return
		}

		testurls(probes)
	}
}
