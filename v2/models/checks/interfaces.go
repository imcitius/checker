package checks

import (
	"context"
	"time"
)

type ICommonCheck interface {
	Execute() TCommonCheck
	Alert(ctx context.Context, message string) TCommonCheck
	GetProject() string
	GetHealthcheck() string
	GetName() string
	GetHost() string
	GetType() string
	GetSID() string
	GetUUID() string
	GetCheckDetails() TCheckDetails
	SetStatus(bool)
	GetEnabled() bool
}

type ISpecificCheck interface {
	RealExecute() (time.Duration, error)
}
