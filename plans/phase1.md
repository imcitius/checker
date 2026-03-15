# Phase 1 Implementation Plan — Foundation

**Parts covered:** Part 1 (Check Engine Expansion), Part 3 (Alerting & Notifications), Part 9 (Frontend UI Overhaul)

**Execution order:** Part 1 backend → Part 3 backend → Part 9 frontend

The reason for this order: the frontend forms for creating/editing checks need to know what check types exist. The alert channel UI needs to know what alert channels exist. Build the data model first, then the UI on top of it once — not twice.

---

## How the codebase is wired (read this first)

Before touching anything, understand the data flow:

1. **A check is defined** in the database as a `CheckDefinition` (see `internal/models/check_definition.go`). It has a `Type` string (e.g. `"http"`, `"tcp"`) and a `Config` field which is a Go interface (`CheckConfig`). The actual config struct depends on the type.

2. **BSON (de)serialization is manual** — the `UnmarshalBSON` and `MarshalBSON` methods on `CheckDefinition` handle converting between the database flat-row format and the polymorphic Go structs. Every new check type must be added to both methods.

3. **`CheckerFactory`** in `internal/scheduler/factories.go` reads a `CheckDefinition` and returns a `Checker` interface that knows how to `Run()`. Every new check type needs a case in this switch.

4. **The actual check logic** lives in `internal/checks/`. Each check type is its own file (e.g. `http.go`, `tcp.go`). Each implements the `Checker` interface: `Run() (time.Duration, error)`.

5. **Alert routing** is currently wired per-check via `AlertType` and `AlertDestination` fields on `CheckDefinition`. The scheduler reads these and dispatches to `internal/alerts/telegram.go` or `internal/alerts/slack.go`. The new notification channels follow the same pattern — add a file in `internal/alerts/`, wire it in the scheduler.

6. **The frontend** is a React SPA in `frontend/src/`. It talks to the Go backend via REST (`internal/web/`). The check edit form is in `frontend/src/components/CheckEditDrawer.tsx`. New check types need new form fields there.

7. **Tests** live next to the code: `internal/checks/http_test.go` next to `http.go`. Follow this pattern.

---

## Part 1: Check Engine Expansion

### What we're adding

15 new check types. Each follows the same pattern as existing ones.

**Tier 1 — High value, implement first:**
- `dns` — DNS record resolution
- `ssl_cert` — standalone TLS certificate expiry + chain check
- `smtp` — mail server connectivity and STARTTLS
- `ssh` — TCP connect + SSH handshake + banner grab
- `redis` — PING command, optional AUTH, INFO metrics
- `mongodb` — connectivity + `ping` command
- `domain_expiry` — WHOIS-based domain expiration

**Tier 2 — Implement second:**
- `grpc` — gRPC health check protocol
- `websocket` — WebSocket connect + optional message exchange
- `udp` — UDP port reachability (DNS-over-UDP as test target)
- `smtp_imap` — IMAP mailbox access (CAPABILITY command)
- `docker` — Docker Engine API container health

**Tier 3 — Implement last (lower complexity/value tradeoff):**
- `kafka` — Kafka broker connectivity + consumer group lag
- `icmp_enhanced` — extends existing ICMP with packet loss %, jitter
- `heartbeat` — improves the existing `passive` type with richer config

