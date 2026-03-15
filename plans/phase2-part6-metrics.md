# Phase 2 — Part 6: Metrics, Analytics & Reporting — LLD

**Dependency for:** Status page uptime bars (Part 5), check detail charts (frontend)
**Build this first** in Phase 2.

---

## 1. Data Model

### `check_results` table

```sql
-- migrations/000010_check_results.up.sql
CREATE TABLE check_results (
    id          BIGSERIAL PRIMARY KEY,
    check_uuid  TEXT NOT NULL,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms INTEGER NOT NULL,         -- response time in milliseconds
    is_healthy  BOOLEAN NOT NULL,
    message     TEXT,                     -- error message or success detail
    check_type  TEXT NOT NULL             -- denormalized for query efficiency
);

CREATE INDEX idx_check_results_uuid_time ON check_results (check_uuid, checked_at DESC);
CREATE INDEX idx_check_results_time ON check_results (checked_at DESC);

-- TimescaleDB hypertable (only runs if TimescaleDB extension is available)
-- The scheduler detects TimescaleDB at startup and runs this if present:
-- SELECT create_hypertable('check_results', 'checked_at', if_not_exists => TRUE);
```

**Design notes:**
- `duration_ms` is integer milliseconds, not a duration string. Easier to aggregate.
- `check_type` is denormalized — avoids a join on `check_definitions` for every metrics query. Stale if a check type changes (rare), acceptable trade-off.
- No foreign key constraint on `check_uuid` — metrics outlive deleted checks intentionally. We want to retain history even after a check is removed.
- `message` is nullable. On success it may contain useful detail (e.g. "cert expires in 45 days"). On failure it contains the error.

**Corner cases:**
- A check that runs every 10s generates 8,640 rows/day. At 100 checks, that's 864,000 rows/day, 78M rows/90 days. Fine for Postgres with good indexes; TimescaleDB handles it better. Document the recommendation.
- `duration_ms` overflow: max int32 is ~2.1 billion ms = ~24 days. Use BIGINT to be safe, even though no check will ever take that long.
- Clock skew: if the host clock is adjusted backward, `checked_at` could appear out of order. The index handles this fine; queries always use `ORDER BY checked_at DESC`.

---

## 2. Writing Results

### Scheduler integration

In `internal/scheduler/scheduler.go`, after recording a check result to `check_definitions` (updating `is_healthy`, `last_message`, etc.), also write to `check_results`:

```go
result := models.CheckResult{
    CheckUUID:  checkDef.UUID,
    CheckedAt:  time.Now(),
    DurationMs: int64(elapsed.Milliseconds()),
    IsHealthy:  isHealthy,
    Message:    lastMessage,
    CheckType:  checkDef.Type,
}
if err := repo.CreateCheckResult(ctx, result); err != nil {
    // Non-fatal — log and continue. Never block alerting on metrics write failure.
    logrus.Warnf("Failed to record metric for %s: %v", checkDef.UUID, err)
}
```

**Critical:** metrics write failure must never block the scheduler or delay alerting. Use a fire-and-forget pattern with a short timeout context (5s max).

**SQLite mode:** `CreateCheckResult` on the SQLite repository is a no-op that returns nil immediately.

### Repository interface additions

```go
// Add to internal/db/repository.go:
CreateCheckResult(ctx context.Context, result models.CheckResult) error
GetCheckResults(ctx context.Context, checkUUID string, from, to time.Time, limit int) ([]models.CheckResult, error)
GetUptimeStats(ctx context.Context, checkUUID string, window time.Duration) (UptimeStats, error)
GetCheckResultsSummary(ctx context.Context, checkUUIDs []string, window time.Duration) (map[string]UptimeStats, error)
PurgeOldCheckResults(ctx context.Context, olderThan time.Time) (int64, error)
```

### Model

```go
// internal/models/metrics.go
type CheckResult struct {
    ID         int64     `json:"id"`
    CheckUUID  string    `json:"check_uuid"`
    CheckedAt  time.Time `json:"checked_at"`
    DurationMs int64     `json:"duration_ms"`
    IsHealthy  bool      `json:"is_healthy"`
    Message    string    `json:"message,omitempty"`
    CheckType  string    `json:"check_type"`
}

type UptimeStats struct {
    CheckUUID      string        `json:"check_uuid"`
    Window         string        `json:"window"`           // "24h", "7d", "30d", "90d"
    UptimePct      float64       `json:"uptime_pct"`       // 0-100
    TotalChecks    int           `json:"total_checks"`
    HealthyChecks  int           `json:"healthy_checks"`
    AvgDurationMs  int64         `json:"avg_duration_ms"`
    P95DurationMs  int64         `json:"p95_duration_ms"`
    P99DurationMs  int64         `json:"p99_duration_ms"`
    MaxDurationMs  int64         `json:"max_duration_ms"`
}
```

