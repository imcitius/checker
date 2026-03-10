# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with optimized flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o checker \
    .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/checker /app/checker

# Copy migrations directory for runtime migration (created by migration tasks)
# Using a wildcard so the build doesn't fail if the directory doesn't exist yet
COPY --from=builder /build/migration[s] /app/migrations/

# Copy config.yaml as default config (can be overridden via volume mount or env vars)
# Using a wildcard so the build doesn't fail if the file doesn't exist yet
COPY --from=builder /build/config.yam[l] /app/

EXPOSE 8080

ENTRYPOINT ["/app/checker"]
