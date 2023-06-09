package checks

import "my/checker/config"

type TProjectsConfig struct {
	Projects *map[string]config.TProject `mapstructure:"projects"`
}

type TCommonCheck struct {
	Name string
	Sid  string
	UUID string

	Project     string
	Healthcheck string
	Type        string

	Parameters  config.TCheckParameters
	CheckConfig config.TCheckConfig
	RealCheck   ISpecificCheck
}

type TChecksCollection struct {
	Checks map[string][]TCheckWithDuration
}

type TCheckWithDuration struct {
	Check    ICommonCheck
	Duration string
}
