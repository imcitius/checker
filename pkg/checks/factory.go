package checks

import (
	"fmt"

	"checker/pkg/models"

	"github.com/sirupsen/logrus"
)

// CheckerFactory creates Checker instances based on the CheckDefinition.
// Returns nil and logs a warning if the check type is unknown.
func CheckerFactory(checkDef models.CheckDefinition, logger *logrus.Entry) Checker {
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
		return &HTTPCheck{
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
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &TCPCheck{
			Host:    config.Host,
			Port:    config.Port,
			Timeout: config.Timeout,
		}
	case *models.SSHCheckConfig:
		port := config.Port
		if port == 0 {
			port = DefaultSSHPort
		}
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		logger.Debugf("Creating SSH check for host: %s, port: %d", config.Host, port)
		return &SSHCheck{
			Host:         config.Host,
			Port:         config.Port,
			Timeout:      config.Timeout,
			ExpectBanner: config.ExpectBanner,
			Logger:       logger,
		}
	case *models.ICMPCheckConfig:
		logger.Debugf("Creating ICMP check for host: %s", config.Host)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &ICMPCheck{
			Host:    config.Host,
			Count:   config.Count,
			Timeout: config.Timeout,
		}
	case *models.DNSCheckConfig:
		logger.Debugf("Creating DNS check for domain: %s, record type: %s", config.Domain, config.RecordType)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &DNSCheck{
			Host:       config.Host,
			Domain:     config.Domain,
			RecordType: config.RecordType,
			Timeout:    config.Timeout,
			Expected:   config.Expected,
			Logger:     logger,
		}
	case *models.PassiveCheckConfig:
		logger.Debugf("Creating Passive check")
		return &PassiveCheck{
			LastPing:    checkDef.LastRun,
			Timeout:     config.Timeout,
			ErrorHeader: fmt.Sprintf("Passive check '%s' [%s/%s]", checkDef.Name, checkDef.Project, checkDef.GroupName),
			Logger:      logger,
		}
	case *models.MySQLCheckConfig:
		switch checkDef.Type {
		case "mysql_query":
			logger.Debugf("Creating MySQL query check for host: %s, port: %d", config.Host, config.Port)
			return &MySQLCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
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
			return &MySQLTimeCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: MySQLTimeQueryConfig{
					MySQLConfig: MySQLConfig{
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
			return &MySQLReplicationCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: MySQLReplicationConfig{
					MySQLConfig: MySQLConfig{
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
			return &PostgreSQLCheck{
				Host:    config.Host,
				Port:    config.Port,
				Timeout: config.Timeout,
				Config: PostgreSQLQueryConfig{
					PostgreSQLConfig: PostgreSQLConfig{
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
			return &PostgreSQLTimeCheck{
				Host:     config.Host,
				Port:     config.Port,
				Timeout:  config.Timeout,
				TimeType: timeType,
				Config: PostgreSQLTimeQueryConfig{
					PostgreSQLConfig: PostgreSQLConfig{
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
			return &PostgreSQLReplicationCheck{
				Host:      config.Host,
				Port:      config.Port,
				Timeout:   config.Timeout,
				CheckType: checkType,
				Config: PostgreSQLReplicationConfig{
					PostgreSQLConfig: PostgreSQLConfig{
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

	case *models.RedisCheckConfig:
		logger.Debugf("Creating Redis check for host: %s, port: %d", config.Host, config.Port)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &RedisCheck{
			Host:     config.Host,
			Port:     config.Port,
			Timeout:  config.Timeout,
			Password: config.Password,
			DB:       config.DB,
			Logger:   logger,
		}

	case *models.MongoDBCheckConfig:
		logger.Debugf("Creating MongoDB check for URI: %s", config.URI)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &MongoDBCheck{
			URI:     config.URI,
			Timeout: config.Timeout,
			Logger:  logger,
		}

	case *models.DomainExpiryCheckConfig:
		logger.Debugf("Creating Domain Expiry check for domain: %s", config.Domain)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &DomainExpiryCheck{
			Domain:            config.Domain,
			ExpiryWarningDays: config.ExpiryWarningDays,
			Timeout:           config.Timeout,
			Logger:            logger,
		}

	case *models.SSLCertCheckConfig:
		logger.Debugf("Creating SSL cert check for host: %s, port: %d", config.Host, config.Port)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &SSLCertCheck{
			Host:              config.Host,
			Port:              config.Port,
			Timeout:           config.Timeout,
			ExpiryWarningDays: config.ExpiryWarningDays,
			ValidateChain:     config.ValidateChain,
			Logger:            logger,
		}

	case *models.SMTPCheckConfig:
		logger.Debugf("Creating SMTP check for host: %s, port: %d", config.Host, config.Port)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &SMTPCheck{
			Host:     config.Host,
			Port:     config.Port,
			Timeout:  config.Timeout,
			StartTLS: config.StartTLS,
			Username: config.Username,
			Password: config.Password,
		}

	case *models.GRPCHealthCheckConfig:
		logger.Debugf("Creating gRPC health check for host: %s", config.Host)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &GRPCHealthCheck{
			Host:    config.Host,
			UseTLS:  config.UseTLS,
			Timeout: config.Timeout,
			Logger:  logger,
		}

	case *models.WebSocketCheckConfig:
		logger.Debugf("Creating WebSocket check for URL: %s", config.URL)
		if config.Timeout == "" {
			config.Timeout = "10s"
		}
		return &WebSocketCheck{
			URL:     config.URL,
			Timeout: config.Timeout,
		}

	default:
		logger.Warnf("Unknown check config type: %T for type %s", checkDef.Config, checkDef.Type)
		return nil
	}
}
