package main

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

	initBots()
	runScheduler()

}
