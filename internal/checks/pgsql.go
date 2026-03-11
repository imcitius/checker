package checks

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"sort"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// PostgreSQLConfig contains common configuration for PostgreSQL checks
type PostgreSQLConfig struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode,omitempty"`
}

// PostgreSQLQueryConfig extends the common config with query-specific settings
type PostgreSQLQueryConfig struct {
	PostgreSQLConfig
	Query    string `json:"query"`
	Response string `json:"response,omitempty"`
}

// PostgreSQLTimeQueryConfig extends the common config with time query settings
type PostgreSQLTimeQueryConfig struct {
	PostgreSQLConfig
	Query      string `json:"query"`
	Difference string `json:"difference"`
}

// PostgreSQLReplicationConfig extends the common config with replication settings
type PostgreSQLReplicationConfig struct {
	PostgreSQLConfig
	TableName        string   `json:"table_name,omitempty"`
	Lag              string   `json:"lag"`
	ServerList       []string `json:"server_list"`
	AnalyticReplicas []string `json:"analytic_replicas,omitempty"`
}

// PostgreSQLCheck represents a basic PostgreSQL health check.
type PostgreSQLCheck struct {
	Host    string
	Port    int
	Timeout string
	Config  PostgreSQLQueryConfig
	Logger  *logrus.Entry
}

// PostgreSQLTimeCheck represents a PostgreSQL time-based health check.
type PostgreSQLTimeCheck struct {
	Host     string
	Port     int
	Timeout  string
	Config   PostgreSQLTimeQueryConfig
	TimeType string // "unixtime" or "timestamp"
	Logger   *logrus.Entry
}

// PostgreSQLReplicationCheck represents a PostgreSQL replication check.
type PostgreSQLReplicationCheck struct {
	Host      string
	Port      int
	Timeout   string
	Config    PostgreSQLReplicationConfig
	CheckType string // "replication" or "replication_status"
	Logger    *logrus.Entry
}

// Run executes the PostgreSQL query health check.
func (check *PostgreSQLCheck) Run() (time.Duration, error) {
	start := time.Now()
	var id string
	sslMode := "disable"

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "pgsql_query")
	}

	errorHeader := fmt.Sprintf("PGSQL query error for host: %s: ", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ErrEmptyHost)
	}

	dbUser := check.Config.UserName
	dbPassword := check.Config.Password
	dbHost := check.Host
	dbName := check.Config.DBName
	dbPort := check.Port
	if dbPort == 0 {
		dbPort = 5432
	}

	// Parse timeout
	if check.Timeout == "" {
		check.Timeout = "10s" // Default timeout
	}
	dbConnectTimeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		check.Logger.WithError(err).Error("Cannot parse timeout duration")
		return time.Since(start), fmt.Errorf(errorHeader+"cannot parse timeout: %v", err)
	}

	if check.Config.SSLMode != "" {
		sslMode = check.Config.SSLMode
	}

	query := check.Config.Query
	if query == "" {
		query = "SELECT 1;" // Default query
	}

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
	}

	check.Logger.Debugf("Connecting to PostgreSQL with: %s", connStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		check.Logger.WithError(err).Error("Error: The data source arguments are not valid")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not establish a connection with the database")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	// Execute query
	err = db.QueryRow(query).Scan(&id)
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not query database")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	// Check response if specified
	if check.Config.Response != "" && id != check.Config.Response {
		err = fmt.Errorf("error: db response does not match expected: %s (expected %s)", id, check.Config.Response)
		check.Logger.WithError(err).Error("Response validation failed")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	return time.Since(start), nil
}

// Run executes the PostgreSQL time-based health check.
func (check *PostgreSQLTimeCheck) Run() (time.Duration, error) {
	start := time.Now()
	sslMode := "disable"

	// Ensure logger is initialized
	if check.Logger == nil {
		checkType := "pgsql_timestamp"
		if check.TimeType == "unixtime" {
			checkType = "pgsql_unixtime"
		}
		check.Logger = logrus.WithField("check", checkType)
	}

	errorHeader := fmt.Sprintf("PGSQL %s check error for host: %s: ", check.TimeType, check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ErrEmptyHost)
	}

	dbUser := check.Config.UserName
	dbPassword := check.Config.Password
	dbHost := check.Host
	dbName := check.Config.DBName
	dbPort := check.Port
	if dbPort == 0 {
		dbPort = 5432
	}

	// Parse timeout
	if check.Timeout == "" {
		check.Timeout = "10s" // Default timeout
	}
	dbConnectTimeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		check.Logger.WithError(err).Error("Cannot parse timeout duration")
		return time.Since(start), fmt.Errorf(errorHeader+"cannot parse timeout: %v", err)
	}

	if check.Config.SSLMode != "" {
		sslMode = check.Config.SSLMode
	}

	// Parse difference
	dif, err := time.ParseDuration(check.Config.Difference)
	if err != nil {
		check.Logger.WithError(err).Error("Cannot parse difference value")
		return time.Since(start), fmt.Errorf(errorHeader+"cannot parse difference value: %v", err)
	}

	query := check.Config.Query
	if query == "" {
		query = "SELECT 1;" // Default query
	}

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
	}

	check.Logger.Debugf("Connecting to PostgreSQL with: %s", connStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		check.Logger.WithError(err).Error("Error: The data source arguments are not valid")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not establish a connection with the database")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	// Execute query based on type
	if check.TimeType == "unixtime" {
		var id int64
		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			check.Logger.WithError(err).Error("Error: Could not query database")
			return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
		}

		if dif > 0 {
			lastRecord := time.Unix(id, 0)
			curDif := time.Since(lastRecord)
			check.Logger.Debugf("Last record time: %v, difference: %v", lastRecord, curDif)
			if curDif > dif {
				err := fmt.Errorf("unixtime difference error: got %v, difference %v", lastRecord, curDif)
				check.Logger.WithError(err).Error("Time difference check failed")
				return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
			}
		}
	} else {
		// timestamp type
		var timestamp time.Time
		err = db.QueryRow(query).Scan(&timestamp)
		if err != nil {
			check.Logger.WithError(err).Error("Error: Could not query database")
			return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
		}

		lastRecord := timestamp
		curDif := time.Since(lastRecord)
		check.Logger.Debugf("Last record time: %v, difference: %v", lastRecord, curDif)
		if curDif > dif {
			err := fmt.Errorf("timestamp difference error: got %v, difference %v", lastRecord, curDif)
			check.Logger.WithError(err).Error("Time difference check failed")
			return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
		}
	}

	return time.Since(start), nil
}

