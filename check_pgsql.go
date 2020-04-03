package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

func runPgsqlQueryCheck(c *Check, p *Project) error {

	var id, query string
	var dbPort int

	dbUser := c.SqlQueryConfig.UserName
	dbPassword := c.SqlQueryConfig.Password
	dbHost := c.Host
	dbName := c.SqlQueryConfig.DBName
	if c.Port == 0 {
		dbPort = 5432
	} else {
		dbPort = c.Port
	}

	dbConnectTimeout := c.Timeout / 1000 // milliseconds to seconds

	if c.SqlQueryConfig.Query == "" {
		query = "select 1;"
	} else {
		query = c.SqlQueryConfig.Query
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
	}

	//log.Printf("Connect string: %s", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error: The data source arguments are not valid: %+v", err)
		return err
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
