package store

import (
	"context"
	"my/checker/config"
)

var (
	logger     = config.GetLog()
	configurer = config.GetConfig()

	Store     IStore
	DBContext context.Context
)
