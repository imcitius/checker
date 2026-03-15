# Phase 2 — Part 4: Incident Management — LLD

**Depends on:** Nothing. Can be built in parallel with Part 5 and Part 6.

---

## 1. Concept & Scope

An incident is the canonical record of a service degradation. It is distinct from an alert (which is a notification event) and from a check result (which is a data point). Incidents have a lifecycle, a timeline, severity, and optionally a postmortem.

**What Checker handles:**
- Auto-creation when a check transitions to unhealthy (opt-in per check)
- Manual creation by an operator
- Lifecycle management: open → acknowledged → investigating → resolved
- Timeline: all events appended in order (status changes, comments, linked alerts)
- Multiple checks linked to one incident (e.g. "API down" links HTTP + DNS + SSL checks)
- Postmortem: a markdown document attached to a resolved incident
- P1–P4 severity, set manually or inherited from check severity

**What is deferred:**
- Incident-driven communication (posting updates to Slack/Teams during an incident — this is a future enhancement on top of the alert channel system)
- On-call schedule routing (Part 3 extension, Phase 3)
- SLA tracking per incident (Phase 3)

---

## 2. Data Model

### `incidents` table

```sql
-- migrations/000011_incidents.up.sql
CREATE TABLE incidents (
    id              BIGSERIAL PRIMARY KEY,
    uuid            TEXT UNIQUE NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    severity        TEXT NOT NULL DEFAULT 'P3',     -- P1, P2, P3, P4
    status          TEXT NOT NULL DEFAULT 'open',   -- open, acknowledged, investigating, resolved
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    resolved_at     TIMESTAMPTZ,
    created_by      TEXT NOT NULL DEFAULT 'system', -- 'system' or username
    postmortem      TEXT,                           -- markdown, nullable until resolved
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_status ON incidents (status);
CREATE INDEX idx_incidents_started_at ON incidents (started_at DESC);
```

### `incident_updates` table (timeline)

```sql
-- part of migrations/000012_incident_updates.up.sql
CREATE TABLE incident_updates (
    id           BIGSERIAL PRIMARY KEY,
    incident_uuid TEXT NOT NULL REFERENCES incidents(uuid) ON DELETE CASCADE,
    kind         TEXT NOT NULL,   -- 'status_change', 'comment', 'check_alert', 'check_recovery', 'severity_change'
    content      TEXT NOT NULL,   -- human-readable description or comment text
    metadata     JSONB,           -- structured data depending on kind
    author       TEXT NOT NULL DEFAULT 'system',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incident_updates_incident ON incident_updates (incident_uuid, created_at ASC);
```

**`metadata` examples by `kind`:**
- `status_change`: `{"from": "open", "to": "acknowledged"}`
- `check_alert`: `{"check_uuid": "...", "check_name": "API Health", "error": "connection refused"}`
- `check_recovery`: `{"check_uuid": "...", "check_name": "API Health"}`
- `severity_change`: `{"from": "P3", "to": "P1"}`
- `comment`: null (content is the comment text)

### `incident_check_links` table

```sql
-- part of migrations/000013_incident_check_links.up.sql
CREATE TABLE incident_check_links (
    incident_uuid TEXT NOT NULL REFERENCES incidents(uuid) ON DELETE CASCADE,
    check_uuid    TEXT NOT NULL,
    linked_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (incident_uuid, check_uuid)
);

CREATE INDEX idx_incident_check_links_check ON incident_check_links (check_uuid);
```

**Why no FK on `check_uuid`:** same rationale as `check_results` — incidents must survive check deletion.

### `check_definitions` column addition

```sql
-- part of migrations/000011_incidents.up.sql
ALTER TABLE check_definitions ADD COLUMN auto_incident BOOLEAN DEFAULT FALSE;
```

When `auto_incident = true`, the scheduler auto-creates an incident when the check transitions to unhealthy.

---

## 3. Incident Lifecycle

### State machine

