# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Checker is a distributed health-checking system for monitoring services, databases, and endpoints. It runs scheduled checks via an event-driven scheduler and sends alerts through multiple channels (Telegram, Slack, Discord, PagerDuty, OpsGenie, Teams, Email). It has a Go backend with an embedded React SPA frontend.

## Build & Dev Commands

```bash
make build              # Full build: frontend + Go binary
make build-go           # Go binary only (requires internal/web/spa/ to exist)
make frontend           # Build React frontend, copy to internal/web/spa/
make dev-frontend       # Frontend dev server (proxies API to :8080)
make dev-go             # Run Go backend with `go run`
make setup              # First-time setup: install git hooks + npm install
make docker             # Build Docker image
make clean              # Remove build artifacts
```

Run the binary: `./checker -config config.yaml` (use `--debug` for debug logging).

## Testing

```bash
go test ./...                                    # Unit tests
go test -race -count=1 ./...                     # CI-style (race detector, no cache)
go test ./internal/checks                        # Single package
go test ./internal/checks -run=^TestHTTP         # Single test pattern

# Integration tests (require real databases)
INTEGRATION_TESTS=true go test ./...
INTEGRATION_TESTS=true TEST_MYSQL_USERNAME=root TEST_MYSQL_PASSWORD=password \
  go test ./internal/checks -run=^TestMySQL
```

Linting: `go vet ./...`

## Architecture

**Go module:** `checker` (Go 1.25.0)

### Startup flow (`cmd/app/main.go`)

1. Load YAML config → 2. Init database (postgres/sqlite) → 3. Init Slack/Telegram clients → 4. Init auth manager → 5. Start scheduler goroutine → 6. Start web server goroutine → graceful shutdown on SIGINT/SIGTERM.

### Key packages under `internal/`

| Package | Purpose |
|---------|---------|
| `config` | YAML config loading and validation |
| `db` | `Repository` interface + PostgreSQL and SQLite implementations |
| `models` | Data models (`CheckDefinition`, `AlertEvent`, etc.) |
| `checks` | 20+ health check type implementations |
| `scheduler` | Event-driven heap scheduler, worker pool, alerter integrations |
| `alerts` | Alert channel implementations (Telegram, Slack, Discord, etc.) |
| `web` | Gin HTTP server, REST handlers, embedded React SPA, WebSocket |
| `auth` | OIDC, JWT, password authentication |
| `slack` | Slack App client integration |
| `telegram` | Telegram Bot client integration |

### Core interfaces

- **`db.Repository`** — database abstraction (70+ methods). Implemented by PostgreSQL and SQLite.
- **`checks.Checker`** — all health checks implement `Run() (time.Duration, error)`. Created via `CheckerFactory()` in `internal/checks`.
- **`alerts.Alerter`** — alert channels implement `SendAlert()` and `SendRecovery()`. Uses a registry pattern for pluggable channels.

### Adding a new check type

1. Create a new file in `internal/checks/` with a struct implementing the `Checker` interface.
2. Register it in `CheckerFactory()` (`internal/checks/factories.go`).
3. Add corresponding config struct and type case.

### Adding an alert channel

1. Implement the `Alerter` interface in `internal/alerts/`.
2. Register it in the alerter registry.

### Frontend

React 19 + TypeScript + Vite + Tailwind CSS + Radix UI. Source in `frontend/`, built output gets copied to `internal/web/spa/` and embedded into the Go binary.

### Database & Migrations

Migrations live in `migrations/` (golang-migrate). They run automatically on startup. PostgreSQL is the primary database; SQLite is for development/demo. SQLite auto-seeds from `demo/seed.yaml` when empty.

## Conventions

- Tests are colocated (`*_test.go` in the same package). Use `httptest.Server` for HTTP check tests.
- Logging via `logrus`.
- CLI via `github.com/urfave/cli/v2`.
- HTTP framework: Gin.
- Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`.
- Default branch is `dev`.

## Babysitter

The `.a5c/` directory contains babysitter orchestration config (gitignored). Project profile: `.a5c/project-profile.json`. Default methodology: ATDD/TDD. Preferred processes: `gsd/feature-implementation`, `gsd/bugfix`, `specializations/qa-testing-automation`.

### Related Projects

`checker-cloud` (closed-source SaaS layer) imports checker's `pkg/` packages. Changes to `pkg/` interfaces may need propagation to checker-cloud.

## Stoneforge

The `.stoneforge/` directory is a multi-agent orchestration workspace (task management, git worktrees). See `AGENTS.md` for details. Not part of the application runtime.
