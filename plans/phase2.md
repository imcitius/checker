# Phase 2 Implementation Plan — Differentiation

**Parts covered:** Part 4 (Incident Management), Part 5 (Status Pages), Part 6 (Metrics, Analytics & Reporting)

**Goal:** Deliver the features that justify paid tiers and meaningfully beat free alternatives (Uptime Kuma, Checkly free, UptimeRobot free). After Phase 2, Checker is a credible commercial product.

---

## Architectural Principles

### What stays in Checker
Everything in Phase 2 lives inside the existing Go binary + React SPA. No new services.

### What is explicitly deferred
- **Subscriber notifications** (status page email/SMS opt-in lists) — this is a separate deliverability concern (bounce handling, unsubscribe, GDPR, sending reputation). Checker will emit webhook events; a future `checker-notify` microservice handles fan-out. Phase 2 status pages show an embeddable RSS feed only.
- **Billing / feature gating** — Part 10, Phase 4. No plan limits enforced in Phase 2.
- **SMS / voice alerting** — Twilio integration is Part 3 scope extension, Phase 3+.

### Migration strategy
All Phase 2 DB changes use the existing `golang-migrate` pattern. New tables only — no destructive changes to existing schema. All migrations numbered sequentially from `000010_*`.

### Data model philosophy
- Incidents are first-class entities, linked to checks but independent of them. A check going down creates an incident; the incident survives even if the check is deleted.
- Status pages are configuration objects that reference checks. One page can reference many checks; one check can appear on many pages. Many-to-many via a join table.
- Metrics are append-only. Never update a metric row. Purge old rows via a background job, not on write.

---

## Part 4: Incident Management

See: `plans/phase2-part4-incidents.md`

**Summary:** Incidents track the lifecycle of a service degradation from first alert to resolution and postmortem. They are auto-created when a check transitions to unhealthy, and auto-resolved when it recovers. Operators can also create incidents manually.

**Core entities:** `incidents`, `incident_updates`, `incident_check_links`

**Key design decisions:**
- Incident = single source of truth for an outage. Multiple checks can be linked to one incident.
- Status: `open → acknowledged → investigating → resolved`
- P1–P4 severity, independent of check severity
- Auto-creation: configurable per-check (`auto_incident: true`)
- Postmortem is a freeform markdown document attached to a resolved incident
- Timeline is append-only (updates, status changes, alert events, comments all land there)

---

## Part 5: Status Pages

See: `plans/phase2-part5-status-pages.md`

**Summary:** Public-facing pages showing the health of selected checks, organized into service groups. Auto-updated from check results. Support custom domains and branding. Show active incidents and 90-day uptime history.

**Core entities:** `status_pages`, `status_page_components`, `status_page_incidents`

**Key design decisions:**
- Status pages are served at `/status/:slug` — no auth required
- Custom domain: Host-header routing. Checker serves the correct page when the request hostname matches a registered domain. TLS is the user's responsibility.
- Component status is computed from linked checks: worst-case across all checks in the component
- Manual override: operator can force a component to any status
- 90-day uptime: computed from `check_results` metrics table (Part 6 dependency)
- Embeddable badge: SVG endpoint `/status/:slug/badge.svg`
- RSS feed: `/status/:slug/feed.rss` — subscriber notifications deferred
- No subscriber management in Phase 2 (see architectural principles above)

---

## Part 6: Metrics, Analytics & Reporting

See: `plans/phase2-part6-metrics.md`

**Summary:** Store check result history as time-series data. Compute uptime percentages and response time statistics. Expose a Prometheus `/metrics` endpoint. Show charts in the frontend.

**Core entities:** `check_results` (time-series), no additional service

**Key design decisions:**
- Use Postgres with optional TimescaleDB. If TimescaleDB is not available, fall back to plain Postgres with a time-based index. The schema is identical; only the hypertable creation differs. Checker detects TimescaleDB presence at startup.
- Retention: configurable via env var `METRICS_RETENTION_DAYS` (default 90). Background goroutine purges old rows daily.
- Downsampling: not in Phase 2. Store every result at full resolution for 90 days.
- Prometheus exporter: standard `/metrics` endpoint, compatible with existing Prometheus scrapers.
- SQLite: metrics not stored in SQLite mode (demo instances). `check_results` writes are no-ops when `DB_DRIVER=sqlite`. Status page uptime bars show "N/A" in demo.

---

## Execution Order

Phase 2 can largely be parallelised across parts, with one dependency:

1. **Part 6 first** (metrics storage) — Status pages need the `check_results` table for 90-day uptime bars. Start this immediately.
2. **Part 4 and Part 5 in parallel** once the metrics table exists.
3. **Part 5 custom domain** — can be done last within Part 5; it's independent of the rest.

Within each part, backend first, then frontend.

---

## DB Migrations (Phase 2)

| # | File | Description |
|---|------|-------------|
| 000010 | `check_results` | Time-series check result rows |
| 000011 | `incidents` | Incident table |
| 000012 | `incident_updates` | Incident timeline entries |
| 000013 | `incident_check_links` | Many-to-many checks ↔ incidents |
| 000014 | `status_pages` | Status page config |
| 000015 | `status_page_components` | Components within a page |
| 000016 | `status_page_incidents` | Incidents linked to a page |
| 000017 | `custom_domains` | Custom domain → page mapping |

Each has a corresponding `.down.sql`.

---

## Definition of Done for Phase 2

1. A check going unhealthy auto-creates an incident (when `auto_incident: true`)
2. Incident can be acknowledged, progressed through states, and resolved with a postmortem
3. A status page at `/status/my-page` shows component health, active incidents, and 90-day uptime bars
4. Custom domain correctly routes to its status page (Host-header match)
5. `/metrics` Prometheus endpoint returns check up/down gauges and response time histograms
6. Frontend shows response time sparklines on dashboard and full charts on check detail page
7. `go test ./...` passes
8. All migrations run cleanly up and down
9. SQLite mode degrades gracefully (no metrics stored, status page uptime shows N/A)
