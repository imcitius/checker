package models

import (
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
