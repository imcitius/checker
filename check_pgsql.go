package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"math/rand"
	"time"
)

func runPgsqlQueryCheck(c *Check, p *Project) error {

	var (
		id, query string
		dbPort    int
	)

	dbUser := c.SqlQueryConfig.UserName
	dbPassword := c.SqlQueryConfig.Password
	dbHost := c.Host
	dbName := c.SqlQueryConfig.DBName
	if c.Port == 0 {
		dbPort = 5432
	} else {
		dbPort = c.Port
	}
	dbConnectTimeout, err := time.ParseDuration(c.Timeout)

	if c.SqlQueryConfig.Query == "" {
		query = "select 1;"
	} else {
		query = c.SqlQueryConfig.Query
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
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

func runPgsqlUnixtimeCheck(c *Check, p *Project) error {

	var (
		id     int64
		query  string
		dbPort int
	)

	dbUser := c.SqlQueryConfig.UserName
	dbPassword := c.SqlQueryConfig.Password
	dbHost := c.Host
	dbName := c.SqlQueryConfig.DBName
	if c.Port == 0 {
		dbPort = 5432
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

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	if dbConnectTimeout > 0 {
		connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
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
func runPgsqlReplicationCheck(c *Check, p *Project) error {

	var (
		dbPort, recordId, recordValue, id int
		dbTable                           string = "repl_test"
	)

	recordId = rand.Intn(5 - 1)
	recordValue = rand.Intn(9999 - 1)

	dbUser := c.SqlReplicationConfig.UserName
	dbPassword := c.SqlReplicationConfig.Password
	dbHost := c.Host
	dbName := c.SqlReplicationConfig.DBName
	if c.SqlReplicationConfig.TableName != "repl_test" {
		dbTable = c.SqlReplicationConfig.TableName
	}

	if c.Port == 0 {
		dbPort = 5432
	} else {
		dbPort = c.Port
	}

	dbConnectTimeout, err := time.ParseDuration(c.Timeout)

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
		return fmt.Errorf("pgsql insert error: %+v\n", err)
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
