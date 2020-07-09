package check

import (
	"database/sql"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go"
	"my/checker/config"
	projects "my/checker/projects"
	"time"
)

func init() {
	Checks["clickhouse_query"] = func(c *config.Check, p *projects.Project) error {

		var (
			query, id string
			dbPort    int
		)
		//var items interface{}

		errorHeader := fmt.Sprintf("Clickhouse query at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbName := c.SqlQueryConfig.DBName
		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 9000
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Warnf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s", dbHost, dbPort, dbUser, dbPassword, dbName)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&read_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Clickhouse connect string: %s", connStr)

		db, err := sql.Open("clickhouse", connStr)
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}
		defer db.Close()

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		if c.SqlQueryConfig.Response != "" {
			if id != c.SqlQueryConfig.Response {
				err = fmt.Errorf("db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	Checks["clickhouse_query_unixtime"] = func(c *config.Check, p *projects.Project) error {

		var (
			query  string
			id     int64
			dbPort int
		)

		errorHeader := fmt.Sprintf("Clickhouse query unixtime at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbName := c.SqlQueryConfig.DBName
		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 9000
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Warnf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		if err != nil {
			config.Log.Warnf("cannot parse difference value: %v", dif)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s", dbHost, dbPort, dbUser, dbPassword, dbName)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&read_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Clickhouse connect string: %s", connStr)

		db, err := sql.Open("clickhouse", connStr)
		if err != nil {
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
				err := fmt.Errorf("unixtime difference error: got %v, difference %v", lastRecord, curDif)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}
}
