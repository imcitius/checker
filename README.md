# Checker System

A distributed health checking system for monitoring various services, databases, and endpoints.

## Features

- HTTP endpoint monitoring with SSL and redirects support
- TCP port connectivity checks
- ICMP (ping) availability checks
- Passive monitoring capabilities
- PostgreSQL database monitoring (query, time synchronization, replication)
- MySQL database monitoring (query, time synchronization, replication)
- Extensible architecture for adding new check types
- Web-based monitoring dashboard
- Alerting capabilities via various channels

## Installation

### Requirements

- Go 1.23.5 or later
- MongoDB (for storing check configurations and results)
- Access to monitored services

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/checker-github.git
cd checker-github

# Build the binary
go build -o checker ./cmd/checker

# Run the checker
./checker -config config.yaml
```

## Configuration

Configuration is provided via YAML files. A basic example:

```yaml
defaults:
  duration: 10s
  alerts_channel: telegram
  maintenance_duration: 15m

db:
  protocol: mongodb
  host: localhost
  username: checker-dev
  database: checker_dev
  password: password

alerts:
  telegram:
    type: telegram
    bot_token: YOUR_BOT_TOKEN
    critical_channel: CHANNEL_ID
    noncritical_channel: CHANNEL_ID
  slack:
    type: slack
    webhook_url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
    channel: "#general"

projects:
  example:
    parameters:
      duration: 30s
    healthchecks:
      api_checks:
        parameters:
          duration: 30s
        checks:
          Google:
            type: http
            url: https://google.com
            timeout: 5s
```

## Check Types

### HTTP Checks

HTTP checks verify that web endpoints are responding correctly.

```yaml
Google:
  type: http
  url: https://google.com
  timeout: 5s
  code: [200]  # Expected status codes
  answer: "Google"  # Expected content in response
```

### TCP Checks

TCP checks verify connectivity to a specific port.

```yaml
Database:
  type: tcp
  host: db.example.com
  port: 5432
  timeout: 3s
```

### ICMP Checks

ICMP checks verify that a host responds to ping requests.

```yaml
ServerPing:
  type: icmp
  host: server.example.com
  count: 3
  timeout: 5s
```

### Passive Checks

Passive checks wait for external signals rather than actively testing.

```yaml
CronJob:
  type: passive
  timeout: 10m  # Alert if no signal received within this timeframe
```

### MySQL Checks

#### MySQL Query Check

Performs a simple query to verify database connectivity and operation.

```yaml
MySQL Basic Query:
  type: mysql_query
  host: db.example.com
  port: 3306
  timeout: 5s
  mysql:
    username: dbuser
    password: dbpassword
    dbname: mydatabase
    query: "SELECT 1;"
    response: "1"  # Optional expected response
```

#### MySQL Time Check

Verifies that the database server's time is synchronized within a specified tolerance.

```yaml
MySQL Time Check:
  type: mysql_query_unixtime
  host: db.example.com
  port: 3306
  timeout: 5s
  mysql:
    username: dbuser
    password: dbpassword
    dbname: mydatabase
    query: "SELECT UNIX_TIMESTAMP();"
    difference: "10s"  # Maximum allowed time difference
```

#### MySQL Replication Check

Monitors MySQL replication by inserting test data on the master and verifying it appears on replicas.

```yaml
MySQL Replication:
  type: mysql_replication
  host: master-db.example.com
  port: 3306
  timeout: 5s
  mysql:
    username: repluser
    password: replpassword
    dbname: test_db
    table_name: replication_test  # Table must exist on all servers
    lag: "5s"  # Maximum allowed replication lag
      - "replica1.example.com"
      - "replica2.example.com:3307"

### PostgreSQL Checks

#### PostgreSQL Query Check

Performs a simple query to verify database connectivity and operation.

```yaml
PostgreSQL Basic Query:
  type: pgsql_query
  host: db.example.com
  port: 5432
  timeout: 5s
  mysql: # Using common config structure, key is usually ignored or reused
    username: dbuser
    password: dbpassword
    dbname: mydatabase
    query: "SELECT 1;"
    response: "1"  # Optional expected response
```

#### PostgreSQL Time Check

Verifies that the database server's time is synchronized.

```yaml
PostgreSQL Time Check:
  type: pgsql_unixtime # or pgsql_timestamp
  host: db.example.com
  port: 5432
  timeout: 5s
  mysql:
    username: dbuser
    password: dbpassword
    dbname: mydatabase
    query: "SELECT CAST(EXTRACT(EPOCH FROM NOW()) AS INTEGER);" # for unixtime
    difference: "10s"  # Maximum allowed time difference
```

#### PostgreSQL Replication Check

Monitors PostgreSQL replication.

```yaml
PostgreSQL Replication:
  type: pgsql_replication
  host: master-db.example.com
  port: 5432
  timeout: 5s
  mysql:
    username: repluser
    password: replpassword
    dbname: test_db
    table_name: replication_test
    lag: "5s"
    server_list: 
      - "replica1.example.com"
```
```

## Development

### Adding New Check Types

1. Define the check type in the `internal/checks` package
2. Update the `CheckerFactory` in `internal/scheduler/factories.go`
3. Add UI components in `internal/web/templates/check_management.html`
4. Create tests in `internal/checks/your_check_test.go`

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests (requires services to be available)
INTEGRATION_TESTS=true go test ./...

# Run MySQL integration tests
INTEGRATION_TESTS=true TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD=password go test ./internal/checks -run=^TestMySQL

# Note: When running individual test files, use the package approach instead of the file approach
# Correct:   go test ./internal/checks -run=^TestMySQL
# Incorrect: go test ./internal/checks/mysql_test.go
```

## License

This project is licensed under the MIT License - see the LICENSE file for details. 