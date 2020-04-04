package main

import (
	"math/rand"
	"time"
)

func main() {
	err := jsonLoad("config.json", &Config)
	if err != nil {
		panic(err)
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
	//conf, _ := json.Marshal(Config)
	//log.Printf("Config: %+v\n\n", string(conf))

	rand.Seed(time.Now().UnixNano())

	initBots()
	runScheduler()

}
