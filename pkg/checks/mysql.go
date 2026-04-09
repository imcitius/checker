// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

// MySQL error constants
const (
	MySQLErrEmptyHost = "empty host"
)

// MySQLConfig contains common configuration for MySQL checks
type MySQLConfig struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

// MySQLQueryConfig extends the common config with query-specific settings
type MySQLQueryConfig struct {
	MySQLConfig
	Query    string `json:"query"`
	Response string `json:"response,omitempty"`
}

// MySQLTimeQueryConfig extends the common config with time query settings
type MySQLTimeQueryConfig struct {
	MySQLConfig
	Query      string `json:"query"`
	Difference string `json:"difference"`
}

// MySQLReplicationConfig extends the common config with replication settings
type MySQLReplicationConfig struct {
	MySQLConfig
	TableName  string   `json:"table_name,omitempty"`
	Lag        string   `json:"lag"`
	ServerList []string `json:"server_list"`
}

// MySQLCheck represents a basic MySQL health check.
type MySQLCheck struct {
	Host    string
	Port    int
	Timeout string
	Config  MySQLQueryConfig
	Logger  *logrus.Entry
}

// MySQLTimeCheck represents a MySQL time-based health check.
type MySQLTimeCheck struct {
	Host    string
	Port    int
	Timeout string
	Config  MySQLTimeQueryConfig
	Logger  *logrus.Entry
}

// MySQLReplicationCheck represents a MySQL replication check.
type MySQLReplicationCheck struct {
	Host    string
	Port    int
	Timeout string
	Config  MySQLReplicationConfig
	Logger  *logrus.Entry
}

// Run executes the MySQL query health check.
func (check *MySQLCheck) Run() (time.Duration, error) {
	var (
		id    string
		start = time.Now()
	)

	errorHeader := fmt.Sprintf("MySQL query error for host: %s", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ": " + MySQLErrEmptyHost)
	}

	// Set defaults
	dbuser := check.Config.UserName
	dbpassword := check.Config.Password
	dbhost := check.Host
	dbname := check.Config.DBName
	var dbport int
	if check.Port == 0 {
		dbport = 3306
	} else {
		dbport = check.Port
	}

	// Parse timeout
	dbConnectTimeout, err := parseCheckTimeout(check.Timeout, 5*time.Second)
	if err != nil {
		if check.Logger != nil {
			check.Logger.Errorf("Cannot parse timeout duration: %s", check.Timeout)
		}
		return time.Since(start), fmt.Errorf("invalid timeout: %v", err)
	}

	// Set default query if not provided
	query := check.Config.Query
	if query == "" {
		query = "SELECT 1;"
	}

	// Build connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
	}

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: The data source arguments are not valid", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: sql open: %w", errorHeader, err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Could not establish a connection with the database", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: connect: %w", errorHeader, err)
	}

	// Execute query
	err = db.QueryRow(query).Scan(&id)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Could not query database", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: query: %w", errorHeader, err)
	}

	// Verify response if expected
	if check.Config.Response != "" && id != check.Config.Response {
		return time.Since(start), fmt.Errorf("%s: db response does not match expected: %s (expected %s)", errorHeader, id, check.Config.Response)
	}

	return time.Since(start), nil
}

// Run executes the MySQL time-based health check.
func (check *MySQLTimeCheck) Run() (time.Duration, error) {
	var (
		id    int64
		start = time.Now()
	)

	errorHeader := fmt.Sprintf("MySQL time check error for host: %s", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ": " + MySQLErrEmptyHost)
	}

	// Set defaults
	dbuser := check.Config.UserName
	dbpassword := check.Config.Password
	dbhost := check.Host
	dbname := check.Config.DBName
	var dbport int
	if check.Port == 0 {
		dbport = 3306
	} else {
		dbport = check.Port
	}

	// Parse timeout
	dbConnectTimeout, err := parseCheckTimeout(check.Timeout, 5*time.Second)
	if err != nil {
		if check.Logger != nil {
			check.Logger.Errorf("Cannot parse timeout duration: %s", check.Timeout)
		}
		return time.Since(start), fmt.Errorf("invalid timeout: %v", err)
	}

	// Parse difference value
	dif, err := time.ParseDuration(check.Config.Difference)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Cannot parse difference value", errorHeader)
		}
		return time.Since(start), fmt.Errorf("invalid difference value: %v", err)
	}

	// Set default query if not provided
	query := check.Config.Query
	if query == "" {
		query = "SELECT UNIX_TIMESTAMP();"
	}

	// Build connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
	}

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: The data source arguments are not valid", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: sql open: %w", errorHeader, err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Could not establish a connection with the database", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: connect: %w", errorHeader, err)
	}

	// Execute query
	err = db.QueryRow(query).Scan(&id)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Could not query database", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: query: %w", errorHeader, err)
	}

	// Check time difference
	if dif > 0 {
		lastRecord := time.Unix(id, 0)
		curDif := time.Since(lastRecord)
		if curDif > dif {
			err := fmt.Errorf("%s: Unixtime difference error: got %v, difference %v", errorHeader, lastRecord, curDif)
			return time.Since(start), err
		}
	}

	return time.Since(start), nil
}