```
         ┌──────────────────────────────┐
         │                              ▼
[open] ──► [acknowledged] ──► [investigating] ──► [resolved]
  │                                                    ▲
  └────────────────────────────────────────────────────┘
  (can skip stages, can jump directly to resolved)
```

**Rules:**
- Only forward transitions allowed. Cannot go from `resolved` back to `open`. If a resolved incident's check fails again, a **new incident** is created.
- `acknowledged_at` is set when entering `acknowledged` state. Never updated again.
- `resolved_at` is set when entering `resolved` state.
- Each transition appends a `status_change` row to `incident_updates`.

**Corner case — duplicate auto-creation:** if a check is already linked to an open incident and it fires again (e.g. `ReAlertInterval`), do not create a new incident. Check: before creating, query for open incidents linked to this `check_uuid`. If found, append a `check_alert` update to the existing incident instead.

**Corner case — check recovered but incident still open:** when a check transitions back to healthy, append a `check_recovery` update to any open incidents linked to that check. Do NOT auto-resolve the incident — an operator must resolve it. The recovery event is informational. Rationale: an incident might span multiple checks; one recovering doesn't mean the incident is over.

**Corner case — auto_incident race condition:** if two checks linked to the same incident both go unhealthy at the same time (parallel scheduler goroutines), both might try to create an incident simultaneously. Use a DB-level unique constraint or advisory lock. Simplest: try to insert, if duplicate UUID, retry with a new UUID (UUID collision is negligible). For "existing open incident" check, use `SELECT ... FOR UPDATE` or rely on the unique index on `(incident_uuid, check_uuid)` in the link table.

---

## 4. Repository Interface

```go
// Add to internal/db/repository.go:

// Incidents
CreateIncident(ctx context.Context, incident models.Incident) (string, error)
GetIncidentByUUID(ctx context.Context, uuid string) (models.Incident, error)
GetAllIncidents(ctx context.Context, filters models.IncidentFilters) ([]models.Incident, int, error)
UpdateIncidentStatus(ctx context.Context, uuid, status, author string) error
UpdateIncidentSeverity(ctx context.Context, uuid, severity, author string) error
UpdateIncidentPostmortem(ctx context.Context, uuid, postmortem string) error
ResolveIncident(ctx context.Context, uuid, author string) error

// Incident timeline
CreateIncidentUpdate(ctx context.Context, update models.IncidentUpdate) error
GetIncidentUpdates(ctx context.Context, incidentUUID string) ([]models.IncidentUpdate, error)

// Check links
LinkCheckToIncident(ctx context.Context, incidentUUID, checkUUID string) error
UnlinkCheckFromIncident(ctx context.Context, incidentUUID, checkUUID string) error
GetOpenIncidentForCheck(ctx context.Context, checkUUID string) (models.Incident, bool, error)
GetIncidentsForCheck(ctx context.Context, checkUUID string, limit int) ([]models.Incident, error)
```

---

## 5. Models

```go
// internal/models/incident.go

type Incident struct {
    ID             int64      `json:"id"`
    UUID           string     `json:"uuid"`
    Title          string     `json:"title"`
    Description    string     `json:"description,omitempty"`
    Severity       string     `json:"severity"`            // P1, P2, P3, P4
    Status         string     `json:"status"`              // open, acknowledged, investigating, resolved
    StartedAt      time.Time  `json:"started_at"`
    AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
    ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
    CreatedBy      string     `json:"created_by"`
    Postmortem     string     `json:"postmortem,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`

    // Populated on read (joins)
    LinkedChecks []string         `json:"linked_checks,omitempty"`
    Updates      []IncidentUpdate `json:"updates,omitempty"`
}

type IncidentUpdate struct {
    ID           int64          `json:"id"`
    IncidentUUID string         `json:"incident_uuid"`
    Kind         string         `json:"kind"`
    Content      string         `json:"content"`
    Metadata     map[string]any `json:"metadata,omitempty"`
    Author       string         `json:"author"`
    CreatedAt    time.Time      `json:"created_at"`
}

