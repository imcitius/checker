package store

import (
	"my/checker/config"
	"sync"
	"time"
)

type DbCheckObject struct {
	UUID        string
	Project     string
	Healthcheck string
	Name        string
	LastPing    time.Time
	LastExec    time.Time
	LastResult  bool
	Enabled     bool
}

type MessagesContextStorage struct {
	sync.RWMutex

	data map[int64]map[int]config.TAlertDetails
}
