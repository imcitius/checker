package http

import "my/checker/config"

var (
	configurer = config.GetConfig()
	//Defaults = config.GetDefaults()
	logger = config.GetLog()
)