type IncidentFilters struct {
    Status   string // "", "open", "acknowledged", "investigating", "resolved"
    Severity string // "", "P1", "P2", "P3", "P4"
    Limit    int
    Offset   int
}
```

---

## 6. Scheduler Integration

In `internal/scheduler/scheduler.go`, in the section that handles state transitions (healthy → unhealthy), after sending alerts:

```go
if checkDef.AutoIncident && isNewIncident {
    // Check for existing open incident for this check
    existingIncident, found, err := repo.GetOpenIncidentForCheck(ctx, checkDef.UUID)
    if err != nil {
        logrus.Warnf("Failed to check existing incidents for %s: %v", checkDef.UUID, err)
    } else if found {
        // Append alert event to existing incident
        repo.CreateIncidentUpdate(ctx, models.IncidentUpdate{
            IncidentUUID: existingIncident.UUID,
            Kind:         "check_alert",
            Content:      fmt.Sprintf("Check %q failed again: %s", checkDef.Name, lastMessage),
            Metadata:     map[string]any{"check_uuid": checkDef.UUID, "check_name": checkDef.Name, "error": lastMessage},
            Author:       "system",
        })
    } else {
        // Auto-create incident
        incident := models.Incident{
            UUID:      uuid.New().String(),
            Title:     fmt.Sprintf("%s is DOWN", checkDef.Name),
            Severity:  mapCheckSeverityToIncident(checkDef.Severity),
            Status:    "open",
            StartedAt: time.Now(),
            CreatedBy: "system",
        }
        incidentUUID, err := repo.CreateIncident(ctx, incident)
        if err != nil {
            logrus.Warnf("Failed to create incident for %s: %v", checkDef.UUID, err)
        } else {
            repo.LinkCheckToIncident(ctx, incidentUUID, checkDef.UUID)
            repo.CreateIncidentUpdate(ctx, models.IncidentUpdate{
                IncidentUUID: incidentUUID,
                Kind:         "check_alert",
                Content:      fmt.Sprintf("Check %q triggered incident: %s", checkDef.Name, lastMessage),
                Metadata:     map[string]any{"check_uuid": checkDef.UUID, "check_name": checkDef.Name, "error": lastMessage},
                Author:       "system",
            })
        }
    }
}

