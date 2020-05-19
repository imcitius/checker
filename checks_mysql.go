package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"time"
)

func init() {
	Checks["mysql_query"] = func(c *Check, p *Project) error {

		var (
			id, query string
			dbport    int
		)

		dbuser := c.SqlQueryConfig.UserName
		dbpassword := c.SqlQueryConfig.Password
		dbhost := c.Host
		dbname := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbport = 3306
		} else {
			dbport = c.Port
		}

		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%ds", int(dbConnectTimeout.Seconds()))
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

	Checks["mysql_query_unixtime"] = func(c *Check, p *Project) error {

		var (
			id     int64
			query  string
			dbport int
		)

		dbuser := c.SqlQueryConfig.UserName
		dbpassword := c.SqlQueryConfig.Password
		dbhost := c.Host
		dbname := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbport = 3306
		} else {
			dbport = c.Port
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

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%ds", int(dbConnectTimeout.Seconds()))
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

	Checks["mysql_replication"] = func(c *Check, p *Project) error {

		var dbPort, recordId, recordValue, id int

		recordId = rand.Intn(5-1) + 1
		recordValue = rand.Intn(9999-1) + 1

		dbUser := c.SqlReplicationConfig.UserName
		dbPassword := c.SqlReplicationConfig.Password
		dbHost := c.Host
		dbName := c.SqlReplicationConfig.DBName
		dbTable := c.SqlReplicationConfig.TableName
		if c.Port == 0 {
			dbPort = 3306
		} else {
			dbPort = c.Port
		}

		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%ds", dbConnectTimeout)
		}

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%ds", dbConnectTimeout)
		}

		//log.Printf("Replication Connect string: %s", connStr)

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

		insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON DUPLICATE KEY UPDATE test_value=%d, id=%d;"

		sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, recordId)
		//log.Printf( "Insert statement string: %s", sqlStatement)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			return fmt.Errorf("Mysql insert error: %+v\n", err)
		}

		// allow replication to pass
		time.Sleep(1 * time.Second)

		for _, server := range c.SqlReplicationConfig.ServerList {
			selectSql := "SELECT test_value FROM %s where id=%d;"
			sqlStatement := fmt.Sprintf(selectSql, dbTable, recordId)

			//log.Printf("Read from %s", server)
			//log.Printf(" query: %s\n", sqlStatement)
			connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, server, dbPort, dbName)

			if dbConnectTimeout > 0 {
				connStr = connStr + fmt.Sprintf("?timeout=%ds", dbConnectTimeout)
			}
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

			err = db.QueryRow(sqlStatement).Scan(&id)
			if err != nil {
				log.Printf("Error: Could not query database: %+v (server %s)", err, server)
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
}
