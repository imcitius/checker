package checks

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMySQLCheck_Run(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	// Get MySQL credentials from environment or use defaults
	username := getEnvOrDefault("TEST_MYSQL_USERNAME", "root")
	password := getEnvOrDefault("TEST_MYSQL_PASSWORD", "password")
	dbname := getEnvOrDefault("TEST_MYSQL_DATABASE", "mysql")
	host := getEnvOrDefault("TEST_MYSQL_HOST", "localhost")
	portStr := getEnvOrDefault("TEST_MYSQL_PORT", "3306")

	tests := []struct {
		name          string
		check         MySQLCheck
		wantErr       bool
		expectedError string
	}{
		{
			name: "Valid basic query",
			check: MySQLCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query: "SELECT 1",
				},
				Logger: logEntry,
			},
			wantErr: false,
		},
		{
			name: "Valid query with expected response",
			check: MySQLCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:    "SELECT 1",
					Response: "1",
				},
				Logger: logEntry,
			},
			wantErr: false,
		},
		{
			name: "Invalid response expectation",
			check: MySQLCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:    "SELECT 1",
					Response: "2", // Will fail as query returns 1
				},
				Logger: logEntry,
			},
			wantErr:       true,
			expectedError: "db response does not match expected",
		},
		{
			name: "Invalid query",
			check: MySQLCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query: "SELECT invalid_column FROM nonexistent_table",
				},
				Logger: logEntry,
			},
			wantErr: true,
		},
		{
			name: "Connection failure - wrong host",
			check: MySQLCheck{
				Host:    "nonexistent-host",
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "2s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query: "SELECT 1",
				},
				Logger: logEntry,
			},
			wantErr: true,
		},
		{
			name: "Invalid credentials",
			check: MySQLCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: "invalid_user",
						Password: "invalid_password",
						DBName:   dbname,
					},
					Query: "SELECT 1",
				},
				Logger: logEntry,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := tt.check.Run()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, duration >= 0)
			}
		})
	}
}

func TestMySQLTimeCheck_Run(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	// Get MySQL credentials from environment or use defaults
	username := getEnvOrDefault("TEST_MYSQL_USERNAME", "root")
	password := getEnvOrDefault("TEST_MYSQL_PASSWORD", "password")
	dbname := getEnvOrDefault("TEST_MYSQL_DATABASE", "mysql")
	host := getEnvOrDefault("TEST_MYSQL_HOST", "localhost")
	portStr := getEnvOrDefault("TEST_MYSQL_PORT", "3306")

	tests := []struct {
		name          string
		check         MySQLTimeCheck
		wantErr       bool
		expectedError string
	}{
		{
			name: "Valid time check - within difference",
			check: MySQLTimeCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLTimeQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:      "SELECT UNIX_TIMESTAMP()",
					Difference: "1h", // Should always pass unless server time is way off
				},
				Logger: logEntry,
			},
			wantErr: false,
		},
		{
			name: "Valid time check - very small difference",
			check: MySQLTimeCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLTimeQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:      "SELECT UNIX_TIMESTAMP()",
					Difference: "10s", // Should pass as server time should be close
				},
				Logger: logEntry,
			},
			wantErr: false,
		},
		{
			name: "Invalid query",
			check: MySQLTimeCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLTimeQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:      "SELECT invalid_column FROM nonexistent_table",
					Difference: "1h",
				},
				Logger: logEntry,
			},
			wantErr: true,
		},
		{
			name: "Invalid time difference",
			check: MySQLTimeCheck{
				Host:    host,
				Port:    getPortOrDefault(portStr, 3306),
				Timeout: "5s",
				Config: MySQLTimeQueryConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					Query:      "SELECT UNIX_TIMESTAMP() - 3600", // 1 hour in the past
					Difference: "30s",                            // 30 seconds tolerance
				},
				Logger: logEntry,
			},
			wantErr:       true,
			expectedError: "Unixtime difference error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := tt.check.Run()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, duration >= 0)
			}
		})
	}
}

// TestMySQLReplicationCheck_Run requires a real MySQL replication setup
// This is a more complex test and would typically be run in a CI environment with proper setup
func TestMySQLReplicationCheck_Run(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" || os.Getenv("TEST_MYSQL_REPLICATION") != "true" {
		t.Skip("Skipping MySQL replication test - requires real replication setup")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	// Get MySQL replication setup from environment
	username := getEnvOrDefault("TEST_MYSQL_USERNAME", "root")
	password := getEnvOrDefault("TEST_MYSQL_PASSWORD", "password")
	dbname := getEnvOrDefault("TEST_MYSQL_DATABASE", "test")
	masterHost := getEnvOrDefault("TEST_MYSQL_MASTER_HOST", "localhost")
	masterPortStr := getEnvOrDefault("TEST_MYSQL_MASTER_PORT", "3306")
	replicaHost := getEnvOrDefault("TEST_MYSQL_REPLICA_HOST", "localhost")
	replicaPortStr := getEnvOrDefault("TEST_MYSQL_REPLICA_PORT", "3307")
	tableName := getEnvOrDefault("TEST_MYSQL_REPL_TABLE", "replication_test")

	// Skip if all defaults are used
	if masterHost == "localhost" && replicaHost == "localhost" && masterPortStr == "3306" && replicaPortStr == "3307" {
		t.Skip("Skipping replication test - no custom replication configuration provided")
	}

	tests := []struct {
		name          string
		check         MySQLReplicationCheck
		wantErr       bool
		expectedError string
	}{
		{
			name: "Valid replication check",
			check: MySQLReplicationCheck{
				Host:    masterHost,
				Port:    getPortOrDefault(masterPortStr, 3306),
				Timeout: "5s",
				Config: MySQLReplicationConfig{
					MySQLConfig: MySQLConfig{
						UserName: username,
						Password: password,
						DBName:   dbname,
					},
					TableName:  tableName,
					Lag:        "5s",
					ServerList: []string{replicaHost + ":" + replicaPortStr},
				},
				Logger: logEntry,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := tt.check.Run()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, duration >= 0)
			}
		})
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getPortOrDefault(portStr string, defaultPort int) int {
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		// Try to parse as integer
		fmt.Sscanf(portStr, "%d", &port)
		if port <= 0 || port > 65535 {
			return defaultPort
		}
		return port
	}
	return port
}