**Corner case — uptime calculation:** what counts as "up"? Only healthy results? What about periods with no results (check was disabled, server was down)? Decision: uptime% = healthy_count / total_count * 100 where total_count is all rows in the window. Periods with no rows (e.g. check disabled) are excluded entirely — don't count against uptime. Document this definition clearly in the UI.

---

## 3. Retention & Purging

### Background goroutine

In `cmd/app/main.go`, start a goroutine that runs daily:

```go
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            retentionDays := 90
            if v := os.Getenv("METRICS_RETENTION_DAYS"); v != "" {
                if n, err := strconv.Atoi(v); err == nil && n > 0 {
                    retentionDays = n
                }
            }
            cutoff := time.Now().AddDate(0, 0, -retentionDays)
            n, err := repo.PurgeOldCheckResults(ctx, cutoff)
            if err != nil {
                logrus.Warnf("Failed to purge old metrics: %v", err)
            } else {
                logrus.Infof("Purged %d check results older than %d days", n, retentionDays)
            }
        case <-ctx.Done():
            return
        }
    }
}()
```

**Corner case:** first run on a large existing DB could delete millions of rows and lock the table for seconds. The `DELETE WHERE checked_at < $1` query should use a LIMIT per batch:

```sql
DELETE FROM check_results
WHERE id IN (
    SELECT id FROM check_results
    WHERE checked_at < $1
    LIMIT 10000
)
```

Run in a loop until 0 rows deleted. This batched approach keeps lock duration bounded.

---

## 4. REST API Endpoints

```
GET /api/checks/:uuid/metrics
    ?window=1h|6h|24h|7d|30d|90d  (default: 1h)
    ?limit=100                      (max 1000, for sparkline data)
    → { results: [CheckResult], stats: UptimeStats }

GET /api/checks/:uuid/uptime
    ?windows=24h,7d,30d,90d
    → { "24h": UptimeStats, "7d": UptimeStats, ... }

GET /api/metrics/summary
    → { check_uuid: UptimeStats, ... }  (all checks, 24h window, for dashboard)
```

**Corner case — large window + small check interval:** `/api/checks/:uuid/metrics?window=90d` on a 10s check = 777,600 rows. The API must never return raw rows for large windows. For windows > 24h, return downsampled data: one data point per hour (average duration, worst health status in that hour).

Downsampling query (Postgres):
```sql
SELECT
    date_trunc('hour', checked_at) AS bucket,
    AVG(duration_ms)::bigint AS avg_duration_ms,
    BOOL_AND(is_healthy) AS is_healthy,
    COUNT(*) AS sample_count
FROM check_results
WHERE check_uuid = $1 AND checked_at > NOW() - $2::interval
GROUP BY bucket
ORDER BY bucket ASC
```

For `window <= 1h`, return raw rows (max ~360 at 10s interval).

---

## 5. Prometheus Exporter

Endpoint: `GET /metrics` (no auth — standard Prometheus convention; document that users should firewall this if sensitive).

Metrics to expose:

```
# Check health (gauge: 1 = healthy, 0 = unhealthy)
checker_check_up{check_uuid="...", name="...", project="...", type="..."} 1

# Last response time
checker_check_duration_seconds{check_uuid="...", name="...", project="...", type="..."} 0.123

# Total check executions counter
checker_check_total{check_uuid="...", name="...", project="...", type="...", result="success|failure"} 1234

# Uptime percentage (24h window)
checker_uptime_ratio{check_uuid="...", name="...", project="...", window="24h"} 0.9987

# Active incidents (gauge)
checker_incidents_active{severity="P1|P2|P3|P4"} 2
```

Implementation: use `github.com/prometheus/client_golang/prometheus`. Register metrics in a custom registry (not the default global registry, to avoid conflicts with Go runtime metrics if not wanted). Expose via `promhttp.HandlerFor`.

**Corner case — cardinality:** if a user has 1000 checks, the `checker_check_up` metric has 1000 time series. This is fine for Prometheus (handles millions). But if check names or projects contain high-cardinality data (e.g. UUIDs in the name), label cardinality explodes. Enforce: only use `check_uuid`, `name`, `project`, `type` as labels. Never include error messages or dynamic values as labels.

**Corner case — auth:** the `/metrics` endpoint must be accessible to Prometheus without auth, but it leaks check names and project names. Add an optional `METRICS_TOKEN` env var: if set, require `Authorization: Bearer <token>` or `?token=<token>`. Document this. Default: no auth (standard Prometheus setup).

