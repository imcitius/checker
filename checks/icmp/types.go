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
	ErrWrongCheckType = "wrong check type: %s (should be icmp)"
	ErrEmptyHost      = "host is empty"
	ErrICMPError      = "ICMP error: "
	ErrPingError      = "ping error: %s"
	ErrPacketLoss     = "ping error: %f percent packet loss"
	ErrOther          = "other ping error: %s"
)
