package passive

import (
	"time"
)

type IPassiveCheck interface {
	RealExecute() (time.Duration, error)
}

type TPassiveCheck struct {
	Project   string
	CheckName string

	Timeout string
	UUid    string
}

const (
	ErrPassiveError      = "Passive error: "
	ErrWrongCheckType    = "wrong check type: %s (should be passive)"
	ErrEmptyTimeout      = "timeout is empty"
	ErrCheckNotFound     = "Requested check not found in DB: %s"
	ErrTimeoutParseError = "timeout parse error: %s"
	ErrCheckExpired      = "Last ping is too old: %s, at %s"
	//ErrLastPingParseError = "last ping parse error: %s"
	//ErrConnectError   = "connect error: %s (attempt %d of %d). %s."
	//ErrPacketLoss     = "ping error: %f percent packet loss"
	//ErrOther = "other error: %s"
)
