package check

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"math/rand"
	"my/checker/config"
	projects "my/checker/projects"
	"net"
	"strconv"
	"time"
)

func init() {
	Checks["pgsql_query"] = func(c *config.Check, p *projects.Project) error {

		var (
			id, query string
			dbPort    int
			sslMode   string
		)

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
			c.Timeout = config.DefaultConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.SSLMode == "" {
			sslMode = "disable"
		} else {
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
		defer db.Close()

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

	Checks["pgsql_query_unixtime"] = func(c *config.Check, p *projects.Project) error {

		var (
			id      int64
			query   string
			dbPort  int
			sslMode string
		)

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
			c.Timeout = config.DefaultConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.SSLMode == "" {
			sslMode = "disable"
		} else {
			sslMode = c.SqlReplicationConfig.SSLMode
		}

		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		if err != nil {
			config.Log.Printf("Cannot parse differenct value: %v", dif)
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
		defer db.Close()

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
				err := fmt.Errorf("unixtime differenct error: got %v, difference %v", lastRecord, curDif)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	Checks["pgsql_replication"] = func(c *config.Check, p *projects.Project) error {

		var (
			dbPort, recordId, recordValue, id int
			dbTable                           string = "repl_test"
			sslMode                           string
		)

		errorHeader := fmt.Sprintf("PGSQL replication check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		recordId = rand.Intn(5 - 1)
		recordValue = rand.Intn(9999 - 1)

		dbUser := c.SqlReplicationConfig.UserName
		dbPassword := c.SqlReplicationConfig.Password
		dbHost := c.Host
		dbName := c.SqlReplicationConfig.DBName
		if c.SqlReplicationConfig.SSLMode == "" {
			sslMode = "disable"
		} else {
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
			c.Timeout = config.DefaultConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
		}

		config.Log.Debugf("Replication Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Errorf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer db.Close()

		err = db.Ping()
		if err != nil {
			config.Log.Errorf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		config.Log.Debugf("Set info on master, id: %d, value: %d", recordId, recordValue)
		insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON CONFLICT (id) DO UPDATE set test_value=%d where %s.id=%d;"

		sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, dbTable, recordId)
		//config.Log.Printf("sqlStatement string: %s", sqlStatement)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			return fmt.Errorf(errorHeader+"pgsql insert error: %+v\n", err.Error())
		}

		// allow replication to pass
		time.Sleep(1 * time.Second)

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

			if c.SqlQueryConfig.SSLMode == "" {
				sslMode = "disable"
			} else {
				sslMode = c.SqlReplicationConfig.SSLMode
			}

			slaveConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, host, dbPort, dbName, sslMode)

			if dbConnectTimeout > 0 {
				slaveConnStr = slaveConnStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
			}

			db, err := sql.Open("postgres", slaveConnStr)
			if err != nil {
				config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
				return fmt.Errorf(errorHeader + err.Error())
			}
			defer db.Close()

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

			if c.SqlQueryConfig.Response != "" {
				if id != recordValue {
					err = fmt.Errorf("replication error: db response does not match expected: %d (expected %d) on server %s", id, recordValue, host)
					return fmt.Errorf(errorHeader + err.Error())
				}
			}

		}

		return nil
	}
}
