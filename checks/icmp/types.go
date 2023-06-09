package icmp

import (
	"time"
)

type IICMPCheck interface {
	RealExecute() (time.Duration, error)
}

type TICMPCheck struct {
	Project   string
	CheckName string

	Host    string
	Count   int
	Timeout string

	ErrorHeader string
}

const (
	ErrWrongCheckType = "Wrong check type: %s (should be icmp)"
	ErrEmptyHost      = "Host is empty"
	ErrICMPError      = "ICMP error at project %s, check: %s, host: %s"
	ErrPingError      = "ping error: %s"
	ErrPacketLoss     = "ping error: %f percent packet loss"
	ErrOther          = "other ping error: %s"
)
