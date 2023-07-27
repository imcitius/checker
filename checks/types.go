package checks

import (
	"my/checker/alerts"
	"my/checker/config"
	"time"
)

type TProjectsConfig struct {
	Projects *map[string]config.TProject `mapstructure:"projects"`
}

type TCheckResult struct {
	Duration time.Duration
	Error    error
}

type TCommonCheck struct {
	Name string
	Sid  string
	UUID string

	Project     string
	Healthcheck string
	Type        string
	Alerter     alerts.ICommonAlerter

	Parameters  config.TCheckParameters
	CheckConfig config.TCheckConfig
	RealCheck   ISpecificCheck

	Result TCheckResult
}

type TChecksCollection struct {
	Checks []TCheckWithDuration
}

type TCheckWithDuration struct {
	Check    ICommonCheck
	Duration string
}
