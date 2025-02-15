package store

import (
	"context"

	"github.com/sirupsen/logrus"
)

var (
	logger     = logrus.New()

	Store     IStore
	DBContext context.Context
)