**Check engine improvements (alongside new types):**
- Configurable retry logic before alerting
- Sub-minute check intervals (down to 10s)
- Check dependencies (don't alert on child if parent is down)
- Maintenance windows

---

### Task 1.1 — DNS Check

**File to create:** `internal/checks/dns.go`

**Config struct to add in** `internal/models/check_types.go`:
```go
type DNSCheckConfig struct {
    Host       string `json:"host"`        // DNS resolver to query (empty = system default)
    Domain     string `json:"domain"`      // Domain to resolve
    RecordType string `json:"record_type"` // A, AAAA, CNAME, MX, TXT, NS
    Expected   string `json:"expected"`    // Expected value in the answer (optional)
    Timeout    string `json:"timeout"`
}
```

**What the check does:**
1. Parse `Timeout` as a `time.Duration`.
2. Use `net.DefaultResolver.LookupHost` (for A/AAAA) or `net.LookupMX`, `net.LookupTXT`, etc. depending on `RecordType`.
3. If `Expected` is set, verify it appears in the results.
4. Return success + elapsed time, or an error.

**Go standard library is enough** — no external packages needed for basic DNS.

**Tests in** `internal/checks/dns_test.go` — use a known public domain like `google.com` for A record.

**Wire it up:**
- Add `DNSCheckConfig` to `MarshalBSON`/`UnmarshalBSON` in `check_definition.go` under type string `"dns"`
- Add `*models.DNSCheckConfig` case in `CheckerFactory` in `factories.go`
- Register `"dns"` as a valid type in `internal/models/check_types.go`

---

### Task 1.2 — SSL Certificate Check (standalone)

**File to create:** `internal/checks/ssl_cert.go`

**Config struct:**
```go
type SSLCertCheckConfig struct {
    Host              string `json:"host"`               // hostname (no https://)
    Port              int    `json:"port"`               // default 443
    Timeout           string `json:"timeout"`
    ExpiryWarningDays int    `json:"expiry_warning_days"` // alert if cert expires within N days
    ValidateChain     bool   `json:"validate_chain"`      // verify full cert chain
}
```

**What the check does:**
1. Open a TLS connection to `host:port` using `crypto/tls`.
2. Get `tls.ConnectionState().PeerCertificates[0]` (leaf cert).
3. Check `cert.NotAfter` — fail if it's less than `ExpiryWarningDays` days away.
4. If `ValidateChain` is true, verify the full chain against system roots.
5. Return remaining days in the success message.

This is different from the HTTP check's SSL sub-check — this works on any port (not just HTTPS) and is the primary metric, not a side check.

---

### Task 1.3 — SMTP Check

**File to create:** `internal/checks/smtp.go`

**Config struct:**
```go
type SMTPCheckConfig struct {
    Host        string `json:"host"`
    Port        int    `json:"port"`   // 25, 465, 587
    Timeout     string `json:"timeout"`
    CheckTLS    bool   `json:"check_tls"` // attempt STARTTLS and verify it works
    ExpectBanner string `json:"expect_banner"` // optional: substring expected in greeting
}
```

**What the check does:**
1. Dial `host:port` with a timeout.
2. Read the SMTP greeting line (starts with `220`).
3. If `ExpectBanner` is set, verify it's in the greeting.
4. If `CheckTLS` is true, send `EHLO`, then `STARTTLS`, verify TLS upgrade succeeds.
5. Send `QUIT`. Return success.

Use `net.DialTimeout` + raw `bufio.Scanner` — the `net/smtp` package is fine too.

---

### Task 1.4 — SSH Check

**File to create:** `internal/checks/ssh.go`

**Config struct:**
```go
type SSHCheckConfig struct {
    Host          string `json:"host"`
    Port          int    `json:"port"`   // default 22
    Timeout       string `json:"timeout"`
    ExpectBanner  string `json:"expect_banner"` // optional: e.g. "OpenSSH"
}
```

**What the check does:**
1. Dial `host:port`.
2. Read the SSH banner line (e.g. `SSH-2.0-OpenSSH_8.9`).
3. If `ExpectBanner` is set, verify substring is present.
4. Close connection. We do NOT authenticate — banner grab is enough.

**Important:** Do not add `golang.org/x/crypto/ssh` as a dependency just for this. A raw TCP read of the first line is sufficient and avoids a large dependency.

---

### Task 1.5 — Redis Check

**File to create:** `internal/checks/redis.go`

**Config struct:**
```go
type RedisCheckConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`   // default 6379
    Password string `json:"password"`
    DB       int    `json:"db"`
    Timeout  string `json:"timeout"`
}
```

**What the check does:**
1. Dial TCP to `host:port`.
2. If `Password` is set, send `AUTH password\r\n` and read response.
3. Send `PING\r\n`, expect `+PONG\r\n`.
4. Return success.

Use raw TCP + RESP protocol — do NOT add the `go-redis` package unless we later need INFO metrics. RESP is trivially simple for PING/AUTH.

---

### Task 1.6 — MongoDB Check

**File to create:** `internal/checks/mongodb.go`

**Config struct:**
```go
type MongoDBCheckConfig struct {
    URI     string `json:"uri"`     // full MongoDB URI including auth if needed
    Timeout string `json:"timeout"`
}
```

**What the check does:**
1. Use `go.mongodb.org/mongo-driver` (already in `go.mod`) to create a client.
2. Call `client.Ping(ctx, nil)`.
3. Disconnect. Return success or error.

This one CAN use the existing MongoDB driver since it's already a dependency.

---

### Task 1.7 — Domain Expiry Check

**File to create:** `internal/checks/domain_expiry.go`

**Config struct:**
```go
type DomainExpiryCheckConfig struct {
    Domain            string `json:"domain"`
    Timeout           string `json:"timeout"`
    ExpiryWarningDays int    `json:"expiry_warning_days"` // alert if expiring within N days
}
```

**What the check does:**
1. Run a WHOIS query against the appropriate WHOIS server. Use `golang.org/x/net` or a raw TCP connection to port 43 of the WHOIS server.
2. Parse the expiry date from the WHOIS response. This requires regex matching for common formats like `Expiry Date: 2025-03-15T00:00:00Z`.
3. Fail if expiry is within `ExpiryWarningDays` days.

**Note to agent:** WHOIS parsing is fragile — different registrars format the expiry date differently. Handle the most common formats (RFC3339, `DD-Mon-YYYY`, `YYYY-MM-DD`). Wrap parsing errors gracefully. This is a best-effort check.

Add `golang.org/x/net` to `go.mod` if not present, or use a small WHOIS library if one exists in the dependency tree already.

---

### Task 1.8 — gRPC Health Check

**File to create:** `internal/checks/grpc_health.go`

**Config struct:**
```go
type GRPCCheckConfig struct {
    Host    string `json:"host"`    // host:port
    Service string `json:"service"` // gRPC service name (empty = check all)
    UseTLS  bool   `json:"use_tls"`
    Timeout string `json:"timeout"`
}
```

**What the check does:**
1. Connect to `host` using `google.golang.org/grpc`.
2. Call the standard gRPC health check RPC: `grpc.health.v1.Health/Check`.
3. Verify the response status is `SERVING`.

**Dependency:** Add `google.golang.org/grpc` and `google.golang.org/grpc/health/grpc_health_v1` to `go.mod`.

---

### Task 1.9 — WebSocket Check

**File to create:** `internal/checks/websocket.go`

**Config struct:**
```go
type WebSocketCheckConfig struct {
    URL            string `json:"url"`             // ws:// or wss://
    SendMessage    string `json:"send_message"`    // optional: message to send after connect
    ExpectMessage  string `json:"expect_message"`  // optional: expected substring in response
    Timeout        string `json:"timeout"`
}
```

**What the check does:**
1. Use `github.com/gorilla/websocket` (already in `go.mod`) to dial the URL.
2. If `SendMessage` is set, write it as a text frame.
3. If `ExpectMessage` is set, read the next frame and verify substring.
4. Close cleanly. Return success.

---

### Task 1.10 — Retry Logic and Sub-Minute Intervals

These are engine improvements, not new check types. They touch the scheduler.

**Retry logic** — Add two fields to `CheckDefinition`:
```go
RetryCount    int    `json:"retry_count"`    // how many times to retry before declaring failure
RetryInterval string `json:"retry_interval"` // how long to wait between retries
```

In `internal/scheduler/scheduler.go` (or `worker_pool.go`), after a check returns an error, retry up to `RetryCount` times with `RetryInterval` sleep before recording the failure and triggering alerts.

**Sub-minute intervals** — The scheduler currently parses `Duration` as a `time.Duration`. Sub-minute intervals (e.g. `"10s"`, `"30s"`) should already work if the heap timer logic uses `time.Duration`. Verify the scheduler's `heap.go` handles sub-minute durations correctly. If not, fix it. Add a test with `10s` duration.

**Important:** Sub-minute intervals increase DB write frequency. Ensure the DB connection pool is sized appropriately. Document this in code comments.

---

### Task 1.11 — Maintenance Windows

Add a `MaintenanceUntil` field to `CheckDefinition`:
```go
MaintenanceUntil *time.Time `json:"maintenance_until,omitempty"`
```

In the scheduler, before executing a check, test: if `MaintenanceUntil != nil && time.Now().Before(*MaintenanceUntil)` — skip the check and do not alert.

Add a REST API endpoint to set/clear maintenance windows:
- `PUT /api/checks/{uuid}/maintenance` — body: `{"until": "2024-03-15T15:00:00Z"}`
- `DELETE /api/checks/{uuid}/maintenance` — clear it

Add a button in the frontend check edit drawer to set a maintenance window duration (15m, 1h, 4h, 24h, custom).

---

### Task 1.12 — Register All New Types in Model Layer

This is a housekeeping task that must happen before any of the new checks can be created via API.

In `internal/models/check_types.go`, ensure:
1. All new config structs have `CheckType()` and `GetTarget()` methods.
2. A `ValidCheckTypes` map or list exists that the import/validation logic uses.

In `internal/models/check_definition.go`, add `UnmarshalBSON`/`MarshalBSON` cases for each new type. Follow the exact pattern used for existing types.

In `internal/scheduler/factories.go`, add a `case` in `CheckerFactory` for each new config type.

In `internal/config/config.go`, update the YAML import parser to recognize the new type strings.

---

## Part 3: Alerting & Notifications

### What we're adding

New notification channels and a proper alert routing layer.

**Channels to add:**
- Email (SMTP)
- Discord webhook
- Microsoft Teams webhook
- PagerDuty Events API v2
- Opsgenie Alerts API
- Custom webhooks v2 (with templates and retry)

**Routing improvements:**
- Escalation policies (person A → if no ack in N min → person B)
- Alert severity levels (critical / warning / info)
- Alert deduplication (don't re-alert for same ongoing failure)
- Per-channel notification templates

---

### Task 3.1 — Email (SMTP) Alerter

**File to create:** `internal/alerts/email.go`

**Config (in main `config.yaml` alerts section):**
```yaml
alerts:
  email:
    type: email
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_user: alerts@example.com
    smtp_password: secret
    from: "Checker Alerts <alerts@example.com>"
    to: ["ops@example.com"]
    use_tls: true
