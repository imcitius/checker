package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func runMysqlQueryCheck(c *Check, p *Project) error {

	var id, query string
	var dbport int

	dbuser := c.SqlQueryConfig.UserName
	dbpassword := c.SqlQueryConfig.Password
	dbhost := c.Host
	dbname := c.SqlQueryConfig.DBName
	if c.Port == 0 {
		dbport = 3306
	} else {
		dbport = c.Port
	}

	dbConnectTimeout := c.Timeout / 1000 // milliseconds to seconds

	if c.SqlQueryConfig.Query == "" {
		query = "select 1;"
	} else {
		query = c.SqlQueryConfig.Query
	}

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("?timeout=%ds", dbConnectTimeout)
	}

	//log.Printf("Connect string: %s", connStr)

	db, err := sql.Open("mysql", connStr)
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
