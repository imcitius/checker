# Generate Checker Config

You are helping the user generate a `config.yaml` for **Checker** — an open-source distributed health-checking and alerting system. Use everything you know about the user's infrastructure from this conversation to produce a complete, ready-to-use configuration.

## How to use this prompt

Paste this file into a Claude conversation where you've already discussed your infrastructure, or describe your setup and ask Claude to generate the config.

## What you should ask the user (if not already known)

1. **What services/endpoints do you want to monitor?** (URLs, hosts, ports, databases)
2. **Where should alerts go?** (Telegram, Slack, Discord, PagerDuty, OpsGenie, Teams, Email, ntfy)
3. **Database preference?** PostgreSQL (production) or SQLite (dev/demo)
4. **Authentication?** OIDC, password, API keys, or none
5. **Multi-region?** If yes, which regions and consensus settings

Then generate a complete `config.yaml` following the reference below.

---

## Config Structure

```yaml
# ──────────────────────────────────────────────
# Server
# ──────────────────────────────────────────────
server_port: "8080"

# ──────────────────────────────────────────────
# Database
# ──────────────────────────────────────────────
db:
  driver: postgres          # "postgres" or "sqlite"
  # PostgreSQL:
  host: localhost
  username: checker
  password: ""
  database: checker
  # database_url: postgres://user:pass@host:5432/checker  # alternative
  # SQLite:
  # driver: sqlite
  # dsn: ./checker.db

# ──────────────────────────────────────────────
# Defaults (applied to all checks unless overridden)
# ──────────────────────────────────────────────
defaults:
  duration: 30s             # Check interval
  alerts_channel: telegram  # Default alert channel name (must match a key in alerts:)

# ──────────────────────────────────────────────
# Alert Channels
# ──────────────────────────────────────────────
alerts:
  # Each key is a channel name you reference from checks.
  # Include ONLY the channels the user actually needs.

  telegram:
    type: telegram
    bot_token: "BOT_TOKEN"
    critical_channel: "CHAT_ID"
    noncritical_channel: "CHAT_ID"

  slack:
    type: slack
    webhook_url: "https://hooks.slack.com/services/T.../B.../xxx"

  discord:
    type: discord
    bot_token: "BOT_TOKEN"
    app_id: "APP_ID"
    default_channel: "CHANNEL_ID"

  pagerduty:
    type: pagerduty
    routing_key: "ROUTING_KEY"

  opsgenie:
    type: opsgenie
    api_key: "API_KEY"
    region: "us"              # "us" or "eu"

  email:
    type: email
    smtp_host: "smtp.example.com"
    smtp_port: 587
    smtp_user: "alerts@example.com"
    smtp_password: "PASSWORD"
    from: "alerts@example.com"
    to: ["oncall@example.com"]
    use_tls: true

  teams:
    type: teams
    webhook_url: "https://outlook.webhook.office.com/webhookb2/..."

  ntfy:
    type: ntfy
    topic: "checker-alerts"
    server_url: "https://ntfy.sh"   # default; use your own server if self-hosted
    # token: ""                     # optional auth
    # click_url: "https://checker.example.com"

# ──────────────────────────────────────────────
# Bot Integrations (optional — for interactive commands)
# ──────────────────────────────────────────────
# slack_app:
#   bot_token: "xoxb-..."
#   signing_secret: "SECRET"
#   default_channel: "C123456"

# telegram_app:
#   bot_token: "123456:ABCdef..."
#   secret_token: "SECRET"
#   default_chat_id: "CHAT_ID"
#   webhook_url: "https://checker.example.com/telegram/webhook"

# discord_app:
#   bot_token: "BOT_TOKEN"
#   app_id: "APP_ID"
#   public_key: "PUBLIC_KEY"
#   default_channel: "CHANNEL_ID"

# ──────────────────────────────────────────────
# Authentication (optional)
# ──────────────────────────────────────────────
# auth:
#   password: "your-password"
#   # OR OIDC:
#   oidc:
#     issuer_url: "https://auth.example.com"
#     client_id: "CLIENT_ID"
#     client_secret: "CLIENT_SECRET"
#     redirect_url: "https://checker.example.com/auth/callback"
#   api_keys: ["key1", "key2"]

# ──────────────────────────────────────────────
# Multi-Region Consensus (optional)
# ──────────────────────────────────────────────
# consensus:
#   region: "us-east-1"
#   min_regions: 2
#   evaluation_interval: "10s"
#   timeout: "30s"

# ──────────────────────────────────────────────
# Projects & Checks
# ──────────────────────────────────────────────
projects:
  # Group checks into projects. Each project can override defaults.
  my_project:
    healthchecks:
      web:
        checks:
          api_health:
            type: http
            url: "https://api.example.com/health"
            # ... check-specific fields (see reference below)
```

---

## Check Types Reference

Generate checks using the types below. Only include fields that are relevant.

### HTTP
```yaml
type: http
url: "https://example.com/health"
timeout: "10s"
code: [200]                           # Expected status codes (default: [200])
answer: "ok"                          # Expected substring in response body
answer_present: true                  # true = body must contain answer; false = must NOT contain
headers: ["Authorization: Bearer TOKEN"]
cookies: ["session=abc"]
skip_check_ssl: false                 # Skip TLS verification
ssl_expiration_period: "30d"          # Alert if SSL cert expires within this period
stop_follow_redirects: false
auth:
  user: "username"
  password: "password"
```

