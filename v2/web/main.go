package web

import (
	"my/checker/config"
)

var (
	logger     = config.GetLog()
	configurer = config.GetConfig()
	//cache      *memoize.Memoizer
)
