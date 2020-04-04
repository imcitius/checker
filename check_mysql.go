package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math/rand"
	"time"
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

func runMysqlReplicationCheck(c *Check, p *Project) error {

	var dbPort, recordId, recordValue, id int

	recordId = rand.Intn(5 - 1)
	recordValue = rand.Intn(9999 - 1)

	dbUser := c.SqlReplicationConfig.UserName
	dbPassword := c.SqlReplicationConfig.Password
	dbHost := c.Host
	dbName := c.SqlReplicationConfig.DBName
	dbTable := c.SqlReplicationConfig.TableName
	if c.Port == 0 {
		dbPort = 5432
	} else {
		dbPort = c.Port
	}

	dbConnectTimeout := c.Timeout / 1000 // milliseconds to seconds

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
	}

	//log.Printf("Replication Connect string: %s", connStr)

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

	insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON CONFLICT (id) DO UPDATE set test_value=%d where %s.id=%d;"

	sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, dbTable, recordId)
	//log.Printf("sqlStatement string: %s", sqlStatement)
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	// allow replication to pass
	time.Sleep(1 * time.Second)

	for _, server := range c.SqlReplicationConfig.ServerList {
		selectSql := "SELECT test_value FROM %s where %s.id=%d;"
		sqlStatement := fmt.Sprintf(selectSql, dbTable, dbTable, recordId)

		//log.Printf("Read from %s", server)
		//log.Printf(" query: %s\n", sqlStatement)
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, server, dbPort, dbName)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
		}

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

		err = db.QueryRow(sqlStatement).Scan(&id)
		if err != nil {
			log.Printf("Error: Could not query database: %+v", err)
			return err
		}

		if c.SqlQueryConfig.Response != "" {
			if id != recordValue {
				err = errors.New(fmt.Sprintf("Replication error: db response does not match expected: %d (expected %d) on server %s", id, recordValue, server))
				return err
			}
		}

	}

	return nil
}
