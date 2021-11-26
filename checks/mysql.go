package check

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"my/checker/config"
	projects "my/checker/projects"
	"net"
	"strconv"
	"time"
)

func init() {
	Checks["mysql_query"] = func(c *config.Check, p *projects.Project) error {

		var (
			id, query string
			dbport    int
		)

		errorHeader := fmt.Sprintf("MYSQL query error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbuser := c.SqlQueryConfig.UserName
		dbpassword := c.SqlQueryConfig.Password
		dbhost := c.Host
		dbname := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbport = 3306
		} else {
			dbport = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("mysql", connStr)
		if err != nil {
			config.Log.Printf(errorHeader+"Error: The data source arguments are not valid: %+v", err)
			return err
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf(errorHeader+"Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf(errorHeader+"Error: Could not query database: %+v", err)
			return err
		}

		if c.SqlQueryConfig.Response != "" {
			if id != c.SqlQueryConfig.Response {
				err = fmt.Errorf(errorHeader+"Error: db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response)
				return err
			}
		}

		return nil
	}

	Checks["mysql_query_unixtime"] = func(c *config.Check, p *projects.Project) error {

		var (
			id     int64
			query  string
			dbport int
		)

		errorHeader := fmt.Sprintf("PGSQL query unixtime error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbuser := c.SqlQueryConfig.UserName
		dbpassword := c.SqlQueryConfig.Password
		dbhost := c.Host
		dbname := c.SqlQueryConfig.DBName
		if c.Port == 0 {
			dbport = 3306
		} else {
			dbport = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		dif, err := time.ParseDuration(c.SqlQueryConfig.Difference)
		if err != nil {
			config.Log.Printf(errorHeader+"Cannot parse difference value: %v", dif)
		}

		if c.SqlQueryConfig.Query == "" {
			query = "select 1;"
		} else {
			query = c.SqlQueryConfig.Query
		}

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbuser, dbpassword, dbhost, dbport, dbname)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("mysql", connStr)
		if err != nil {
			config.Log.Printf(errorHeader+"Error: The data source arguments are not valid: %+v", err)
			return err
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf(errorHeader+"Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		err = db.QueryRow(query).Scan(&id)
		if err != nil {
			config.Log.Printf(errorHeader+"Error: Could not query database: %+v", err)
			return err
		}

		if dif > 0 {
			lastRecord := time.Unix(id, 0)
			curDif := time.Since(lastRecord)
			if curDif > dif {
				err := fmt.Errorf(errorHeader+"Unixtime difference error: got %v, difference %v\n", lastRecord, curDif)
				return err
			}
		}

		return nil
	}

	Checks["mysql_replication"] = func(c *config.Check, p *projects.Project) error {

		var dbPort, recordId, recordValue, id int

		errorHeader := fmt.Sprintf("MYSQL replication error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

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

		if c.Timeout == "" {
			c.Timeout = config.DefaultTCPConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Errorf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
		}

		//config.Log.Printf("Replication Connect string: %s", connStr)

		db, err := sql.Open("mysql", connStr)
		if err != nil {
			config.Log.Printf(errorHeader+"Error: The data source arguments are not valid: %+v", err)
			return err
		}
		defer func() { _ = db.Close() }()

		err = db.Ping()
		if err != nil {
			config.Log.Printf(errorHeader+"Error: Could not establish a connection with the database: %+v", err)
			return err
		}

		insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON DUPLICATE KEY UPDATE test_value=%d, id=%d;"

		sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, recordId)
		//config.Log.Printf( "Insert statement string: %s", sqlStatement)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			return fmt.Errorf(errorHeader+"Mysql insert error: %+v\n", err)
		}

		// allow replication to pass
		time.Sleep(1 * time.Second)

		for _, server := range c.SqlReplicationConfig.ServerList {
			selectSql := "SELECT test_value FROM %s where id=%d;"
			sqlStatement := fmt.Sprintf(selectSql, dbTable, recordId)

			//config.Log.Printf("Read from %s", server)
			//config.Log.Printf(" query: %s\n", sqlStatement)

			// if slave defined as `host:port`
			host, port, err := net.SplitHostPort(server)
			if err == nil {
				server = host
				dbPort, err = strconv.Atoi(port)
				if err != nil {
					config.Log.Warnf("Cannot parse slave port %s", err)
				}
			}

			connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbUser, dbPassword, server, dbPort, dbName)

			if dbConnectTimeout > 0 {
				connStr = connStr + fmt.Sprintf("?timeout=%.0fs", dbConnectTimeout.Seconds())
			}
			db, err := sql.Open("mysql", connStr)
			if err != nil {
				config.Log.Printf(errorHeader+"Error: The data source arguments are not valid: %+v", err)
				return err
			}
			defer func() { _ = db.Close() }()

			err = db.Ping()
			if err != nil {
				config.Log.Printf(errorHeader+"Error: Could not establish a connection with the database: %+v", err)
				return err
			}

			err = db.QueryRow(sqlStatement).Scan(&id)
			if err != nil {
				config.Log.Printf(errorHeader+"Error: Could not query database: %+v (server %s)", err, server)
				return err
			}

			if c.SqlQueryConfig.Response != "" {
				if id != recordValue {
					err = fmt.Errorf("replication error: db response does not match expected: %d (expected %d) on server %s", id, recordValue, host)
					return fmt.Errorf(errorHeader + err.Error())
				}
			} else {
				err = fmt.Errorf("replication error: slave db reply is empty: on server %s", host)
				return fmt.Errorf(errorHeader + err.Error())
			}

		}

		return nil
	}
}
