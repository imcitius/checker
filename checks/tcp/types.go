package tcp

import (
	"time"
)

type ITCPCheck interface {
	RealExecute() (time.Duration, error)
}

type TTCPCheck struct {
	Project   string
	CheckName string

	Host    string
	Port    int
	Count   int
	Timeout string

	ErrorHeader string
}

const (
	ErrTCPError       = "TCP error: "
	ErrWrongCheckType = "wrong check type: %s (should be tcp)"
	ErrEmptyHost      = "host is empty"
	ErrEmptyPort      = "port is empty"
	ErrConnectError   = "connect error: %s (attempt %d of %d). %s."
	//ErrPacketLoss     = "ping error: %f percent packet loss"
	ErrOther = "other error: %s"
)