// Run executes the PostgreSQL replication health check.
func (check *PostgreSQLReplicationCheck) Run() (time.Duration, error) {
	start := time.Now()
	sslMode := "disable"

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "pgsql_replication")
	}

	errorHeader := fmt.Sprintf("PGSQL replication check error for host: %s: ", check.Host)

	if check.Host == "" {
		return time.Since(start), errors.New(errorHeader + ErrEmptyHost)
	}

	// If it's a replication status check
	if check.CheckType == "replication_status" {
		return check.runReplicationStatusCheck(start)
	}

	// Regular replication check
	dbUser := check.Config.UserName
	dbPassword := check.Config.Password
	dbHost := check.Host
	dbName := check.Config.DBName
	dbPort := check.Port
	if dbPort == 0 {
		dbPort = 5432
	}

	// Parse timeout
	if check.Timeout == "" {
		check.Timeout = "10s" // Default timeout
	}
	dbConnectTimeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		check.Logger.WithError(err).Error("Cannot parse timeout duration")
		return time.Since(start), fmt.Errorf(errorHeader+"cannot parse timeout: %v", err)
	}

	if check.Config.SSLMode != "" {
		sslMode = check.Config.SSLMode
	}

	// Set table name
	dbTable := "repl_test"
	if check.Config.TableName != "" {
		dbTable = check.Config.TableName
	}

	// Generate test data
	recordId := rand.Intn(5)
	recordValue := rand.Intn(9999)

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
	}

	check.Logger.Debugf("Connecting to PostgreSQL master with: %s", connStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		check.Logger.WithError(err).Error("Error: The data source arguments are not valid")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not establish a connection with the database")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	// Insert/update test record on master
	check.Logger.Debugf("Set info on master, id: %d, value: %d", recordId, recordValue)
	insertSql := "INSERT INTO %s (id, test_value, timestamp) VALUES (%d, %d, now()) ON CONFLICT (id) DO UPDATE SET test_value=%d, timestamp=now() WHERE %s.id=%d;"
	sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, dbTable, recordId)

	_, err = db.Exec(sqlStatement)
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not insert/update test record")
		return time.Since(start), fmt.Errorf(errorHeader+"pgsql insert error: %v", err)
	}

	// Allow replication to propagate
	lagAllowed, err := time.ParseDuration(check.Config.Lag)
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not parse lag allowed, using default 3s")
		lagAllowed = 3 * time.Second
	}
	time.Sleep(lagAllowed)

	// Check replication to slaves
	for _, slave := range check.Config.ServerList {
		var host string
		var slavePort = dbPort

		// Check if slave is defined as host:port
		host, port, err := net.SplitHostPort(slave)
		if err == nil {
			slavePort, err = strconv.Atoi(port)
			if err != nil {
				check.Logger.WithError(err).Warn("Cannot parse slave port, using default")
			}
		} else {
			host = slave
		}

		// Build select query
		selectSql := "SELECT test_value FROM %s WHERE %s.id=%d;"
		sqlStatement := fmt.Sprintf(selectSql, dbTable, dbTable, recordId)

		check.Logger.Debugf("Reading from slave %s", host)

		// Connect to slave
		slaveConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dbUser, dbPassword, host, slavePort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			slaveConnStr = slaveConnStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		db, err := sql.Open("postgres", slaveConnStr)
		if err != nil {
			check.Logger.WithError(err).Error("Error: Could not connect to slave")
			return time.Since(start), fmt.Errorf(errorHeader+"slave connection error: %v", err)
		}
		defer db.Close()

		err = db.Ping()
		if err != nil {
			check.Logger.WithError(err).Error("Error: Could not establish a connection with the slave")
			return time.Since(start), fmt.Errorf(errorHeader+"slave ping error: %v", err)
		}

		// Query slave for the test value
		var id int
		err = db.QueryRow(sqlStatement).Scan(&id)
		if err != nil {
			check.Logger.WithError(err).Error("Error: Could not query slave database")
			return time.Since(start), fmt.Errorf(errorHeader+"slave query error: %v", err)
		}

		// Check if value matches master
		if id != recordValue {
			err = fmt.Errorf("replication error: db response does not match expected: %d (expected %d) on server %s after %s",
				id, recordValue, host, lagAllowed)
			check.Logger.WithError(err).Error("Replication check failed")
			return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
		}
	}

	return time.Since(start), nil
}