---

## 6. Frontend: Charts

### Dashboard sparklines (already partially done via MetricsRow)

Enhance `MetricsRow.tsx` / `HealthMap.tsx`:
- Fetch `/api/checks/:uuid/metrics?window=1h&limit=60` for each visible check
- Render a 60-point sparkline (SVG or recharts `<Sparkline>`)
- Colour: green line if currently healthy, red if unhealthy, gray if no data

**Corner case — N+1 fetches:** if the dashboard has 50 checks and each fetches its own metrics, that's 50 concurrent API calls on page load. Instead, use `/api/metrics/summary` which returns stats for all checks in one call. For the sparkline data specifically, batch: `POST /api/checks/metrics/batch` with a list of UUIDs, returns a map. Add this endpoint.

Alternatively: extend the WebSocket `/ws` event to include recent duration in the check status push. This avoids REST polling entirely for sparklines. Preferred approach.

### Check detail page

Add a `CheckDetail` page (or expand existing `CheckDetails.tsx`) with:
- Full response time chart (recharts `<LineChart>`) for selected window (1h / 6h / 24h / 7d / 30d)
- Uptime percentage badges for each window
- Incident history for this check (from Part 4)
- Recent results table (last 50 rows: timestamp, duration, status, message)

---

## 7. Uptime Calculation for Status Pages

Status pages (Part 5) need 90-day uptime bars per component. The bar is divided into 90 day-buckets. Each bucket is green (≥99% uptime that day), yellow (95–99%), or red (<95%). Gray if no data.

Query:
```sql
SELECT
    date_trunc('day', checked_at) AS day,
    BOOL_AND(is_healthy) AS all_healthy,
    COUNT(*) FILTER (WHERE is_healthy) AS healthy_count,
    COUNT(*) AS total_count
FROM check_results
WHERE check_uuid = ANY($1)  -- all checks in the component
  AND checked_at > NOW() - INTERVAL '90 days'
GROUP BY day
ORDER BY day ASC
```

The result is a map of day → uptime%, used by the status page renderer.

**Corner case — component has multiple checks:** a component's uptime is the worst-case across all linked checks. If check A was up but check B was down, the component was degraded. Use `MIN(healthy_count/total_count)` across checks for each day bucket.

**Corner case — no results for a day:** check was disabled, or the server was down. Show gray (unknown), not red. Don't penalise uptime for intentional maintenance windows. Cross-reference with `maintenance_until` from `check_definitions` to detect maintenance periods and colour them differently (e.g. blue/striped).

---

## 8. TimescaleDB Detection

At startup, attempt:
```sql
SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'
```

If present, run:
```sql
SELECT create_hypertable('check_results', 'checked_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);
```

Store the result in a package-level bool `metricsIsTimescaleDB`. Log at startup: "TimescaleDB detected — using hypertable for check_results" or "TimescaleDB not available — using standard Postgres table".

No code differences beyond this — all queries work on both. TimescaleDB just makes time-range queries faster.

---

## Files to Create / Modify

| File | Action |
|------|--------|
| `migrations/000010_check_results.up.sql` | Create |
| `migrations/000010_check_results.down.sql` | Create |
| `internal/models/metrics.go` | Create |
| `internal/db/repository.go` | Add 5 new interface methods |
| `internal/db/postgres.go` | Implement new methods |
| `internal/db/sqlite.go` | Implement as no-ops |
| `internal/scheduler/scheduler.go` | Write result after each check run |
| `internal/web/metrics_handlers.go` | New file: REST API handlers |
| `internal/web/prometheus.go` | New file: Prometheus exporter setup |
| `internal/web/server.go` | Wire new routes |
| `cmd/app/main.go` | Start purge goroutine, init Prometheus |
| `go.mod` | Add `prometheus/client_golang` |
| `frontend/src/components/MetricsRow.tsx` | Enhance sparklines |
| `frontend/src/pages/CheckDetail.tsx` | New page or enhance existing |
| `frontend/src/lib/api.ts` | Add metrics API calls |

---

## Acceptance Criteria

1. Every check execution writes a row to `check_results`
2. `GET /api/checks/:uuid/metrics?window=24h` returns downsampled data
3. `GET /metrics` returns valid Prometheus exposition format
4. Uptime % is correct for a check with known history (testable with seeded data)
5. Purge goroutine deletes rows older than `METRICS_RETENTION_DAYS`
6. SQLite mode: no writes to `check_results`, no errors, metrics endpoint returns empty
7. TimescaleDB: if extension present, hypertable created at startup
8. Dashboard sparklines load without N+1 requests
9. `go test ./internal/db/...` passes (mock the metrics writes)
