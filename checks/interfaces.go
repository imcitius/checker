package checks

import "time"

type ICommonCheck interface {
	Execute() TCommonCheck

	GetProject() string
	GetHealthcheck() string
	GetHost() string
	GetType() string
	GetSID() string
	SetStatus(bool)
}

type ISpecificCheck interface {
	RealExecute() (time.Duration, error)
}

type IProject interface{}
type IHealthcheck interface{}
