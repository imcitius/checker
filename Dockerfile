# Global build arg — pass via: docker build --build-arg GIT_SHA=$(git rev-parse HEAD) .
ARG GIT_SHA=unknown

# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
ARG GIT_SHA
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
ENV VITE_GIT_SHA=${GIT_SHA}
RUN npm run build
# Write the SHA to a .version file so the Go backend can read it
RUN echo "${GIT_SHA}" > dist/.version

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
ARG GIT_SHA
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /frontend/dist ./internal/web/spa/
RUN CGO_ENABLED=0 GOOS=linux go build \
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
