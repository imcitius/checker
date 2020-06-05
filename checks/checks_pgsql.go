package check

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"math/rand"
	"my/checker/config"
	"time"
)

func init() {
	config.Checks["pgsql_query"] = func(c *config.Check, p *config.Project) error {

		var (
			id, query string
			dbPort    int
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

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

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
				err = fmt.Errorf("Error: db response does not match expected: %s (expected %s)", id, c.SqlQueryConfig.Response)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	config.Checks["pgsql_query_unixtime"] = func(c *config.Check, p *config.Project) error {

		var (
			id     int64
			query  string
			dbPort int
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

		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

		if dbConnectTimeout > 0 {
			connStr = connStr + fmt.Sprintf("&connect_timeout=%d", int(dbConnectTimeout.Seconds()))
		}

		//config.Log.Printf("Connect string: %s", connStr)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

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
			curDif := time.Now().Sub(lastRecord)
			if curDif > dif {
				err := fmt.Errorf("Unixtime differenct error: got %v, difference %v\n", lastRecord, curDif)
				return fmt.Errorf(errorHeader + err.Error())
			}
		}

		return nil
	}

	config.Checks["pgsql_replication"] = func(c *config.Check, p *config.Project) error {

		var (
			dbPort, recordId, recordValue, id int
			dbTable                           string = "repl_test"
		)

		errorHeader := fmt.Sprintf("PGSQL replication check error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		recordId = rand.Intn(5 - 1)
		recordValue = rand.Intn(9999 - 1)

		dbUser := c.SqlReplicationConfig.UserName
		dbPassword := c.SqlReplicationConfig.Password
		dbHost := c.Host
		dbName := c.SqlReplicationConfig.DBName
		sslMode := c.SqlReplicationConfig.SSLMode
		if c.SqlReplicationConfig.TableName != "repl_test" {
			dbTable = c.SqlReplicationConfig.TableName
		}

		if c.Port == 0 {
			dbPort = 5432
		} else {
			dbPort = c.Port
		}

		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

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

		err = db.Ping()
		if err != nil {
			config.Log.Errorf("Error: Could not establish a connection with the database: %+v", err)
			return fmt.Errorf(errorHeader + err.Error())
		}

		insertSql := "INSERT INTO %s (id,test_value) VALUES (%d,%d) ON CONFLICT (id) DO UPDATE set test_value=%d where %s.id=%d;"

		sqlStatement := fmt.Sprintf(insertSql, dbTable, recordId, recordValue, recordValue, dbTable, recordId)
		//config.Log.Printf("sqlStatement string: %s", sqlStatement)
		_, err = db.Exec(sqlStatement)
		if err != nil {
			return fmt.Errorf(errorHeader+"pgsql insert error: %+v\n", err.Error())
		}

		// allow replication to pass
		time.Sleep(1 * time.Second)

		for _, server := range c.SqlReplicationConfig.ServerList {
			selectSql := "SELECT test_value FROM %s where %s.id=%d;"
			sqlStatement := fmt.Sprintf(selectSql, dbTable, dbTable, recordId)

			//config.Log.Printf("Read from %s", server)
			//config.Log.Printf(" query: %s\n", sqlStatement)
			connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbUser, dbPassword, server, dbPort, dbName)

			if dbConnectTimeout > 0 {
				connStr = connStr + fmt.Sprintf("&connect_timeout=%d", dbConnectTimeout)
			}

			db, err := sql.Open("postgres", connStr)
			if err != nil {
				config.Log.Printf("Error: The data source arguments are not valid: %+v", err)
				return fmt.Errorf(errorHeader + err.Error())
			}

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
					err = fmt.Errorf("Replication error: db response does not match expected: %d (expected %d) on server %s", id, recordValue, server)
					return fmt.Errorf(errorHeader + err.Error())
				}
			}

		}

		return nil
	}
}
