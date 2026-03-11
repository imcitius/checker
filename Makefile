.PHONY: build build-frontend build-go clean dev

# Build everything: frontend + Go binary
build: build-frontend build-go

# Build the React frontend and copy to internal/web/spa/
build-frontend:
	cd frontend && npm run build
	rm -rf internal/web/spa
	cp -r frontend/dist internal/web/spa

# Build the Go binary (expects internal/web/spa/ to exist)
build-go:
	go build -o app ./cmd/app

# Clean build artifacts
clean:
	rm -rf frontend/dist internal/web/spa app

# Run frontend dev server (proxies API to Go backend on :8080)
dev-frontend:
	cd frontend && npm run dev

# Run Go backend
dev-go:
	go run ./cmd/app