```

**What it does:**
1. Use Go's `net/smtp` package.
2. Build a plain-text + HTML email body from a template.
3. Dial with STARTTLS or TLS depending on port/config.
4. Send and handle errors.

**Email template** — keep it simple. Subject: `[ALERT] {check_name} is DOWN` or `[RESOLVED] {check_name} is UP`. Body includes check name, project, error message, timestamp.

Create `internal/alerts/templates/email_alert.html` and `email_alert.txt` for the templates.

---

### Task 3.2 — Discord Alerter

**File to create:** `internal/alerts/discord.go`

Discord uses simple webhook POST with JSON payload. No SDK needed.

**Config:**
```yaml
alerts:
  discord:
    type: discord
    webhook_url: https://discord.com/api/webhooks/...
```

**Payload format:**
```json
{
  "embeds": [{
    "title": "🔴 check_name is DOWN",
    "description": "Error: ...",
    "color": 15158332,
    "timestamp": "2024-03-15T12:00:00Z"
  }]
}
```

Use color `15158332` (red) for failures, `3066993` (green) for recoveries.

---

### Task 3.3 — Microsoft Teams Alerter

**File to create:** `internal/alerts/teams.go`

Teams uses Incoming Webhooks with an Adaptive Card payload (or legacy MessageCard). Use the legacy MessageCard format for simplicity — it's still supported and requires no SDK.

**Payload:**
```json
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "FF0000",
  "summary": "check_name is DOWN",
  "sections": [{
    "activityTitle": "check_name",
    "activitySubtitle": "Project: project_name",
    "facts": [
      {"name": "Status", "value": "DOWN"},
      {"name": "Error", "value": "..."},
      {"name": "Time", "value": "..."}
    ]
  }]
}
```

---

### Task 3.4 — PagerDuty Integration

**File to create:** `internal/alerts/pagerduty.go`

Use PagerDuty Events API v2. No SDK needed — it's a simple HTTPS POST.

**Config:**
```yaml
alerts:
  pagerduty:
    type: pagerduty
    routing_key: "your-integration-key"
