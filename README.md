# Checker System

A distributed health checking system for monitoring various services, databases, and endpoints.

## Features

- **15+ check types** — HTTP, TCP, ICMP, SSH, DNS, Redis, MongoDB, SSL/TLS, SMTP, gRPC, WebSocket, Domain Expiry, Passive, and full MySQL/PostgreSQL suites
- **9 alert channels** — Slack, Discord, Telegram, Email, PagerDuty, OpsGenie, Microsoft Teams, ntfy, and Webhooks
- Rich App integrations for Slack, Discord, and Telegram with incident threading and interactive buttons
- Web-based monitoring dashboard
- Extensible architecture for adding new check types and alert channels

## Installation

### Requirements

- Go 1.24.0 or later
- PostgreSQL (for storing check configurations and results)
- Access to monitored services

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/checker-github.git
cd checker-github

# Build the binary
go build -o checker ./cmd/app

# Run the checker
./checker -config config.yaml
```

## Kubernetes Deployment

`checker-edge` is the lightweight agent that runs inside your network and reports back to the Ensafely SaaS. It is distributed as a Helm chart via GitHub Pages.

### Add the Helm repository

```bash
helm repo add ensafely https://imcitius.github.io/checker
helm repo update
```

### Install checker-edge

```bash
helm install checker-edge ensafely/checker-edge \
  --set apiKey=ck_YOUR_KEY \
  --set region=office-london
```

Replace `ck_YOUR_KEY` with your API key (create one at [app.ensafely.com → API Keys](https://app.ensafely.com)) and set `region` to a label that identifies this deployment (e.g. `us-east-k8s`, `office-london`).

### Available values

See [`charts/checker-edge/values.yaml`](charts/checker-edge/values.yaml) for the full list of configurable parameters, including resource limits, node selectors, tolerations, and pod annotations.

### Production: use an existing Secret

Avoid putting your API key in plain text on the command line. Create a Kubernetes Secret first, then reference it:

```bash
kubectl create secret generic checker-edge-secret \
  --from-literal=api-key=ck_YOUR_KEY
```

```bash
helm install checker-edge ensafely/checker-edge \
  --set existingSecret.name=checker-edge-secret \
  --set existingSecret.key=api-key \
  --set region=office-london
```

---

## Configuration

Configuration is provided via YAML files. A basic example:

```yaml
defaults:
  duration: 10s
  alerts_channel: telegram
  maintenance_duration: 15m

db:
  protocol: postgres
  host: localhost:5432
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

## Supported Check Types

### HTTP

HTTP checks verify that web endpoints are responding correctly.

```yaml
Google:
  type: http
  url: https://google.com
  timeout: 5s
  code: [200]  # Expected status codes
  answer: "Google"  # Expected content in response
  skip_check_ssl: false
  ssl_expiration_period: "720h"  # Warn if SSL cert expires within 30 days
  stop_follow_redirects: false
  auth:
    user: admin
    password: secret
  headers:
    - Authorization: "Bearer token"
```

### TCP

TCP checks verify connectivity to a specific port.

```yaml
Database:
  type: tcp
  host: db.example.com
  port: 5432
  timeout: 3s
```

### ICMP (Ping)

ICMP checks verify that a host responds to ping requests.

```yaml
ServerPing:
  type: icmp
  host: server.example.com
  count: 3
  timeout: 5s
```

### SSH

SSH checks verify that an SSH server is reachable and optionally validate its banner string.

```yaml
GitServer:
  type: ssh
  host: git.example.com
  port: 22
  timeout: 5s
  expect_banner: "OpenSSH"  # Optional: verify the SSH banner contains this string
```

### DNS

DNS checks verify that a domain resolves correctly for a given record type.

```yaml
DNSLookup:
  type: dns
  domain: example.com
  record_type: A  # A, AAAA, MX, TXT, NS, CNAME
  expected: "93.184.216.34"  # Optional: expected value in results
  host: 8.8.8.8  # Optional: custom DNS resolver
  timeout: 5s
```

### Redis

Redis checks verify connectivity to a Redis instance using a PING command.

```yaml
CacheServer:
  type: redis
  host: redis.example.com
  port: 6379
  password: secret  # Optional
  db: 0  # Optional: database number
  timeout: 5s
```

