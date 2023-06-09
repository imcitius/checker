package checks

import "time"

type ICommonCheck interface {
	Execute() (time.Duration, error)

	GetProject() string
	GetHealthcheck() string
	GetHost() string
	GetType() string
	GetSID() string
}

type ISpecificCheck interface {
	RealExecute() (time.Duration, error)
}

type IProject interface{}
type IHealthcheck interface{}