```

**Trigger payload** (POST to `https://events.pagerduty.com/v2/enqueue`):
```json
{
  "routing_key": "...",
  "event_action": "trigger",
  "dedup_key": "{check_uuid}",
  "payload": {
    "summary": "check_name is DOWN: error message",
    "source": "checker",
    "severity": "critical"
  }
}
```

For recovery, send `"event_action": "resolve"` with the same `dedup_key`.

The `dedup_key` is critical — it's how PagerDuty links trigger and resolve events to the same incident. Use the check UUID as the dedup key.

---

### Task 3.5 — Opsgenie Integration

**File to create:** `internal/alerts/opsgenie.go`

Similar pattern to PagerDuty. Use Opsgenie Alert API v2.

**Config:**
```yaml
alerts:
  opsgenie:
    type: opsgenie
    api_key: "your-api-key"
    region: "us"  # or "eu"
```

POST to `https://api.opsgenie.com/v2/alerts` (or `api.eu.opsgenie.com` for EU).

For resolve: POST to `https://api.opsgenie.com/v2/alerts/{alias}/close`.

Use check UUID as the alert alias for linking.

---

### Task 3.6 — Alert Severity Levels

Add `Severity` field to `CheckDefinition`:
```go
Severity string `json:"severity"` // "critical", "warning", "info" — default "critical"
```

