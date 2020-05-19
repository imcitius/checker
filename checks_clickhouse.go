package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go"
	"time"
)

func init() {
	Checks["clickhouse_query"] = func(c *Check, p *Project) error {

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

		//log.Printf("Clickhouse connect string: %s", connStr)

		db, err := sql.Open("clickhouse", connStr)
		if err != nil {
			log.Fatal(err)
		}

		err = db.Ping()
		if err != nil {
			log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			log.Printf("Error: Could not query database: %+v", err)
			return err
		}

		if c.SqlQueryConfig.Response != "" {
			if id != c.SqlQueryConfig.Response {
				err = errors.New(fmt.Sprintf("Error: db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response))
				return err
			}
		}

		return nil
	}

	Checks["clickhouse_query_unixtime"] = func(c *Check, p *Project) error {

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
			log.Printf("Cannot parse differenct value: %v", dif)
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

		//log.Printf("Clickhouse connect string: %s", connStr)

		db, err := sql.Open("clickhouse", connStr)
		if err != nil {
			log.Fatal(err)
		}

		err = db.Ping()
		if err != nil {
			log.Printf("Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			log.Printf("Error: Could not query database: %+v", err)
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
