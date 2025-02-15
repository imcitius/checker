package config

import (
	checks "my/checker/models/checks"
	alerts "my/checker/models/alerts"
)

type Configurer interface {
    GetDBConnectionString() string
    SetDBPassword(string)
    GetAllChecks() ([]checks.TCheckConfig, error)
    GetCheckByUUid(string) (checks.TCheckConfig, error)
    UpdateCheckByUUID(checks.TCheckConfig) error
    GetDB() DBConfig
    SetDBConnected()
    GetAlerts() []alerts.TAlert
}
