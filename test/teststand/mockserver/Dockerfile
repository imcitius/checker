FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mockserver .

FROM alpine:3.19
COPY --from=builder /app/mockserver /usr/local/bin/mockserver
EXPOSE 8080 8081 8082 9001 2222 2223 2525 2526 50051 50052 8091 8092 8443 8444 5353/udp
ENTRYPOINT ["mockserver"]
