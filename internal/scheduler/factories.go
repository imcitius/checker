package scheduler

import (
	"fmt"

	"checker/internal/actors"
	"checker/internal/checks"
	"checker/internal/models"

	"github.com/sirupsen/logrus"
)

// CheckerFactory creates Checker instances based on the CheckDefinition.
// Returns nil and logs a warning if the check type is unknown.
func CheckerFactory(checkDef models.CheckDefinition, logger *logrus.Entry) checks.Checker {
	if logger == nil {
		logger = logrus.WithField("function", "CheckerFactory")
	}

	if checkDef.Config == nil {
		logger.Warnf("Check definition %s has nil config", checkDef.UUID)
		return nil
	}

	switch config := checkDef.Config.(type) {
	case *models.HTTPCheckConfig:
		logger.Debugf("Creating HTTP check for URL: %s", config.URL)
		return &checks.HTTPCheck{
			URL:                 config.URL,
			Timeout:             config.Timeout,
			Answer:              config.Answer,
			AnswerPresent:       config.AnswerPresent,
			Code:                config.Code,
			Auth:                struct{ User, Password string }{config.Auth.User, config.Auth.Password},
			Headers:             config.Headers,
			SkipCheckSSL:        config.SkipCheckSSL,
			SSLExpirationPeriod: config.SSLExpirationPeriod,
			StopFollowRedirects: config.StopFollowRedirects,
			Logger:              logger,
		}
	case *models.TCPCheckConfig:
		logger.Debugf("Creating TCP check for host: %s, port: %d", config.Host, config.Port)
		return &checks.TCPCheck{
			Host:    config.Host,
			Port:    config.Port,
			Timeout: config.Timeout,
		}
	case *models.ICMPCheckConfig:
		logger.Debugf("Creating ICMP check for host: %s", config.Host)
		return &checks.ICMPCheck{
			Host:    config.Host,
			Count:   config.Count,
			Timeout: config.Timeout,
		}
	case *models.PassiveCheckConfig:
		logger.Debugf("Creating Passive check")
		return &checks.PassiveCheck{
			LastPing:    checkDef.LastRun,
			Timeout:     config.Timeout,
			ErrorHeader: fmt.Sprintf("Passive check '%s' [%s/%s]", checkDef.Name, checkDef.Project, checkDef.GroupName),
			Logger:      logger,
		}
	case *models.MySQLCheckConfig:
		switch checkDef.Type {
		case "mysql_query":
			logger.Debugf("Creating MySQL query check for host: %s, port: %d", config.Host, config.Port)
			return &checks.MySQLCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: checks.MySQLQueryConfig{
					MySQLConfig: checks.MySQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
					},
					Query:    config.Query,
					Response: config.Response,
				},
				Logger: logger,
			}
		case "mysql_query_unixtime":
			logger.Debugf("Creating MySQL unixtime check for host: %s, port: %d", config.Host, config.Port)
			return &checks.MySQLTimeCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: checks.MySQLTimeQueryConfig{
					MySQLConfig: checks.MySQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
					},
					Query:      config.Query,
					Difference: config.Difference,
				},
				Logger: logger,
			}
		case "mysql_replication":
			logger.Debugf("Creating MySQL replication check for host: %s, port: %d", config.Host, config.Port)
			return &checks.MySQLReplicationCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: checks.MySQLReplicationConfig{
					MySQLConfig: checks.MySQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
					},
					TableName:  config.TableName,
					Lag:        config.Lag,
					ServerList: config.ServerList,
				},
				Logger: logger,
			}
		default:
			logger.Warnf("Unknown MySQL check type: %s", checkDef.Type)
			return nil
		}

	case *models.PostgreSQLCheckConfig:
		switch checkDef.Type {
		case "pgsql_query":
			logger.Debugf("Creating PostgreSQL query check for host: %s, port: %d", config.Host, config.Port)
			return &checks.PostgreSQLCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: checks.PostgreSQLQueryConfig{
					PostgreSQLConfig: checks.PostgreSQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
						SSLMode:  config.SSLMode,
					},
					Query:    config.Query,
					Response: config.Response,
				},
				Logger: logger,
			}
		case "pgsql_query_unixtime", "pgsql_query_timestamp":
			timeType := "timestamp"
			if checkDef.Type == "pgsql_query_unixtime" {
				timeType = "unixtime"
			}
			logger.Debugf("Creating PostgreSQL %s check for host: %s, port: %d", timeType, config.Host, config.Port)
			return &checks.PostgreSQLTimeCheck{
				Host:     config.Host,
				Port:     config.Port,
				Timeout:  config.Timeout,
				TimeType: timeType,
				Config: checks.PostgreSQLTimeQueryConfig{
					PostgreSQLConfig: checks.PostgreSQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
						SSLMode:  config.SSLMode,
					},
					Query:      config.Query,
					Difference: config.Difference,
				},
				Logger: logger,
			}
		case "pgsql_replication", "pgsql_replication_status":
			checkType := "replication"
			if checkDef.Type == "pgsql_replication_status" {
				checkType = "replication_status"
			}
			logger.Debugf("Creating PostgreSQL %s check for host: %s, port: %d", checkType, config.Host, config.Port)
			return &checks.PostgreSQLReplicationCheck{
				Host:      config.Host,
				Port:      config.Port,
				Timeout:   config.Timeout,
				CheckType: checkType,
				Config: checks.PostgreSQLReplicationConfig{
					PostgreSQLConfig: checks.PostgreSQLConfig{
						UserName: config.UserName,
						Password: config.Password,
						DBName:   config.DBName,
						SSLMode:  config.SSLMode,
					},
					TableName:        config.TableName,
					Lag:              config.Lag,
					ServerList:       config.ServerList,
					AnalyticReplicas: config.AnalyticReplicas,
				},
				Logger: logger,
			}
		default:
			logger.Warnf("Unknown PostgreSQL check type: %s", checkDef.Type)
			return nil
		}

	case *models.MongoDBCheckConfig:
		logger.Debugf("Creating MongoDB check for URI: %s", config.URI)
		return &checks.MongoDBCheck{
			URI:     config.URI,
			Timeout: config.Timeout,
			Logger:  logger,
		}

	case *models.DomainExpiryCheckConfig:
		logger.Debugf("Creating Domain Expiry check for domain: %s", config.Domain)
		return &checks.DomainExpiryCheck{
			Domain:            config.Domain,
			ExpiryWarningDays: config.ExpiryWarningDays,
			Timeout:           config.Timeout,
			Logger:            logger,
		}

	default:
		logger.Warnf("Unknown check config type: %T for type %s", checkDef.Config, checkDef.Type)
		return nil
	}
}

// ActorFactory creates Actor instances based on the CheckDefinition.
// Returns nil and logs a warning if the actor type is unknown.
func ActorFactory(checkDef models.CheckDefinition) (actors.Actor, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function":  "ActorFactory",
		"actorType": checkDef.ActorType,
		"uuid":      checkDef.UUID,
	})

	switch checkDef.ActorType {
	case "log":
		logger.Debugf("Creating Log actor")
		return &actors.LogActor{}, nil
	case "webhook":
		logger.Debugf("Creating Webhook actor")
		config, ok := checkDef.ActorConfig.(*models.WebhookConfig)
		if !ok || config == nil {
			return nil, fmt.Errorf("webhook actor config is missing or invalid")
		}

		return &actors.WebhookActor{
			URL:     config.URL,
			Method:  config.Method,
			Payload: config.Payload,
			Headers: config.Headers,
			Logger:  logger,
		}, nil
	case "":
		logger.Debug("No actor type specified, skipping actor creation")
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown actor type: %s", checkDef.ActorType)
	}
}