// runReplicationStatusCheck executes the PostgreSQL replication status check
func (check *PostgreSQLReplicationCheck) runReplicationStatusCheck(start time.Time) (time.Duration, error) {
	type repStatus struct {
		pid             sql.NullInt32
		usesysid        sql.NullInt32
		usename         sql.NullString
		applicationName sql.NullString
		clientAddr      sql.NullString
		clientHostname  sql.NullString
		clientPort      sql.NullInt32
		backendStart    sql.NullString
		backendXmin     sql.NullString
		state           sql.NullString
		sentLsn         sql.NullString
		writeLsn        sql.NullString
		flushLsn        sql.NullString
		replayLsn       sql.NullString
		writeLag        sql.NullString
		flushLag        sql.NullString
		replayLag       sql.NullString
		syncPriority    sql.NullInt32
		syncState       sql.NullString
		replyTime       sql.NullTime
	}

	var (
		dbTable                               = "pg_stat_replication"
		dbName                                = "postgres"
		sslMode                               = "disable"
		repStatusReply                        []repStatus
		hours, minutes, seconds, microseconds int
		streaming, isAnalytic                 bool
	)

	errorHeader := fmt.Sprintf("PGSQL replication status check error for host: %s: ", check.Host)

	dbUser := check.Config.UserName
	dbPassword := check.Config.Password
	dbHost := check.Host
	dbPort := check.Port
	if dbPort == 0 {
		dbPort = 5432
	}

	// Parse timeout
	if check.Timeout == "" {
		check.Timeout = "10s" // Default timeout
	}
	dbConnectTimeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		check.Logger.WithError(err).Error("Cannot parse timeout duration")
		return time.Since(start), fmt.Errorf(errorHeader+"cannot parse timeout: %v", err)
	}

	if check.Config.SSLMode != "" {
		sslMode = check.Config.SSLMode
	}

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
	}

	check.Logger.Debugf("Connecting to PostgreSQL with: %s", connStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		check.Logger.WithError(err).Error("Error: The data source arguments are not valid")
		return time.Since(start), fmt.Errorf(errorHeader+"sql.Open error: %v", err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not establish a connection with the database")
		return time.Since(start), fmt.Errorf(errorHeader+"db.Ping error: %v", err)
	}

	// Query replication status
	check.Logger.Debug("Getting replication status from master...")
	sqlQuery := fmt.Sprintf("SELECT * FROM %s;", dbTable)

	rows, err := db.Query(sqlQuery)
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not query database")
		return time.Since(start), fmt.Errorf(errorHeader+"repstatus sql.Query error: %v", err)
	}
	defer rows.Close()

	// Scan results
	var rowsCount int
	for rows.Next() {
		var reply repStatus
		err := rows.Scan(
			&reply.pid,
			&reply.usesysid,
			&reply.usename,
			&reply.applicationName,
			&reply.clientAddr,
			&reply.clientHostname,
			&reply.clientPort,
			&reply.backendStart,
			&reply.backendXmin,
			&reply.state,
			&reply.sentLsn,
			&reply.writeLsn,
			&reply.flushLsn,
			&reply.replayLsn,
			&reply.writeLag,
			&reply.flushLag,
			&reply.replayLag,
			&reply.syncPriority,
			&reply.syncState,
			&reply.replyTime,
		)
		if err != nil {
			check.Logger.WithError(err).Error("Error scanning replication status row")
			return time.Since(start), fmt.Errorf(errorHeader+"repstatus rows.Scan error: %v", err)
		}
		repStatusReply = append(repStatusReply, reply)
		rowsCount++
	}

	if rowsCount == 0 {
		check.Logger.Error("Error: no rows in query result")
		return time.Since(start), fmt.Errorf("%srepstatus no rows in query result", errorHeader)
	}

	err = rows.Err()
	if err != nil {
		check.Logger.WithError(err).Error("Error in rows iteration")
		return time.Since(start), fmt.Errorf(errorHeader+"repstatus rows.Err error: %v", err)
	}

	// Analyze replication status
	for i, reply := range repStatusReply {
		if reply.state.String == "streaming" {
			streaming = true

			allowedLag, err := time.ParseDuration(check.Config.Lag)
			if err != nil {
				check.Logger.WithError(err).Error("Error parsing allowed lag")
				return time.Since(start), fmt.Errorf(errorHeader+"allowed lag parsing error: %v", err)
			}

			if reply.replayLag.Valid {
				_, err = fmt.Sscanf(reply.replayLag.String, "%d:%d:%d.%d", &hours, &minutes, &seconds, &microseconds)
				if err != nil {
					errMsg := fmt.Sprintf("Error scanning replay_lag: %v, replay_lag: '%s'", err, reply.replayLag.String)
					check.Logger.Error(errMsg)
					check.Logger.Errorf("Rep status reply row #%d", i)

					fields := reflect.ValueOf(reply)
					for j := 0; j < fields.NumField(); j++ {
						check.Logger.Errorf("Rep status reply field: '%s'\tValue: '%s'",
							fields.Type().Field(j).Name, fields.Field(j))
					}

					return time.Since(start), fmt.Errorf(errorHeader+"%s", errMsg)
				}
			}

			// Parse actual lag
			lag, err := time.ParseDuration(fmt.Sprintf("%dh%dm%ds%dus", hours, minutes, seconds, microseconds))
			if err != nil {
				check.Logger.WithError(err).Error("Error parsing replay_lag")
				return time.Since(start), fmt.Errorf(errorHeader+"replay_lag parsing error: %v", err)
			}

			// Check if replica is in AnalyticReplicas list
			isAnalytic = false
			if len(check.Config.AnalyticReplicas) > 0 {
				appName := reply.applicationName.String
				isAnalytic = func(s []string, searchterm string) bool {
					i := sort.SearchStrings(s, searchterm)
					return i < len(s) && s[i] == searchterm
				}(check.Config.AnalyticReplicas, appName)
			}

			// Check if actual lag is more than allowed
			if lag > allowedLag && !isAnalytic {
				err := fmt.Errorf("replay_lag more than %s detected on %s: %s",
					allowedLag.String(), reply.applicationName.String, lag.String())
				check.Logger.Error(err)
				return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
			}
		}
	}

	if !streaming {
		check.Logger.Error("Replication is not streaming")
		return time.Since(start), fmt.Errorf("%sReplication is not streaming", errorHeader)
	}

	return time.Since(start), nil
}