// Run executes the MySQL replication health check.
func (check *MySQLReplicationCheck) Run() (time.Duration, error) {
	var (
		recordId, recordValue, id int
		start                     = time.Now()
	)

	errorHeader := fmt.Sprintf("MySQL replication error for host: %s", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ": " + MySQLErrEmptyHost)
	}

	// Generate random test values
	recordId = rand.Intn(5-1) + 1
	recordValue = rand.Intn(9999-1) + 1

	// Set defaults
	dbUser := check.Config.UserName
	dbPassword := check.Config.Password
	dbHost := check.Host
	dbName := check.Config.DBName
	dbTable := check.Config.TableName
	if dbTable == "" {
		dbTable = "replication_test"
	}

	var dbPort int
	if check.Port == 0 {
		dbPort = 3306
	} else {
		dbPort = check.Port
	}

	// Parse timeout
	dbConnectTimeout, err := parseCheckTimeout(check.Timeout, 5*time.Second)
	if err != nil {
		if check.Logger != nil {
			check.Logger.Errorf("Cannot parse timeout duration: %s", check.Timeout)
		}
		return time.Since(start), fmt.Errorf("invalid timeout: %v", err)
	}

	// Build connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
	}

	// Connect to master database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: The data source arguments are not valid", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: sql open: %w", errorHeader, err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Could not establish a connection with the database", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: connect: %w", errorHeader, err)
	}

	// Insert test data
	insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON DUPLICATE KEY UPDATE test_value=%d, id=%d;"
	sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, recordId)

	_, err = db.Exec(sqlStatement)
	if err != nil {
		if check.Logger != nil {
			check.Logger.WithError(err).Errorf("%s: Failed to insert test data", errorHeader)
		}
		return time.Since(start), fmt.Errorf("%s: insert: %w", errorHeader, err)
	}

	// Allow replication to complete
	lagAllowed, err := time.ParseDuration(check.Config.Lag)
	if err != nil {
		if check.Logger != nil {
			check.Logger.Errorf("Error: Could not parse lag allowed: '%+v', use default 3s", err)
		}
		lagAllowed = 3 * time.Second
	}
	time.Sleep(lagAllowed)

	// Check each server in the replication chain
	for _, server := range check.Config.ServerList {
		selectSql := "SELECT test_value FROM %s WHERE id=%d;"
		sqlStatement := fmt.Sprintf(selectSql, dbTable, recordId)

		// Check if slave defined as `host:port`
		slaveHost, slavePort := server, dbPort
		host, port, err := net.SplitHostPort(server)
		if err == nil {
			slaveHost = host
			slavePort, err = strconv.Atoi(port)
			if err != nil {
				if check.Logger != nil {
					check.Logger.Warnf("Cannot parse slave port %s", err)
				}
				slavePort = dbPort
			}
		}

		// Connect to slave
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, slaveHost, slavePort, dbName)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
		}

		slaveDb, err := sql.Open("mysql", connStr)
		if err != nil {
			if check.Logger != nil {
				check.Logger.WithError(err).Errorf("%s: The data source arguments are not valid for server %s", errorHeader, slaveHost)
			}
			return time.Since(start), fmt.Errorf("%s: slave %s sql open: %w", errorHeader, slaveHost, err)
		}
		defer slaveDb.Close()

		// Verify connection
		err = slaveDb.Ping()
		if err != nil {
			if check.Logger != nil {
				check.Logger.WithError(err).Errorf("%s: Could not establish a connection with the database on server %s", errorHeader, slaveHost)
			}
			return time.Since(start), fmt.Errorf("%s: slave %s connect: %w", errorHeader, slaveHost, err)
		}

		// Query for the test value
		err = slaveDb.QueryRow(sqlStatement).Scan(&id)
		if err != nil {
			if check.Logger != nil {
				check.Logger.WithError(err).Errorf("%s: Could not query database on server %s", errorHeader, slaveHost)
			}
			return time.Since(start), fmt.Errorf("%s: slave %s query: %w", errorHeader, slaveHost, err)
		}

		// Verify replication worked correctly
		if id != recordValue {
			return time.Since(start), fmt.Errorf("%s: replication error: db response does not match expected: %d (expected %d) on server %s after %s",
				errorHeader, id, recordValue, slaveHost, lagAllowed)
		}
	}

	return time.Since(start), nil
}
