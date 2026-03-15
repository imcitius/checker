# Phase 2 — Part 5: Status Pages — LLD

**Depends on:** Part 6 (Metrics) for 90-day uptime bars. All other features are independent.

---

## 1. Concept & Scope

A status page is a public-facing, auth-free web page that shows the health of selected checks grouped into named service components. It is the face Checker presents to customers and stakeholders during incidents.

**What Checker handles:**
- Multiple status pages per instance (different pages for different audiences)
- Components: named groups of checks. Component status = worst-case of its checks.
- Manual override: operator can force a component to any status regardless of check results.
- Active incidents displayed on the page (sourced from Part 4)
- Scheduled maintenance windows shown on the page
- 90-day uptime history bars per component (sourced from Part 6)
- Custom domain support via ACME/Let's Encrypt
- Embeddable SVG badge per page
- RSS feed per page (for lightweight "subscribe via RSS reader" — subscriber email/SMS deferred)

**What is deferred:**
- Subscriber email/SMS opt-in and dispatch (future `checker-notify` service)
- Custom CSS/branding beyond basic logo and color (Phase 3+)
- CDN / edge caching (Phase 3+)

---

## 2. Data Model

### `status_pages` table

```sql
-- migrations/000014_status_pages.up.sql
CREATE TABLE status_pages (
    id           BIGSERIAL PRIMARY KEY,
    uuid         TEXT UNIQUE NOT NULL,
    slug         TEXT UNIQUE NOT NULL,      -- URL segment: /status/:slug
    title        TEXT NOT NULL,
    description  TEXT,
    logo_url     TEXT,
    brand_color  TEXT DEFAULT '#2563EB',   -- hex color for header accent
    is_public    BOOLEAN NOT NULL DEFAULT TRUE,
    custom_domain TEXT UNIQUE,             -- e.g. "status.example.com", nullable
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_status_pages_slug ON status_pages (slug);
CREATE UNIQUE INDEX idx_status_pages_custom_domain ON status_pages (custom_domain)
    WHERE custom_domain IS NOT NULL;
```

**Corner case — slug validation:** slugs must be URL-safe: lowercase alphanumeric + hyphens, 3–60 chars. Enforce in API validation. Reserved slugs: `api`, `admin`, `auth`, `static`, `assets`, `healthz`, `metrics`, `ws` — reject these.

### `status_page_components` table

```sql
-- migrations/000015_status_page_components.up.sql
CREATE TABLE status_page_components (
    id              BIGSERIAL PRIMARY KEY,
    uuid            TEXT UNIQUE NOT NULL,
    page_uuid       TEXT NOT NULL REFERENCES status_pages(uuid) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    display_order   INTEGER NOT NULL DEFAULT 0,
    status_override TEXT,          -- NULL = auto, or 'operational'/'degraded'/'partial_outage'/'major_outage'/'maintenance'
    override_until  TIMESTAMPTZ,   -- auto-clear override at this time (NULL = permanent until manually cleared)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_status_page_components_page ON status_page_components (page_uuid, display_order ASC);

-- Join table: which checks belong to this component
CREATE TABLE status_page_component_checks (
    component_uuid TEXT NOT NULL REFERENCES status_page_components(uuid) ON DELETE CASCADE,
    check_uuid     TEXT NOT NULL,
    PRIMARY KEY (component_uuid, check_uuid)
);
```

**Component status values:**
- `operational` — all checks healthy
- `degraded_performance` — checks healthy but response time > 2x average (detected from metrics)
- `partial_outage` — some checks down, not all
- `major_outage` — all checks down
- `maintenance` — maintenance window active on at least one linked check (or manual override)
- `unknown` — no checks linked, or no data

**Corner case — no checks linked:** a component with zero linked checks shows `unknown` status, not `operational`. Empty components are valid (placeholder for a future service).

**Corner case — override_until expiry:** the API sets `override_until`. The status page renderer checks if `override_until < NOW()` and ignores the override if expired. No background job needed — lazy evaluation on read.