// On recovery:
if wasUnhealthy && isNowHealthy {
    existingIncident, found, _ := repo.GetOpenIncidentForCheck(ctx, checkDef.UUID)
    if found {
        repo.CreateIncidentUpdate(ctx, models.IncidentUpdate{
            IncidentUUID: existingIncident.UUID,
            Kind:         "check_recovery",
            Content:      fmt.Sprintf("Check %q has recovered", checkDef.Name),
            Metadata:     map[string]any{"check_uuid": checkDef.UUID, "check_name": checkDef.Name},
            Author:       "system",
        })
    }
}
```

**`mapCheckSeverityToIncident`:**
- `"critical"` → `"P1"`
- `"warning"` → `"P3"`
- `"info"` → `"P4"`
- default → `"P3"`

---

## 7. REST API

```
GET    /api/incidents                        List incidents (filterable: status, severity)
POST   /api/incidents                        Create incident manually
GET    /api/incidents/:uuid                  Get incident detail (includes updates + linked checks)
PUT    /api/incidents/:uuid/status           Update status { "status": "acknowledged", "author": "ilya" }
PUT    /api/incidents/:uuid/severity         Update severity { "severity": "P1" }
PUT    /api/incidents/:uuid/postmortem       Set/update postmortem { "postmortem": "## What happened\n..." }
POST   /api/incidents/:uuid/checks           Link check { "check_uuid": "..." }
DELETE /api/incidents/:uuid/checks/:check_uuid  Unlink check
POST   /api/incidents/:uuid/updates          Add comment { "content": "Investigating...", "author": "ilya" }
GET    /api/incidents/:uuid/updates          List all timeline entries
GET    /api/checks/:uuid/incidents           All incidents for a check (for check detail page)
```

**Corner cases:**
- `PUT /status` with invalid status → 400 with list of valid statuses
- `PUT /status` with backward transition (e.g. `open` → `resolved` is valid, but `resolved` → `open` is not) → 409 with explanation
- `PUT /postmortem` on non-resolved incident → 422: "Postmortems can only be added to resolved incidents." (Enforce this — postmortems written on open incidents get lost in the noise)
- `DELETE /api/incidents/:uuid` — do not support deletion. Incidents are immutable history. Instead support a `void` or `false_alarm` status for noise. Actually: add `false_alarm: bool` field that hides the incident from default views but preserves data.

---

## 8. WebSocket Integration

When an incident is created, updated, or resolved, push an event to connected WebSocket clients so the frontend updates in real-time:

```json
{
  "type": "incident_update",
  "incident_uuid": "...",
  "status": "acknowledged",
  "title": "API is DOWN",
  "severity": "P1"
}
```

Add `incident` event type to the existing WebSocket broadcaster in `internal/web/server.go`.

---

## 9. Frontend

### New page: `frontend/src/pages/Incidents.tsx`

- Table: Title | Severity badge | Status badge | Started | Duration | Linked checks count
- Filterable by status and severity
- Click → detail view

### Incident detail

- Header: title, severity (editable), status (editable via buttons: Acknowledge / Investigating / Resolve)
- Linked checks section: list with current health status; link/unlink buttons
- Timeline: chronological list of updates with icons per kind
  - 🔴 check_alert
  - 🟢 check_recovery
  - 🔄 status_change
  - 💬 comment
  - ⚠️ severity_change
- Add comment box (textarea + submit)
- Postmortem section: only shows when status = resolved. Markdown editor (use existing Monaco editor if present, or a simple textarea).

### Dashboard integration

- "Active incidents" count badge in top nav / status bar
- If any P1/P2 incident is open, show a banner at the top of the dashboard

### Check detail integration

- Each check's detail view shows its incident history (last 5 incidents with links)

---

## 10. Files to Create / Modify

| File | Action |
|------|--------|
| `migrations/000011_incidents.up.sql` | Create |
| `migrations/000011_incidents.down.sql` | Create |
| `migrations/000012_incident_updates.up.sql` | Create |
| `migrations/000012_incident_updates.down.sql` | Create |
| `migrations/000013_incident_check_links.up.sql` | Create |
| `migrations/000013_incident_check_links.down.sql` | Create |
| `internal/models/incident.go` | Create |
| `internal/db/repository.go` | Add 11 new interface methods |
| `internal/db/postgres.go` | Implement |
| `internal/db/sqlite.go` | Implement (full, not no-op — incidents work in SQLite too) |
| `internal/scheduler/scheduler.go` | Auto-create/update incidents on state transitions |
| `internal/web/incident_handlers.go` | New file: all incident REST handlers |
| `internal/web/server.go` | Wire routes, add WS incident event |
| `frontend/src/pages/Incidents.tsx` | New page |
| `frontend/src/components/IncidentDetail.tsx` | New component |
| `frontend/src/App.tsx` | Add /incidents route |
| `frontend/src/components/TopBar.tsx` | Add Incidents nav link, active incident badge |

---

## 11. Acceptance Criteria

1. A check with `auto_incident: true` creates an incident on first failure
2. Second failure of same check appends to existing open incident, does not create a new one
3. Recovery of check appends `check_recovery` update to open incident, does not auto-resolve it
4. Operator can manually progress: open → acknowledged → investigating → resolved
5. Cannot transition backward (resolved → open returns 409)
6. Postmortem can only be set on resolved incidents
7. Manual incident creation works with arbitrary linked checks
8. Timeline shows all events in chronological order
9. WebSocket pushes incident updates to connected clients
10. `go test ./internal/...` passes
