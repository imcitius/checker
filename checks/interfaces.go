package checks

import (
	"my/checker/config"
	"time"
)

type ICommonCheck interface {
	Execute() TCommonCheck

	GetProject() string
	GetHealthcheck() string
	GetName() string
	GetHost() string
	GetType() string
	GetSID() string
	GetUUID() string
	GetCheckDetails() config.TCheckDetails
	SetStatus(bool)
}

type ISpecificCheck interface {
	RealExecute() (time.Duration, error)
}

type IProject interface{}
type IHealthcheck interface{}