In the scheduler's alert dispatch logic, pass severity to the alerter. Alerters should reflect severity in their messages (color, subject line, PagerDuty severity field).

Also add `AlertChannels []string` to `CheckDefinition` to allow a check to notify multiple channels:
```go
AlertChannels []string `json:"alert_channels"` // e.g. ["telegram", "pagerduty", "email"]
```

This replaces the current single `AlertType` + `AlertDestination` pattern. Keep backward compatibility — if `AlertChannels` is empty and `AlertType` is set, use the old behavior.

---

### Task 3.7 — Alert Deduplication

Currently, if a check stays DOWN across multiple poll cycles, it sends a new alert every time the check runs. This is noisy.

**The fix:** Only send an alert when the state *changes*:
- Healthy → Unhealthy: send DOWN alert
- Unhealthy → Healthy: send RECOVERY alert
- Unhealthy → Unhealthy: do nothing

`CheckDefinition` already has `IsHealthy` and `LastAlertSent` fields. The scheduler already has some dedup logic via `LastAlertSent`. Review it and ensure:
1. Alerts only fire on state transition.
2. Recovery alerts are sent when `IsHealthy` flips back to `true`.
3. Add an optional `ReAlertInterval` field: if set, re-alert for ongoing failures every N minutes (for escalation scenarios).

---

### Task 3.8 — Escalation Policies (Basic)

Add a `EscalationPolicy` struct. Keep it simple for Phase 1:

```go
type EscalationStep struct {
    Channel    string `json:"channel"`    // alert channel name
    DelayMin   int    `json:"delay_min"`  // minutes after initial failure before alerting this step
}

type EscalationPolicy struct {
    Name  string           `json:"name"`
    Steps []EscalationStep `json:"steps"`
}
```

Store escalation policies in a new DB table `escalation_policies`.

A check references an escalation policy by name in its `EscalationPolicyName string` field.

The scheduler evaluates: if a check has been DOWN for `step.DelayMin` minutes and this step hasn't been notified yet, send the notification.

Track which escalation steps have been notified in a new DB table `escalation_notifications` with columns `check_uuid`, `policy_name`, `step_index`, `notified_at`.

**DB migrations:** Create `000005_escalation_policies.up.sql` and `.down.sql`.

---

### Task 3.9 — Notification Channel Config UI

