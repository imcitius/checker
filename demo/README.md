# Checker Demo — Railway Deployment

This branch (`demo`) auto-deploys to Railway as a zero-dependency demo instance using SQLite instead of PostgreSQL.

## What This Branch Is

The `demo` branch is a long-lived branch that mirrors `dev` with demo-specific configuration. It runs Checker with an embedded SQLite database, requiring no external services. On first start with an empty database, demo checks are automatically seeded from `demo/seed.yaml`.

## Railway Deployment Steps

### 1. Create a Railway Service

1. Go to [railway.app](https://railway.app) and create a new project
2. Add a new service from your GitHub repo
3. Set the **Source Branch** to `demo`

### 2. Set Environment Variables

In the Railway service settings, add the following environment variables:

| Variable | Value | Description |
|----------|-------|-------------|
| `DB_DRIVER` | `sqlite` | Use SQLite instead of PostgreSQL |
| `DB_DSN` | `/data/checker.db` | SQLite database file path |
| `DEMO_MODE` | `true` | Enables demo banner/badge in the UI (when implemented) |
| `RAILWAY_CONFIG_FILE` | `railway.demo.toml` | Use the demo-specific Railway config |

> **Note:** `PORT` is set automatically by Railway (defaults to 8080).

### 3. (Optional) Create a Persistent Volume

For SQLite data to persist across redeploys, create a Railway volume:

1. In the Railway dashboard, go to your service settings
2. Under **Volumes**, click **Add Volume**
3. Set the mount path to `/data`
4. Railway will create a persistent volume mounted at `/data`

Without a volume, the SQLite database resets on every redeploy. The app will re-seed demo checks automatically on startup when the database is empty, so this is acceptable for demo purposes.

## How to Update Demo Checks

1. Edit `demo/seed.yaml` with the desired check definitions
2. Commit and push to the `demo` branch
3. Railway will automatically redeploy

The seed file is embedded into the binary at build time. On startup, if the SQLite database is empty, the app loads checks from the seed file.

## How to Reset Demo Data

There are two ways to reset demo data:

- **Delete the Railway volume** — Remove the volume from the Railway dashboard. On the next deploy, the app starts with a fresh SQLite database and re-seeds from `demo/seed.yaml`.
- **Redeploy without a volume** — If no volume is mounted, every deploy starts fresh automatically.

## Docker Build

The Dockerfile supports CGO (required for go-sqlite3) with `gcc` and `musl-dev` installed in the build stage. The final runtime image remains small — only the compiled binary is copied to the Alpine runtime stage.

To build locally:

```bash
docker build --build-arg GIT_SHA=$(git rev-parse HEAD) -t checker-demo .
```

To run locally with SQLite:

```bash
docker run -p 8080:8080 \
  -e DB_DRIVER=sqlite \
  -e DB_DSN=/data/checker.db \
  -v checker-data:/data \
  checker-demo
```

## Architecture Notes

- **Single binary** — The Go binary embeds the React frontend (via `//go:embed`) and serves everything from one process
- **No external dependencies** — SQLite is compiled into the binary via CGO; no PostgreSQL or other services needed
- **Auto-seed** — On first startup with an empty SQLite database, demo checks are imported from `demo/seed.yaml`
- **Health check** — The `/healthz` endpoint is used by Railway to verify the service is running