### MongoDB

MongoDB checks verify connectivity to a MongoDB instance.

```yaml
DocumentStore:
  type: mongodb
  uri: "mongodb://user:pass@mongo.example.com:27017/mydb"
  timeout: 5s
```

### Domain Expiry

Domain expiry checks monitor domain registration expiration via WHOIS lookups.

```yaml
DomainRenewal:
  type: domain_expiry
  domain: example.com
  expiry_warning_days: 30  # Warn when domain expires within this many days
  timeout: 10s
```

### SSL/TLS Certificate

SSL certificate checks monitor certificate expiration and optionally validate the certificate chain.

```yaml
CertCheck:
  type: ssl_cert
  host: example.com
  port: 443
  expiry_warning_days: 30  # Warn when cert expires within this many days
  validate_chain: true  # Optional: verify the full certificate chain
  timeout: 5s
```

### SMTP

SMTP checks verify that a mail server is accepting connections.

```yaml
MailServer:
  type: smtp
  host: mail.example.com
  port: 587
  starttls: true  # Optional: use STARTTLS
  username: alerts@example.com  # Optional
  password: secret  # Optional
  timeout: 5s
```

### gRPC Health

gRPC health checks use the standard gRPC health checking protocol to verify service availability.

```yaml
PaymentService:
  type: grpc_health
  host: "grpc.example.com:50051"  # host:port format
  use_tls: true  # Optional: connect with TLS
  timeout: 5s
```

### WebSocket

WebSocket checks verify that a WebSocket endpoint accepts connections and optionally send/receive messages.

```yaml
LiveFeed:
  type: websocket
  url: "wss://ws.example.com/feed"  # ws:// or wss://
  send_message: "ping"  # Optional: message to send after connecting
  expect_message: "pong"  # Optional: expected response content
  timeout: 5s
```

### Passive

Passive checks wait for external signals rather than actively testing. An alert fires if no signal is received within the timeout.

```yaml
CronJob:
  type: passive
  timeout: 10m  # Alert if no signal received within this timeframe
```

### MySQL

#### MySQL Query Check

Performs a query to verify database connectivity and operation.

```yaml
MySQL Basic Query:
  type: mysql_query
  host: db.example.com
  port: 3306
  timeout: 5s
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
  username: repluser
  password: replpassword
  dbname: test_db
  table_name: replication_test  # Table must exist on all servers
  lag: "5s"  # Maximum allowed replication lag
  server_list:
    - "replica1.example.com"
    - "replica2.example.com:3307"
```

### PostgreSQL

#### PostgreSQL Query Check

Performs a query to verify database connectivity and operation.

```yaml
PostgreSQL Basic Query:
  type: pgsql_query
  host: db.example.com
  port: 5432
  timeout: 5s
  username: dbuser
  password: dbpassword
  dbname: mydatabase
  sslmode: require  # Optional: disable, require, verify-ca, verify-full
  query: "SELECT 1;"
  response: "1"  # Optional expected response
```

#### PostgreSQL Time Check

Verifies that the database server's time is synchronized.

```yaml
PostgreSQL Time Check:
  type: pgsql_query_unixtime  # or pgsql_query_timestamp
  host: db.example.com
  port: 5432
  timeout: 5s
  username: dbuser
  password: dbpassword
  dbname: mydatabase
  query: "SELECT CAST(EXTRACT(EPOCH FROM NOW()) AS INTEGER);"  # for unixtime
  difference: "10s"  # Maximum allowed time difference
```

#### PostgreSQL Replication Check

Monitors PostgreSQL replication by inserting test data on the master and verifying it appears on replicas.

```yaml
PostgreSQL Replication:
  type: pgsql_replication
  host: master-db.example.com
  port: 5432
  timeout: 5s
  username: repluser
  password: replpassword
  dbname: test_db
  sslmode: require
  table_name: replication_test
  lag: "5s"
  server_list:
    - "replica1.example.com"
  analytic_replicas:  # Optional: replicas with higher lag tolerance
    - "analytics.example.com"
```

#### PostgreSQL Replication Status Check

Checks replication health by querying PostgreSQL's built-in replication status views instead of inserting test data.

