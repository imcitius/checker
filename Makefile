.PHONY: build frontend build-frontend build-go build-edge docker clean dev-frontend dev-go install-hooks setup

# Build everything: frontend + Go binary
build: frontend build-go

# Build the React frontend, copy to internal/web/spa/ for Go embed
frontend:
	cd frontend && npm ci && VITE_GIT_SHA=$$(git rev-parse HEAD 2>/dev/null || echo "unknown") npm run build
	rm -rf internal/web/spa
	cp -r frontend/dist internal/web/spa
	git rev-parse HEAD > internal/web/spa/.version 2>/dev/null || echo "unknown" > internal/web/spa/.version

# Alias for backward compatibility
build-frontend: frontend

# Build the Go binary (expects internal/web/spa/ to exist)
build-go:
	go build -ldflags="-s -w -X main.Version=$$(git rev-parse HEAD 2>/dev/null || echo unknown) -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o checker ./cmd/app

# Build the edge checker binary (no frontend needed)
build-edge:
	go build -o bin/checker-edge ./cmd/edge

# Build Docker image
docker:
	docker build -t checker .

# Clean build artifacts
clean:
	rm -rf frontend/dist internal/web/spa checker app

# Run frontend dev server (proxies API to Go backend on :8080)
dev-frontend:
	cd frontend && npm run dev

# Run Go backend
dev-go:
	go run ./cmd/app

# Install git hooks (pre-commit: auto-rebuild embedded SPA on frontend changes)
install-hooks:
	@mkdir -p $$(git rev-parse --git-dir)/hooks
	@cp dev/hooks/pre-commit $$(git rev-parse --git-dir)/hooks/pre-commit
	@chmod +x $$(git rev-parse --git-dir)/hooks/pre-commit
	@echo "Git hooks installed."

# First-time project setup: install dependencies and hooks
setup: install-hooks
	cd frontend && npm install
