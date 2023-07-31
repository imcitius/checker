package store

import (
	"time"
)

type DbCheckObject struct {
	UUid        string
	Project     string
	Healthcheck string
	Name        string
	LastPing    time.Time
	LastExec    time.Time
	LastResult  bool
}
