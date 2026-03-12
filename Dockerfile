# Stage 0: Extract git SHA from repo (no build-arg needed)
FROM alpine:3.21 AS git-info
RUN apk add --no-cache git
WORKDIR /repo
COPY .git .git
RUN git rev-parse HEAD > /git-sha

# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
COPY --from=git-info /git-sha /tmp/git-sha
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN VITE_GIT_SHA=$(cat /tmp/git-sha) npm run build
# Write the SHA to a .version file so the Go backend can read it
RUN cp /tmp/git-sha dist/.version

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
COPY --from=git-info /git-sha /tmp/git-sha
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /frontend/dist ./internal/web/spa/
RUN GIT_SHA=$(cat /tmp/git-sha) && \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${GIT_SHA} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o checker ./cmd/app

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/checker .
COPY --from=builder /build/migrations ./migrations
EXPOSE 8080
ENTRYPOINT ["./checker"]