In `frontend/src/pages/`, add a new page: `Settings.tsx`.

For Phase 1, the settings page shows only Notification Channels:
- List configured channels
- Add/edit/delete channels
- Test a channel (send a test alert)

Backend: add REST endpoints:
- `GET /api/alert-channels` — list configured channels
- `POST /api/alert-channels` — create channel
- `PUT /api/alert-channels/{name}` — update
- `DELETE /api/alert-channels/{name}` — delete
- `POST /api/alert-channels/{name}/test` — send test alert

Store channel configs in a new DB table `alert_channels` (name, type, config JSON).

---

## Part 9: Frontend UI Overhaul

Do this part after Part 1 and Part 3 backend work is complete. That way, the forms are built once with all check types and alert channels.

### What we're building

The current UI is functional but rough. The goal is to make it look like something you'd pay for.

---

### Task 9.1 — Dark Mode

Add a dark/light theme toggle. The project uses Tailwind.

In `frontend/src/lib/theme.tsx` (already exists), implement:
1. Read system preference via `window.matchMedia('(prefers-color-scheme: dark)')`.
2. Allow manual override stored in `localStorage`.
3. Apply `dark` class to `<html>` element.

Ensure all existing components have proper `dark:` Tailwind variants. This is mostly CSS work — go through each component and add `dark:bg-*`, `dark:text-*`, `dark:border-*` as needed.

---

### Task 9.2 — Command Palette (Cmd+K)

The `useKeyboard.ts` hook and `command.tsx` UI component already exist. Wire them up.

Implement a command palette that:
1. Opens on `Cmd+K` / `Ctrl+K`.
2. Lets you search checks by name, project, type.
3. Lets you jump to pages (Dashboard, Management, Alerts, Settings).
4. Lets you trigger actions (e.g. "Enable check X", "Go to check X").

The `Command` component from `components/ui/command.tsx` is likely already a `cmdk` wrapper. Use it.

---

### Task 9.3 — Health Map Improvements

`HealthMap.tsx` already exists. Improve it:

1. **Sparklines** — Add inline mini response-time charts per check row. Use `recharts` or a simple SVG sparkline. The backend needs a `/api/checks/{uuid}/metrics?window=1h` endpoint returning recent response times (see Part 6 note below — for Phase 1, just return the last 20 check results from the existing DB).

2. **Check status color states** — Currently binary (green/red). Add yellow for "degraded" (high response time) and gray for "unknown" / "maintenance".

3. **Tooltips** — Hover over a check to see: last check time, response time, error message if down.

---

### Task 9.4 — Bulk Actions on Management Page

In `frontend/src/pages/Management.tsx`:

1. Add checkboxes to each check row (using existing `CheckRow.tsx`).
2. Show a bulk action bar when any checks are selected: Enable / Disable / Delete / Set Maintenance.
3. Backend: add bulk endpoints:
   - `POST /api/checks/bulk-enable` — body: `{"uuids": [...]}`
   - `POST /api/checks/bulk-disable`
   - `POST /api/checks/bulk-delete`

---

### Task 9.5 — Check Edit Drawer — New Check Types

`CheckEditDrawer.tsx` currently shows fields for existing check types. Add form sections for all new types from Part 1.

Structure it so each check type has its own tab or collapsible section within the drawer (tabs already exist as a pattern in the codebase).

For each new check type, add:
- The relevant config fields as form inputs
- Field validation (required fields, format checks)
- Help text explaining what each field does (tooltips or small descriptions below fields)

---

### Task 9.6 — Alert Channel Configuration in Check Edit Drawer

Currently, a check has a single `AlertType` + `AlertDestination`. Replace this with:
- A multi-select of configured alert channels (populated from `/api/alert-channels`)
- A severity selector (critical / warning / info)
- An escalation policy selector (if any policies are configured)
- An optional re-alert interval input

---

### Task 9.7 — Mobile Responsiveness

The current layout is desktop-first. Make it usable on mobile (tablet and phone):
1. The top nav collapses to a hamburger menu on small screens.
2. The management table becomes a card list on mobile.
3. The check edit drawer is full-screen on mobile.
4. The health map grid adapts column count based on screen width.

