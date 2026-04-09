// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"testing"
	"time"

	"github.com/imcitius/checker/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func testLogger() *logrus.Entry {
	return logrus.WithField("test", true)
}

func TestCheckerFactory_HTTPCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-http",
		Type: "http",
		Config: &models.HTTPCheckConfig{
			URL:     "https://example.com",
			Timeout: "5s",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*HTTPCheck)
	assert.True(t, ok, "expected *HTTPCheck, got %T", c)
}

func TestCheckerFactory_TCPCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-tcp",
		Type: "tcp",
		Config: &models.TCPCheckConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	tc, ok := c.(*TCPCheck)
	assert.True(t, ok)
	assert.Equal(t, "10s", tc.Timeout, "default timeout should be set")
}

func TestCheckerFactory_SSHCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-ssh",
		Type: "ssh",
		Config: &models.SSHCheckConfig{
			Host: "192.168.1.1",
			Port: 0, // should default
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*SSHCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_ICMPCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-icmp",
		Type: "icmp",
		Config: &models.ICMPCheckConfig{
			Host: "8.8.8.8",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	ic, ok := c.(*ICMPCheck)
	assert.True(t, ok)
	assert.Equal(t, "10s", ic.Timeout)
}

func TestCheckerFactory_DNSCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-dns",
		Type: "dns",
		Config: &models.DNSCheckConfig{
			Domain:     "example.com",
			RecordType: "A",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*DNSCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_PassiveCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID:      "test-passive",
		Name:      "passive-test",
		Project:   "proj",
		GroupName: "grp",
		Type:      "passive",
		LastRun:   time.Now(),
		Config: &models.PassiveCheckConfig{
			Timeout: "60s",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*PassiveCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_MySQLQuery(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-mysql-query",
		Type: "mysql_query",
		Config: &models.MySQLCheckConfig{
			Host:     "localhost",
			Port:     3306,
			UserName: "root",
			DBName:   "test",
			Query:    "SELECT 1",
			Response: "1",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*MySQLCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_MySQLQueryUnixtime(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-mysql-unixtime",
		Type: "mysql_query_unixtime",
		Config: &models.MySQLCheckConfig{
			Host:     "localhost",
			Port:     3306,
			UserName: "root",
			DBName:   "test",
			Query:    "SELECT UNIX_TIMESTAMP()",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*MySQLTimeCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_MySQLReplication(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-mysql-replication",
		Type: "mysql_replication",
		Config: &models.MySQLCheckConfig{
			Host:     "localhost",
			Port:     3306,
			UserName: "root",
			DBName:   "test",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*MySQLReplicationCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_MySQLUnknownSubtype(t *testing.T) {
	def := models.CheckDefinition{
		UUID:   "test-mysql-bad",
		Type:   "mysql_unknown",
		Config: &models.MySQLCheckConfig{Host: "localhost"},
	}
	c := CheckerFactory(def, testLogger())
	assert.Nil(t, c)
}

func TestCheckerFactory_PgsqlQuery(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-pgsql-query",
		Type: "pgsql_query",
		Config: &models.PostgreSQLCheckConfig{
			Host:     "localhost",
			Port:     5432,
			UserName: "postgres",
			DBName:   "test",
			Query:    "SELECT 1",
			Response: "1",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*PostgreSQLCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_PgsqlQueryUnixtime(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-pgsql-unixtime",
		Type: "pgsql_query_unixtime",
		Config: &models.PostgreSQLCheckConfig{
			Host: "localhost",
			Port: 5432,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	tc, ok := c.(*PostgreSQLTimeCheck)
	assert.True(t, ok)
	assert.Equal(t, "unixtime", tc.TimeType)
}

func TestCheckerFactory_PgsqlQueryTimestamp(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-pgsql-timestamp",
		Type: "pgsql_query_timestamp",
		Config: &models.PostgreSQLCheckConfig{
			Host: "localhost",
			Port: 5432,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	tc, ok := c.(*PostgreSQLTimeCheck)
	assert.True(t, ok)
	assert.Equal(t, "timestamp", tc.TimeType)
}

func TestCheckerFactory_PgsqlReplication(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-pgsql-repl",
		Type: "pgsql_replication",
		Config: &models.PostgreSQLCheckConfig{
			Host: "localhost",
			Port: 5432,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	rc, ok := c.(*PostgreSQLReplicationCheck)
	assert.True(t, ok)
	assert.Equal(t, "replication", rc.CheckType)
}

func TestCheckerFactory_PgsqlReplicationStatus(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-pgsql-repl-status",
		Type: "pgsql_replication_status",
		Config: &models.PostgreSQLCheckConfig{
			Host: "localhost",
			Port: 5432,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	rc, ok := c.(*PostgreSQLReplicationCheck)
	assert.True(t, ok)
	assert.Equal(t, "replication_status", rc.CheckType)
}

func TestCheckerFactory_PgsqlUnknownSubtype(t *testing.T) {
	def := models.CheckDefinition{
		UUID:   "test-pgsql-bad",
		Type:   "pgsql_unknown",
		Config: &models.PostgreSQLCheckConfig{Host: "localhost"},
	}
	c := CheckerFactory(def, testLogger())
	assert.Nil(t, c)
}

func TestCheckerFactory_RedisCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-redis",
		Type: "redis",
		Config: &models.RedisCheckConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	rc, ok := c.(*RedisCheck)
	assert.True(t, ok)
	assert.Equal(t, "10s", rc.Timeout)
}

func TestCheckerFactory_MongoDBCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-mongodb",
		Type: "mongodb",
		Config: &models.MongoDBCheckConfig{
			URI: "mongodb://localhost:27017",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*MongoDBCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_DomainExpiryCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-domain-expiry",
		Type: "domain_expiry",
		Config: &models.DomainExpiryCheckConfig{
			Domain:            "example.com",
			ExpiryWarningDays: 30,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*DomainExpiryCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_SSLCertCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-ssl",
		Type: "ssl_cert",
		Config: &models.SSLCertCheckConfig{
			Host: "example.com",
			Port: 443,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*SSLCertCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_SMTPCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-smtp",
		Type: "smtp",
		Config: &models.SMTPCheckConfig{
			Host: "smtp.example.com",
			Port: 587,
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*SMTPCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_GRPCHealthCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-grpc",
		Type: "grpc_health",
		Config: &models.GRPCHealthCheckConfig{
			Host: "localhost:50051",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*GRPCHealthCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_WebSocketCheck(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-ws",
		Type: "websocket",
		Config: &models.WebSocketCheckConfig{
			URL: "ws://localhost:8080/ws",
		},
	}
	c := CheckerFactory(def, testLogger())
	assert.NotNil(t, c)
	_, ok := c.(*WebSocketCheck)
	assert.True(t, ok)
}

func TestCheckerFactory_NilConfig(t *testing.T) {
	def := models.CheckDefinition{
		UUID:   "test-nil-config",
		Type:   "http",
		Config: nil,
	}
	c := CheckerFactory(def, testLogger())
	assert.Nil(t, c)
}

func TestCheckerFactory_UnknownConfigType(t *testing.T) {
	def := models.CheckDefinition{
		UUID:   "test-unknown",
		Type:   "unknown_type",
		Config: &models.WebhookConfig{URL: "http://example.com"},
	}
	c := CheckerFactory(def, testLogger())
	assert.Nil(t, c)
}

func TestCheckerFactory_NilLogger(t *testing.T) {
	def := models.CheckDefinition{
		UUID: "test-nil-logger",
		Type: "tcp",
		Config: &models.TCPCheckConfig{
			Host: "127.0.0.1",
			Port: 80,
		},
	}
	c := CheckerFactory(def, nil)
	assert.NotNil(t, c)
}