### `status_page_incidents` table

```sql
-- migrations/000016_status_page_incidents.up.sql
-- Links incidents to status pages (for display)
CREATE TABLE status_page_incidents (
    page_uuid     TEXT NOT NULL REFERENCES status_pages(uuid) ON DELETE CASCADE,
    incident_uuid TEXT NOT NULL,          -- no FK, incidents can exist without this link
    display_on_page BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (page_uuid, incident_uuid)
);
```

**How incidents appear on status pages:** when an incident is created (auto or manual), the system links it to all status pages that have at least one component containing a check linked to the incident. This is done automatically in the incident creation code. Operators can also manually add/remove incidents from pages.

### `custom_domains` table

```sql
-- migrations/000017_custom_domains.up.sql
CREATE TABLE custom_domains (
    domain       TEXT PRIMARY KEY,
    page_uuid    TEXT NOT NULL REFERENCES status_pages(uuid) ON DELETE CASCADE,
    verified_at  TIMESTAMPTZ,          -- NULL until DNS verification passes
    cert_obtained_at TIMESTAMPTZ,      -- NULL until ACME cert issued
    acme_account_key TEXT,             -- ACME account private key, encrypted at rest
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Note: `status_pages.custom_domain` is a convenience column for fast lookup. The canonical source of truth is `custom_domains`.

---

## 3. Routing Architecture

### Standard routes (no custom domain)

```
GET /status/:slug              → status page HTML (server-rendered or SPA route)
GET /status/:slug/badge.svg    → embeddable SVG badge
GET /status/:slug/feed.rss     → RSS feed
GET /api/status/:slug          → JSON: full page data
GET /api/status/:slug/uptime   → JSON: uptime stats per component
```

These routes are unprotected — no auth middleware.

### Custom domain routing

When a request arrives with `Host: status.example.com`, Gin middleware checks `custom_domains` table. If matched, serves the same page as `/status/:slug` for the associated page.

```go
// In server.go, before route registration:
router.Use(func(c *gin.Context) {
    host := c.Request.Host
    // Strip port if present
    if idx := strings.LastIndex(host, ":"); idx != -1 {
        host = host[:idx]
    }
    // Check if this host is a registered custom domain
    if pageUUID, ok := customDomainCache.Get(host); ok {
        c.Set("status_page_override_uuid", pageUUID)
    }
    c.Next()
})
```

The `customDomainCache` is an in-memory map refreshed every 60 seconds from the DB. Cache TTL is acceptable — custom domains change rarely.

**Corner case — custom domain not yet verified:** the page is not accessible via custom domain until DNS verification passes. Show a "domain pending verification" message instead.

**Corner case — custom domain conflicts with existing routes:** the `Host`-based routing happens before path routing. If someone registers `checker.example.com` as a custom domain, all paths under that host go to the status page, including `/api/*`. This is intentional — a custom domain is dedicated to a status page. Document this.

---

## 4. Status Page Rendering

Status pages are served as server-rendered HTML for SEO and performance, with a React hydration layer for the active incident feed (real-time via WebSocket or polling).

**Option A (preferred):** Go-rendered HTML template for the initial page, with a small embedded JS bundle for real-time updates (separate from the main SPA bundle). This keeps the page fast, SEO-friendly, and avoids shipping the full React app to anonymous status page visitors.

**Option B:** Serve the full SPA and add `/status/:slug` as an SPA route. Simpler to implement but ships the whole app to public visitors.

**Decision: Option A.** Status pages are public-facing and load time matters. A 50KB HTML page is better than a 2MB SPA for public status pages. Implementation: add a Go HTML template at `internal/web/templates/status_page.html`. The React SPA only handles the authenticated admin side.

### Status page content

```html
<!-- Header -->
<div class="header" style="border-color: {{.BrandColor}}">
  <img src="{{.LogoURL}}" />
  <h1>{{.Title}} Status</h1>
  <p class="overall-status">{{.OverallStatus}}</p>  <!-- "All Systems Operational" etc. -->
</div>

<!-- Active incidents (if any) -->
{{range .ActiveIncidents}}
<div class="incident incident-{{.Severity}}">
  <h3>{{.Title}}</h3>
  <p>{{.Status}} · Started {{.StartedAt | humanTime}}</p>
  {{range .RecentUpdates}}
  <p class="update">{{.CreatedAt | humanTime}} — {{.Content}}</p>
  {{end}}
</div>
{{end}}

<!-- Components -->
{{range .Components}}
<div class="component">
  <div class="component-header">
    <span class="name">{{.Name}}</span>
    <span class="status status-{{.Status}}">{{.StatusLabel}}</span>
  </div>
  <!-- 90-day uptime bar (90 divs, each coloured by day uptime%) -->
  <div class="uptime-bar">
    {{range .UptimeBuckets}}
    <div class="day-bucket {{.ColorClass}}" title="{{.Label}}"></div>
    {{end}}
  </div>
  <p class="uptime-summary">{{.UptimePct90d | printf "%.2f"}}% uptime (90 days)</p>
</div>
{{end}}
```

### Overall status computation

```go
func computeOverallStatus(components []ComponentStatus) string {
    hasMajorOutage := false
    hasPartialOutage := false
    hasDegraded := false
    hasMaintenance := false

    for _, c := range components {
        switch c.Status {
        case "major_outage":
            hasMajorOutage = true
        case "partial_outage":
            hasPartialOutage = true
        case "degraded_performance":
            hasDegraded = true
        case "maintenance":
            hasMaintenance = true
        }
    }

    switch {
    case hasMajorOutage:
        return "Major Service Outage"
    case hasPartialOutage:
        return "Partial Service Outage"
    case hasDegraded:
        return "Degraded Performance"
    case hasMaintenance:
        return "Maintenance In Progress"
    default:
        return "All Systems Operational"
    }
}
```

---

## 5. Embeddable Badge

`GET /status/:slug/badge.svg`

Returns an SVG badge similar to shields.io format:

```
[ My Service | ● Operational ]   (green)
[ My Service | ⚠ Degraded ]      (yellow)
[ My Service | ✖ Outage ]        (red)
```

Query params:
- `?style=flat|flat-square|for-the-badge` (default: flat)
- `?label=My+Service` (overrides default title)

No caching headers — always fresh. Cache at CDN if needed.

**Corner case — unknown page slug:** return a "status unknown" gray badge rather than 404. Users embed these in READMEs and a 404 breaks the image.

---

## 6. RSS Feed

`GET /status/:slug/feed.rss`

Standard RSS 2.0 feed. Each incident update is an item. Most recent 20 items.

```xml
<rss version="2.0">
  <channel>
    <title>My Service Status</title>
    <link>https://checker.example.com/status/my-service</link>
    <description>Status updates for My Service</description>
    <item>
      <title>[RESOLVED] API Degradation</title>
      <description>The issue has been resolved. All systems operational.</description>
      <pubDate>Sun, 15 Mar 2026 18:00:00 +0000</pubDate>
      <guid>https://checker.example.com/status/my-service#incident-abc123</guid>
    </item>
  </channel>
</rss>
```

---

## 7. Custom Domain & ACME

### DNS verification flow

1. Operator adds custom domain in settings UI
2. API creates `custom_domains` row with `verified_at = NULL`
3. UI shows: "Add a CNAME record: `status.example.com → checker.yourdomain.com`"
4. Background goroutine (runs every 5 minutes) checks DNS for each unverified domain:
   ```go
   addrs, err := net.LookupCNAME(domain)
   if err == nil && strings.HasSuffix(addrs, checkerHostname+".") {
       // Mark verified
       repo.VerifyCustomDomain(ctx, domain)
       // Trigger ACME cert acquisition
       go acquireACMECert(ctx, domain)
   }
   ```
5. Once verified, ACME cert is obtained via `autocert.Manager`
6. `custom_domains.cert_obtained_at` is updated
7. Custom domain starts working

### ACME implementation

Use `golang.org/x/crypto/acme/autocert`:

```go
certManager := autocert.Manager{
    Prompt:     autocert.AcceptTOS,
    HostPolicy: func(ctx context.Context, host string) error {
        // Allow only hosts in custom_domains table with verified_at != NULL
        if repo.IsVerifiedCustomDomain(ctx, host) {
            return nil
        }
        return fmt.Errorf("host %q not allowed", host)
    },
    Cache: autocert.DirCache("/data/acme-certs"), // persistent directory
}
```

**Corner case — HTTP-01 challenge:** Let's Encrypt requires serving `/.well-known/acme-challenge/` over HTTP port 80. Checker must listen on port 80 (or have a redirect) even if the primary service is on 443. Add an HTTP listener that handles ACME challenges and redirects all other traffic to HTTPS. This requires the service to be accessible on port 80 externally.

**Corner case — Railway/Docker deployment:** Railway terminates TLS at the proxy level. Custom domain ACME won't work behind Railway's proxy unless the user configures Railway to pass through raw TCP (not straightforward). Document: custom domains with auto-TLS work in self-hosted deployments. For Railway/hosted deployments, the user configures TLS at the proxy level and sets `CUSTOM_DOMAIN_TLS=external`.

**Corner case — cert expiry:** `autocert` handles renewal automatically. But if the domain's CNAME is removed, renewal fails silently. Add monitoring: if `cert_obtained_at` > 80 days ago and renewal hasn't happened, send a warning to the operator.

**Corner case — `acme_account_key` storage:** this is a private key. Store encrypted using a `ACME_KEY_ENCRYPTION_KEY` env var (AES-256-GCM). If the env var is not set, ACME is disabled and a warning is logged.

---

## 8. REST API (Admin)

```
GET    /api/status-pages                           List all pages
POST   /api/status-pages                           Create page
GET    /api/status-pages/:uuid                     Get page config
PUT    /api/status-pages/:uuid                     Update page config
DELETE /api/status-pages/:uuid                     Delete page

GET    /api/status-pages/:uuid/components          List components
POST   /api/status-pages/:uuid/components          Add component
PUT    /api/status-pages/:uuid/components/:cuuid   Update component (name, order, override)
DELETE /api/status-pages/:uuid/components/:cuuid   Remove component
POST   /api/status-pages/:uuid/components/:cuuid/checks  Link check { "check_uuid": "..." }
DELETE /api/status-pages/:uuid/components/:cuuid/checks/:check_uuid  Unlink check

GET    /api/status-pages/:uuid/incidents           List linked incidents
POST   /api/status-pages/:uuid/incidents           Manually link incident
DELETE /api/status-pages/:uuid/incidents/:iuuid    Unlink incident

POST   /api/status-pages/:uuid/domain              Set custom domain { "domain": "status.example.com" }
DELETE /api/status-pages/:uuid/domain              Remove custom domain
GET    /api/status-pages/:uuid/domain/status       Domain verification + cert status
```

---

## 9. Repository Interface

```go
// Add to internal/db/repository.go:

// Status pages
CreateStatusPage(ctx context.Context, page models.StatusPage) (string, error)
GetStatusPageByUUID(ctx context.Context, uuid string) (models.StatusPage, error)
GetStatusPageBySlug(ctx context.Context, slug string) (models.StatusPage, error)
GetStatusPageByCustomDomain(ctx context.Context, domain string) (models.StatusPage, error)
GetAllStatusPages(ctx context.Context) ([]models.StatusPage, error)
UpdateStatusPage(ctx context.Context, page models.StatusPage) error
DeleteStatusPage(ctx context.Context, uuid string) error

// Components
CreateStatusPageComponent(ctx context.Context, comp models.StatusPageComponent) (string, error)
GetStatusPageComponents(ctx context.Context, pageUUID string) ([]models.StatusPageComponent, error)
UpdateStatusPageComponent(ctx context.Context, comp models.StatusPageComponent) error
DeleteStatusPageComponent(ctx context.Context, uuid string) error
LinkCheckToComponent(ctx context.Context, componentUUID, checkUUID string) error
UnlinkCheckFromComponent(ctx context.Context, componentUUID, checkUUID string) error
GetComponentChecks(ctx context.Context, componentUUID string) ([]string, error) // returns check UUIDs

// Custom domains
SetCustomDomain(ctx context.Context, pageUUID, domain string) error
VerifyCustomDomain(ctx context.Context, domain string) error
GetUnverifiedCustomDomains(ctx context.Context) ([]models.CustomDomain, error)
IsVerifiedCustomDomain(ctx context.Context, domain string) bool
RemoveCustomDomain(ctx context.Context, domain string) error

// Status page rendering (read-optimised, used by the public page renderer)
GetStatusPageRenderData(ctx context.Context, pageUUID string) (models.StatusPageRenderData, error)
```

`GetStatusPageRenderData` is a single call that joins pages + components + checks + recent incidents + recent metrics. It is the hot path for public page renders and must be fast (< 50ms). Index accordingly.

---

## 10. Frontend (Admin)

### New page: `frontend/src/pages/StatusPages.tsx`

- List of status pages with links and overall status
- "Create new page" button
- Click → page editor

### Page editor

- Tabs: Components | Incidents | Domain | Settings
- **Components tab:** drag-to-reorder list of components. Each component expandable to show linked checks. Add/remove checks. Set status override.
- **Domain tab:** custom domain input, DNS instructions, verification status, cert status.
- **Settings tab:** title, description, logo URL, brand color, slug (read-only after creation).

### Preview button

"Open public page" button that opens `/status/:slug` in a new tab.

---

## 11. Files to Create / Modify

| File | Action |
|------|--------|
| `migrations/000014_status_pages.up.sql` | Create |
| `migrations/000015_status_page_components.up.sql` | Create |
| `migrations/000016_status_page_incidents.up.sql` | Create |
| `migrations/000017_custom_domains.up.sql` | Create |
| + all `.down.sql` counterparts | Create |
| `internal/models/status_page.go` | Create |
| `internal/db/repository.go` | Add ~15 new methods |
| `internal/db/postgres.go` | Implement |
| `internal/db/sqlite.go` | Implement (status pages work in SQLite, just no uptime history) |
| `internal/web/status_page_handlers.go` | New: admin REST API |
| `internal/web/status_page_public.go` | New: public page renderer (Go HTML template) |
| `internal/web/templates/status_page.html` | New: server-rendered status page template |
| `internal/web/templates/status_badge.svg` | New: badge template |
| `internal/web/server.go` | Wire routes (public + admin), add custom domain middleware |
| `internal/scheduler/acme.go` | New: ACME cert acquisition + renewal |
| `internal/scheduler/domain_verifier.go` | New: background DNS verification goroutine |
| `frontend/src/pages/StatusPages.tsx` | New admin page |
| `frontend/src/components/StatusPageEditor.tsx` | New component |
| `frontend/src/App.tsx` | Add /status-pages route |
| `frontend/src/components/TopBar.tsx` | Add Status Pages nav link |
| `go.mod` | Add `golang.org/x/crypto/acme/autocert` if not present |

---

## 12. Acceptance Criteria

1. Create a status page at `/status/my-service` — publicly accessible without auth
2. Components show correct status: operational / degraded / partial / major / maintenance
3. Manual override sets component status, respects `override_until` expiry
4. Active incidents from Part 4 appear on the page automatically when linked checks fail
5. 90-day uptime bars render with correct colors (requires Part 6)
6. SQLite mode: status pages work, uptime bars show "N/A"
7. Embeddable badge returns valid SVG with correct status color
8. RSS feed returns valid RSS 2.0 with recent incident updates
9. Custom domain: CNAME → verification → ACME cert → page accessible at custom domain (self-hosted only)
10. Reserved slugs rejected at creation time
11. `go test ./...` passes
