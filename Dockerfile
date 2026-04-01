# Build args for version injection
# Railway provides RAILWAY_GIT_COMMIT_SHA automatically at build time.
# For local/CI builds, pass --build-arg GIT_SHA=$(git rev-parse HEAD).
# Falls back to "unknown" if neither is set.
ARG GIT_SHA=unknown
ARG RAILWAY_GIT_COMMIT_SHA=unknown

# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
ARG GIT_SHA
ARG RAILWAY_GIT_COMMIT_SHA
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN EFFECTIVE_SHA=$(if [ "${RAILWAY_GIT_COMMIT_SHA}" != "unknown" ]; then echo "${RAILWAY_GIT_COMMIT_SHA}"; else echo "${GIT_SHA}"; fi) && \
    VITE_GIT_SHA="${EFFECTIVE_SHA}" npm run build && \
    echo "${EFFECTIVE_SHA}" > dist/.version

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
ARG GIT_SHA
ARG RAILWAY_GIT_COMMIT_SHA
RUN apk add --no-cache git gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /frontend/dist ./internal/web/spa/
RUN EFFECTIVE_SHA=$(if [ "${RAILWAY_GIT_COMMIT_SHA}" != "unknown" ]; then echo "${RAILWAY_GIT_COMMIT_SHA}"; else echo "${GIT_SHA}"; fi) && \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${EFFECTIVE_SHA} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o checker ./cmd/app

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/checker .
EXPOSE 8080
ENTRYPOINT ["./checker"]
