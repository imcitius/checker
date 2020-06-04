package check

import (
	"database/sql"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go"
	"my/checker/config"
	"time"
)

func init() {
	config.Checks["clickhouse_query"] = func(c *config.Check, p *config.Project) error {

		var (
			query, id string
			dbPort    int
		)
		//var items interface{}

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbName := c.SqlQueryConfig.DBName
		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 9000
		} else {
			dbPort = c.Port
		}

		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

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
			config.Log.Fatal(err)
		}

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return err
		}

		if c.SqlQueryConfig.Response != "" {
			if id != c.SqlQueryConfig.Response {
				err = fmt.Errorf("Error: db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response)
				return err
			}
		}

		return nil
	}

	config.Checks["clickhouse_query_unixtime"] = func(c *config.Check, p *config.Project) error {

		var (
			query  string
			id     int64
			dbPort int
		)

		dbUser := c.SqlQueryConfig.UserName
		dbPassword := c.SqlQueryConfig.Password
		dbName := c.SqlQueryConfig.DBName
		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 9000
		} else {
			dbPort = c.Port
		}

		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		if err != nil {
			config.Log.Printf("Cannot parse differenct value: %v", dif)
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
			config.Log.Fatal(err)
		}

		err = db.Ping()
		if err != nil {
			config.Log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf("Error: Could not query database: %+v", err)
			return err
		}

		if dif > 0 {
			lastRecord := time.Unix(id, 0)
			curDif := time.Now().Sub(lastRecord)
			if curDif > dif {
				err := fmt.Errorf("Unixtime differenct error: got %v, difference %v\n", lastRecord, curDif)
				return err
			}
		}

		return nil
	}
}
