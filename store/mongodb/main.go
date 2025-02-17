package store

import (
	"context"
)

var (
	// logger     = config.GetLog()
	// configurer = config.GetConfig()

	Store     IStore
	DBContext context.Context
)
