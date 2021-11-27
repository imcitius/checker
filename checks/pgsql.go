package check

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"math/rand"
	"my/checker/config"
	projects "my/checker/projects"
	"net"
	"reflect"
	"strconv"
	"time"
)

func init() {
	Checks["pgsql_query"] = func(c *config.Check, p *projects.Project) (ret error) {

		var (
			id, query string
			dbPort    int
			sslMode   = "disable"
		)

		defer func() {
			if err := recover(); err != nil {
				errorHeader := fmt.Sprintf("PGSQL query error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
				errorMess := fmt.Sprintf("panic occurred: %+v", err)
				config.Log.Errorf(errorMess)
				ret = fmt.Errorf(errorHeader + errorMess)
			}
		}()

		errorHeader := fmt.Sprintf("PGSQL query error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbHost := c.Host
		dbName := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.SSLMode != "" {
			sslMode = c.SqlReplicationConfig.SSLMode
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		if c.SqlQueryConfig.Response != "" {
			if id != c.SqlQueryConfig.Response {
				err = fmt.Errorf("error: db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	Checks["pgsql_query_unixtime"] = func(c *config.Check, p *projects.Project) (ret error) {

		var (
			id      int64
			query   string
			dbPort  int
			sslMode = "disable"
		)

		defer func() {
			if err := recover(); err != nil {
				errorHeader := fmt.Sprintf("PGSQL query unixtime error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
				errorMess := fmt.Sprintf("panic occurred: %+v", err)
				config.Log.Errorf(errorMess)
				ret = fmt.Errorf(errorHeader + errorMess)
			}
		}()

		errorHeader := fmt.Sprintf("PGSQL query unixtime error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbHost := c.Host
		dbName := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.SSLMode != "" {
			sslMode = c.SqlReplicationConfig.SSLMode
		}

		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		if err != nil {
			config.Log.Printf("Cannot parse difference value: %v", dif)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		if dif > 0 {
			lastRecord := time.Unix(id, 0)
			curDif := time.Since(lastRecord)
			if curDif > dif {
				err := fmt.Errorf("unixtime difference error: got %v, difference %v", lastRecord, curDif)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	Checks["pgsql_query_timestamp"] = func(c *config.Check, p *projects.Project) (ret error) {

		var (
			timestamp time.Time
			query     string
			dbPort    int
			sslMode   = "disable"
		)

		defer func() {
			if err := recover(); err != nil {
				errorHeader := fmt.Sprintf("PGSQL query timestamp error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
				errorMess := fmt.Sprintf("panic occurred: %+v", err)
				config.Log.Errorf(errorMess)
				ret = fmt.Errorf(errorHeader + errorMess)
			}
		}()

		errorHeader := fmt.Sprintf("PGSQL query timestamp error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbHost := c.Host
		dbName := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.SSLMode != "" {
			sslMode = c.SqlReplicationConfig.SSLMode
		}

		if c.SqlQueryConfig.Difference == "" {
			err := fmt.Sprintf("Cannot parse difference value: '%v'", c.SqlQueryConfig.Difference)
			config.Log.Printf(err)
			return fmt.Errorf(err)
		}
		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		config.Log.Infof("Difference parsed %s", dif)
		if err != nil {
			config.Log.Printf("Cannot parse difference value: %v", dif)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		err = db.QueryRow(query).Scan(&timestamp)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		lastRecord := timestamp
		curDif := time.Since(lastRecord)
		config.Log.Infof("lastRecord %s", lastRecord)
		config.Log.Infof("curDif %s", curDif)
		if curDif > dif {
			err := fmt.Errorf("Timestamp difference error: got %v, difference %v", lastRecord, curDif)
			return fmt.Errorf(errorHeader + err.Error())
		}

		return nil
	}

	Checks["pgsql_replication"] = func(c *config.Check, p *projects.Project) (ret error) {

		var (
			dbPort, recordId, recordValue, id int
			dbTable                           = "repl_test"
			sslMode                           = "disable"
		)

		defer func() {
			if err := recover(); err != nil {
				errorHeader := fmt.Sprintf("PGSQL replication check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
				errorMess := fmt.Sprintf("panic occurred: %+v", err)
				config.Log.Errorf(errorMess)
				ret = fmt.Errorf(errorHeader + errorMess)
			}
		}()

		errorHeader := fmt.Sprintf("PGSQL replication check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		recordId = rand.Intn(5 - 1)
		recordValue = rand.Intn(9999 - 1)

		dbUser := c.SqlReplicationConfig.UserName
		dbPassword := c.SqlReplicationConfig.Password
		dbHost := c.Host
		dbName := c.SqlReplicationConfig.DBName
		if c.SqlReplicationConfig.SSLMode != "" {
			sslMode = c.SqlReplicationConfig.SSLMode
		}
		if c.SqlReplicationConfig.TableName != "repl_test" {
			dbTable = c.SqlReplicationConfig.TableName
		}

		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		config.Log.Debugf("Replication Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Errorf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		config.Log.Debugf("Set info on master, id: %d, value: %d", recordId, recordValue)
		insertSql := "INSERT INTO %s (id, test_value, timestamp) VALUES (%d, %d ,now()) ON CONFLICT (id) DO UPDATE set test_value=%d,timestamp=now() where %s.id=%d;"

		sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, dbTable, recordId)
		//config.Log.Printf("sqlStatement string: %s", sqlStatement)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			return fmt.Errorf(errorHeader+"pgsql insert error: %+v\n", err.Error())
		}

		// allow replication to pass
		lagAllowed, err := time.ParseDuration(c.SqlReplicationConfig.Lag)
		if err != nil {
			config.Log.Errorf("Error: Could not parse lag allowed: '%+v', use default 3s", err)
			lagAllowed = 3 * time.Second
		}
		time.Sleep(lagAllowed)

		for _, slave := range c.SqlReplicationConfig.ServerList {
			var (
				host, port string
			)

			selectSql := "SELECT test_value FROM %s where %s.id=%d;"
			sqlStatement := fmt.Sprintf(selectSql, dbTable, dbTable, recordId)

			config.Log.Debugf("Read from slave %s", slave)
			//config.Log.Printf(" query: %s\n", sqlStatement)

			// if slave defined as `host:port`
			host, port, err := net.SplitHostPort(slave)
			if err == nil {
				dbPort, err = strconv.Atoi(port)
				if err != nil {
					config.Log.Warnf("Cannot parse slave port %s", err)
				}
			} else {
				host = slave
			}

			if c.SqlQueryConfig.SSLMode != "" {
				sslMode = c.SqlReplicationConfig.SSLMode
			}

			slaveConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, host, dbPort, dbName, sslMode)

			if dbConnectTimeout > 0 {
				slaveConnStr = slaveConnStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
			}

			db, err := sql.Open("postgres", slaveConnStr)
			if err != nil {
				config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
				return fmt.Errorf(errorHeader + err.Error())
			}
			defer func() { _ = db.Close() }()

			err = db.Ping()
			if err != nil {
				config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
				return fmt.Errorf(errorHeader + err.Error())
			}

			err = db.QueryRow(sqlStatement).Scan(&id)
			if err != nil {
				config.Log.Printf("Error: Could not query database: %+v", err)
				return fmt.Errorf(errorHeader + err.Error())
			}

			if id != recordValue {
				err = fmt.Errorf("replication error: db response does not match expected: %d (expected %d) on server %s after %s", id, recordValue, host, lagAllowed)
				return fmt.Errorf(errorHeader + err.Error())
			}

		}

		return nil
	}

	Checks["pgsql_replication_status"] = func(c *config.Check, p *projects.Project) (ret error) {

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
			dbPort                                int
			dbTable                               = "pg_stat_replication"
			dbName                                = "postgres"
			sslMode                               = "disable"
			repStatusReply                        []repStatus
			hours, minutes, seconds, microseconds int
			streaming                             bool
		)

		defer func() {
			if err := recover(); err != nil {
				errorHeader := fmt.Sprintf("PGSQL replication status check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
				errorMess := fmt.Sprintf("panic occurred: %+v", err)
				config.Log.Errorf(errorMess)
				ret = fmt.Errorf(errorHeader + errorMess)
			}
		}()

		errorHeader := fmt.Sprintf("PGSQL replication check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlReplicationConfig.UserName
		dbPassword := c.SqlReplicationConfig.Password
		dbHost := c.Host
		if c.SqlReplicationConfig.SSLMode != "" {
			sslMode = c.SqlReplicationConfig.SSLMode
		}

		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		config.Log.Debugf("Replication Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + "sql.Open error\n" + err.Error())
		}
		defer func() { _ = db.Close() }()

		// unable to ping with default pg_monitor role
		//err = db.Ping()
		//if err != nil {
		//	config.Log.Errorf("Error: Could not establish a connection with the database: %+v", err)
		//	return fmt.Errorf(errorHeader + "db.Ping error\n" + err.Error())
		//}

		config.Log.Debugf("Getting replication status from master...")

		sqlQuery := fmt.Sprintf("select * from %s;", dbTable)

		rows, err := db.Query(sqlQuery)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return fmt.Errorf(errorHeader + "repstatus sql.Query error\n" + err.Error())
		}

		defer func() { _ = rows.Close() }()

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
				config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
				return fmt.Errorf(errorHeader + "repstatus rows.Scan error\n" + err.Error())
			}
			repStatusReply = append(repStatusReply, reply)

			rowsCount++
		}
		if rowsCount == 0 {
			config.Log.Printf("Error: no rows in query result")
			return fmt.Errorf("%s repstatus no rows in query result\n", errorHeader)
		}

		err = rows.Err()
		if err != nil {
			config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + "repstatus rows.Err error\n" + err.Error())
		}

		for i, reply := range repStatusReply {
			//config.Log.Infof("Rep status reply row #%d: %v", i, reply)
			if reply.state.String == "streaming" {
				streaming = true

				allowedLag, err := time.ParseDuration(c.SqlReplicationConfig.Lag)
				if err != nil {
					config.Log.Errorf("Error parsing allowed lag: %+v", err)
					return fmt.Errorf(errorHeader + "allowed lag parsing error\n" + err.Error())
				}

				if reply.replayLag.Valid {
					_, err = fmt.Sscanf(reply.replayLag.String, "%d:%d:%d.%d", &hours, &minutes, &seconds, &microseconds)
					if err != nil {
						err := fmt.Sprintf("Error scanning replay_lag: %+v\nreplay_lag: '%s'\n", err, reply.replayLag.String)
						config.Log.Error(err)
						config.Log.Errorf("Rep status reply row #%d\n", i)
						fields := reflect.ValueOf(reply)
						for i := 0; i < fields.NumField(); i++ {
							config.Log.Errorf("Rep status reply field: '%s'\tValue: '%s'\n", fields.Type().Field(i).Name, fields.Field(i))
						}
						return fmt.Errorf(errorHeader + err)
					}
				}

				lag, err := time.ParseDuration(fmt.Sprintf("%dh%dm%ds%dus", hours, minutes, seconds, microseconds))
				if err != nil {
					config.Log.Errorf("Error parsing replay_lag: %+v", err)
					return fmt.Errorf(errorHeader + "replay_lag parsing error\n" + err.Error())
				}

				if lag > allowedLag {
					err := fmt.Errorf("replay_lag is more than %s detected on %s: %s", allowedLag.String(), reply.applicationName.String, lag.String())
					config.Log.Infof(err.Error())
					return fmt.Errorf(errorHeader + err.Error())
				}
			}
		}
		if !streaming {
			config.Log.Errorf("Replication is not streaming\n")
			return fmt.Errorf(errorHeader + "Replication is not streaming\n")
		}
		return nil
	}
}
