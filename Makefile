.PHONY: build build-frontend build-go clean dev install-hooks setup

# Build everything: frontend + Go binary
build: build-frontend build-go

# Build the React frontend and copy to internal/web/spa/
build-frontend:
	cd frontend && VITE_GIT_SHA=$$(git rev-parse HEAD) npm run build
	rm -rf internal/web/spa
	cp -r frontend/dist internal/web/spa
	git rev-parse HEAD > internal/web/spa/.version

# Build the Go binary (expects internal/web/spa/ to exist)
build-go:
	go build -ldflags="-s -w -X main.Version=$$(git rev-parse HEAD) -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o app ./cmd/app

# Clean build artifacts
clean:
	rm -rf frontend/dist internal/web/spa app

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
