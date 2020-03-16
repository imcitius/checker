package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

type urlCheck struct {
	URL string
}

func urlChecks(token string) {
	probes := []urlCheck{}
	for {
		<-time.After(2 * time.Second)

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