```yaml
PostgreSQL Replication Status:
  type: pgsql_replication_status
  host: master-db.example.com
  port: 5432
  timeout: 5s
  username: repluser
  password: replpassword
  dbname: mydatabase
  sslmode: require
  lag: "30s"
  server_list:
    - "replica1.example.com"
```

## Alert Channels

Alert channels define how you are notified when checks fail or recover. Configure them in the `alerts` section of your config file.

### Slack

Webhook-based alerting or full Slack App integration with incident threading, interactive buttons, and silence commands.

```yaml
alerts:
  slack:
    type: slack  # or slack_webhook
    webhook_url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

> **App integration**: When configured as a Slack App (bot token + channel), provides threaded incident tracking, interactive action buttons, and `/silence` commands.

### Discord

Full Discord App integration with rich embeds, interactive buttons, and thread-based incident tracking.

```yaml
alerts:
  discord:
    type: discord
    bot_token: YOUR_BOT_TOKEN
    channel_id: "123456789012345678"
```

### Telegram

Full Telegram Bot integration with message threading, error snapshots, and inline keyboards.

```yaml
alerts:
  telegram:
    type: telegram
    bot_token: YOUR_BOT_TOKEN
    critical_channel: CHANNEL_ID
    noncritical_channel: CHANNEL_ID
```

### Email

SMTP-based alerting with HTML templates.

```yaml
alerts:
  email:
    type: email
    smtp_host: smtp.example.com
    smtp_port: 587
    username: alerts@example.com
    password: secret
    from: alerts@example.com
    to:
      - team@example.com
```

### PagerDuty

Events API v2 integration with automatic resolve and severity mapping.

```yaml
alerts:
  pagerduty:
    type: pagerduty
    routing_key: YOUR_EVENTS_API_V2_ROUTING_KEY
```

### OpsGenie

Alert trigger and resolve with priority mapping (P1–P3).

```yaml
alerts:
  opsgenie:
    type: opsgenie
    api_key: YOUR_API_KEY
```

### Microsoft Teams

Webhook-based alerting using the MessageCard format.

```yaml
alerts:
  teams:
    type: teams
    webhook_url: https://outlook.office.com/webhook/YOUR/WEBHOOK/URL
```

### ntfy

Push notification service with priority mapping and action buttons.

```yaml
alerts:
  ntfy:
    type: ntfy
    topic: checker-alerts
    server: https://ntfy.sh  # Optional, defaults to https://ntfy.sh
    token: YOUR_ACCESS_TOKEN  # Optional
```

### Webhooks

Generic HTTP POST notifications with Go template body and HMAC-SHA256 signing for payload verification.

```yaml
alerts:
  custom_webhook:
    type: webhook
    url: https://api.example.com/alerts
    method: POST
    headers:
      Content-Type: application/json
    payload: '{"check": "{{.CheckName}}", "status": "{{.Status}}"}'
```

## Development

### Adding New Check Types

1. Define the check type in the `pkg/checks` package
2. Add a config struct in `pkg/models/check_types.go`
3. Register the check in `pkg/checks/factory.go`
4. Add UI components in the frontend
5. Create tests in `pkg/checks/your_check_test.go`

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests (requires services to be available)
INTEGRATION_TESTS=true go test ./...

# Run MySQL integration tests
INTEGRATION_TESTS=true TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD=password go test ./pkg/checks -run=^TestMySQL

# Note: When running individual test files, use the package approach instead of the file approach
# Correct:   go test ./pkg/checks -run=^TestMySQL
# Incorrect: go test ./pkg/checks/mysql_test.go
```

## License

This project is licensed under the [Business Source License 1.1](LICENSE) (BSL 1.1).

### What is allowed

- Self-hosting for internal use
- Modifying the source code
- Using the software for personal or internal business purposes
- Non-competing commercial use (e.g. running checks for your own infrastructure)

### What is NOT allowed

- Offering this software as a **managed monitoring, health-checking, or uptime-tracking service** to third parties (i.e. you cannot build a SaaS product on top of this software that competes with Ensafely)

### Change Date

On **May 1, 2031**, the license automatically converts to the **Apache License 2.0**, making the software fully open source.

For alternative licensing arrangements, please contact [Ensafely](https://ensafely.com).