### TCP
```yaml
type: tcp
host: "db.example.com"
port: 5432
timeout: "10s"
```

### ICMP (Ping)
```yaml
type: icmp
host: "server.example.com"
count: 4                              # Ping count
timeout: "10s"
```

### DNS
```yaml
type: dns
domain: "example.com"
record_type: "A"                      # A, AAAA, MX, TXT, NS, CNAME
host: "8.8.8.8"                      # Custom resolver (optional)
expected: "93.184.216.34"            # Expected value (optional)
timeout: "10s"
```

### SSH
```yaml
type: ssh
host: "server.example.com"
port: 22
timeout: "10s"
expect_banner: "SSH-2.0"             # Expected banner prefix (optional)
```

### SSL Certificate
```yaml
type: ssl_cert
host: "example.com"
port: 443
timeout: "10s"
expiry_warning_days: 30
validate_chain: true
```

### SMTP
```yaml
type: smtp
host: "mail.example.com"
port: 587
timeout: "10s"
starttls: true
```

### Domain Expiry (WHOIS)
```yaml
type: domain_expiry
domain: "example.com"
expiry_warning_days: 30
timeout: "10s"
```

### PostgreSQL
```yaml
# Simple query check
type: pgsql_query
host: "db.example.com"
port: 5432
timeout: "10s"
pgsql:
  username: "monitor"
  password: "PASSWORD"
  dbname: "mydb"
  sslmode: "prefer"                   # disable, allow, prefer, require
  query: "SELECT 1"
  response: "1"

# Replication lag
type: pgsql_replication
pgsql:
  username: "monitor"
  password: "PASSWORD"
  dbname: "mydb"
  lag: "10s"
  server_list: ["replica1:5432", "replica2:5432"]

# Timestamp freshness
type: pgsql_query_unixtime
pgsql:
  query: "SELECT EXTRACT(EPOCH FROM updated_at)::int FROM jobs ORDER BY updated_at DESC LIMIT 1"
  difference: "300s"                  # Alert if older than 5 minutes
```

### MySQL
```yaml
# Simple query check
type: mysql_query
host: "db.example.com"
port: 3306
timeout: "10s"
mysql:
  username: "monitor"
  password: "PASSWORD"
  dbname: "mydb"
  query: "SELECT 1"
  response: "1"

# Replication lag
type: mysql_replication
mysql:
  username: "monitor"
  password: "PASSWORD"
  lag: "10s"
  server_list: ["replica1:3306"]
```

### Redis
```yaml
type: redis
host: "redis.example.com"
port: 6379
timeout: "10s"
password: "PASSWORD"                  # optional
db: 0
```

### MongoDB
```yaml
type: mongodb
mongodb_uri: "mongodb://user:pass@host:27017/dbname"
timeout: "10s"
```

### gRPC Health
```yaml
type: grpc_health
host: "service.example.com:50051"
timeout: "10s"
use_tls: false
```

### WebSocket
```yaml
type: websocket
url: "ws://example.com/ws"
timeout: "10s"
```

### Passive (Heartbeat)
```yaml
type: passive
timeout: "60s"                        # Alert if no ping received within this period
# Heartbeat endpoint: POST /api/checks/{uuid}/ping
```

---

## Common Check Options

Every check supports these optional fields:

```yaml
duration: "30s"                       # Check interval (overrides project/global default)
timeout: "10s"                        # Check timeout
severity: "critical"                  # "critical", "warning", "info"
enabled: true
retry_count: 2                        # Retry N times before marking as failed
retry_interval: "5s"                  # Wait between retries
re_alert_interval: "30m"             # Re-send alert every N if still failing
alert_channels: ["telegram", "slack"] # Send alerts to these channels
escalation_policy_name: "on_call"    # Escalation policy name
```

---

## Generation Guidelines

When generating the config:

1. **Group checks by project** — use meaningful project names (e.g., "Production API", "Databases", "Infrastructure")
2. **Set appropriate intervals** — critical services: 15-30s, standard: 1-5m, SSL/domain: 1h-24h
3. **Layer alerting** — critical checks to PagerDuty/OpsGenie + Slack; warnings to Slack/Telegram only
4. **Add SSL checks** for every HTTPS endpoint with `ssl_expiration_period: "30d"`
5. **Add DNS checks** for critical domains
6. **Add domain expiry** checks for owned domains
7. **Use retries** for flaky checks: `retry_count: 2, retry_interval: "5s"`
8. **Set re_alert_interval** to avoid alert fatigue: `"30m"` for critical, `"2h"` for warnings
9. **Replace all placeholder values** (BOT_TOKEN, PASSWORD, etc.) with comments showing what's needed
10. **Only include alert channels the user mentioned** — don't add unused ones

## Environment Variables

Sensitive values can be set via environment variables instead of hardcoding in YAML:

```bash
DATABASE_URL=postgres://user:pass@host:5432/checker
SLACK_BOT_TOKEN=xoxb-...
TELEGRAM_APP_BOT_TOKEN=123456:ABCdef...
AUTH_PASSWORD=secure-password
```

## Running Checker

```bash
# With config file
./checker -config config.yaml

# With Docker
docker run -v $(pwd)/config.yaml:/config.yaml ghcr.io/ensafely/checker -config /config.yaml

# Debug mode
./checker -config config.yaml --debug
```