Use Tailwind responsive prefixes (`sm:`, `md:`, `lg:`). No new libraries needed.

---

### Task 9.8 — Performance: SPA Build Automation

Currently the frontend must be built manually before embedding into the Go binary. The pre-commit hook exists (`dev/hooks/`) but may not be wired correctly.

1. Verify the pre-commit hook rebuilds the frontend on commit.
2. Add a `make frontend` target to `Makefile` that runs `cd frontend && npm run build`.
3. Ensure `make build` depends on `make frontend`.
4. Update the Dockerfile to use the Makefile targets.

---

## DB Migrations Required

Create these migration files in `migrations/`:

| Migration | Description |
|-----------|-------------|
| `000005_escalation_policies.up.sql` | `escalation_policies` table and `escalation_notifications` table |
| `000006_alert_channels.up.sql` | `alert_channels` table (name, type, config JSONB) |
| `000007_check_enhancements.up.sql` | Add `retry_count`, `retry_interval`, `maintenance_until`, `severity`, `alert_channels`, `escalation_policy_name`, `re_alert_interval` to `check_definitions` |

Each migration must have a corresponding `.down.sql` that reverses it exactly.

---

## New Go Dependencies

Add these to `go.mod` (run `go get` for each):

| Package | Used for |
|---------|----------|
| `google.golang.org/grpc` | gRPC health check |
| `google.golang.org/grpc/health/grpc_health_v1` | gRPC health proto |

The following are already present: `gorilla/websocket`, `go-ping/ping`, `go.mongodb.org/mongo-driver`.

For SMTP, SSH, DNS, Redis — use Go stdlib or raw TCP. No new packages needed.

For domain WHOIS — use raw TCP to port 43. No package needed.

---

## Testing Requirements

Every new check type must have tests in `internal/checks/{type}_test.go`.

Test structure:
1. **Unit tests** — test with a mock server or known-good endpoint where possible.
2. **Error tests** — test what happens when the host is unreachable, connection times out, wrong response.
3. **Config validation tests** — test that missing required fields return clear errors.

For integration tests that need a real server (Redis, MongoDB, SMTP), use the existing `INTEGRATION_TESTS=true` env var pattern.

For new alert channels, test in `internal/alerts/{channel}_test.go` with a mock HTTP server that captures the webhook payload.

---

## Implementation Order Within Phase 1

1. Task 1.12 — model layer housekeeping first (no logic, just registration)
2. Tasks 1.1 through 1.10 — new check types (can be done in parallel by separate agents)
3. Tasks 3.1 through 3.5 — new alert channels (parallel)
4. Tasks 3.6, 3.7 — severity + deduplication (scheduler changes)
5. Task 3.8 — escalation policies (needs DB migration 000005)
6. DB migrations 000005, 000006, 000007
7. Task 3.9 — alert channel config UI (needs backend from 3.x done)
8. Tasks 9.1, 9.2, 9.7 — dark mode, command palette, mobile (independent)
9. Tasks 9.3, 9.4 — health map + bulk actions
10. Tasks 9.5, 9.6 — check edit drawer (needs all Part 1 + Part 3 backend done)
11. Task 9.8 — build automation (last, after all code is stable)

---

## Definition of Done for Phase 1

Phase 1 is complete when:

1. `go test ./...` passes with no failures.
2. All 15 new check types can be created via API and execute correctly.
3. All 5 new alert channels send correct payloads (verified by unit tests with mock servers).
4. Escalation policies work end-to-end: create policy, assign to check, verify step-by-step alerts fire at correct delays.
5. Alert deduplication is verified: a check that stays DOWN does not send repeat alerts (except when `ReAlertInterval` is configured).
6. Dark mode works across all pages.
7. Command palette opens, searches, and navigates correctly.
8. Management page bulk actions work.
9. Check edit drawer supports all new types and alert channel multi-select.
10. Frontend builds cleanly via `make frontend` and `make build`.
11. All new DB migrations run cleanly in both up and down directions.
12. Docker build produces a working image.
