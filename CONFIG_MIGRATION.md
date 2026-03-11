# Config Migration Guide

## Problem

After migrating from MongoDB to PostgreSQL, the checker system changed how it loads checks:

- **Before (MongoDB)**: Checks were loaded directly from `config.yaml` at runtime
- **After (PostgreSQL)**: Checks must be imported from `config.yaml` into the database first

## Solution

Your check "Check google http" is now successfully imported into the database! ✅

### How to Import Checks from config.yaml

You have **3 options** to import checks:

#### Option 1: Using the Migration Tool (Recommended)

```bash
go run cmd/migrate-config/main.go
```

This standalone tool will:
- Load your `config.yaml`
- Connect to PostgreSQL
- Import all checks into the database
- Show you what was imported

#### Option 2: Using the Web API (when server is running)

```bash
# Start the checker application
go run cmd/app/main.go

# In another terminal, call the migration endpoint
curl -X POST http://localhost:8080/api/admin/migrate-config
```

#### Option 3: Directly via SQL (Advanced)

You can manually insert checks into the `check_definitions` table, but this is not recommended.

## What Was Fixed

The `ConvertConfigToCheckDefinitions` function in `internal/db/postgres.go` was incomplete. It now:

1. **Properly resolves duration hierarchy**: `check → healthcheck → project → defaults`
2. **Supports all check types**: HTTP, TCP, ICMP, Passive, MySQL, PostgreSQL
3. **Handles all configuration fields**: timeout, headers, SSL settings, etc.

## Your Current Checks

```
UUID: a8cfdc4e-c51c-59af-8b06-d331e0fe49c7
Name: Check google http
Project: testProject
Group: http
Type: http
Duration: 5s
Enabled: true
```

## Next Steps

You can now start the checker application and it will automatically run this check:

```bash
go run cmd/app/main.go
```

The scheduler will load the check from the database and execute it every 5 seconds.

## How the System Works

```
┌─────────────┐
│ config.yaml │
└──────┬──────┘
       │ (manual migration)
       ▼
┌─────────────────┐
│   PostgreSQL    │
│  check_defs DB  │
└──────┬──────────┘
       │ (automatic polling every 10s)
       ▼
┌─────────────┐
│  Scheduler  │
│   (Heap)    │
└─────────────┘
```

1. **Config file** defines checks statically
2. **Database** stores active checks
3. **Scheduler** polls database every 10 seconds and executes checks

## Adding New Checks

When you add a new check to `config.yaml`:

1. Add the check definition to the YAML file
2. Run the migration tool: `go run cmd/migrate-config/main.go`
3. The scheduler will pick it up within 10 seconds (no restart needed!)
