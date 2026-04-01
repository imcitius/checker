package checks

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCHealthCheck represents a gRPC health check using the standard
// grpc.health.v1.Health/Check protocol.
type GRPCHealthCheck struct {
	Host    string
	UseTLS  bool
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the gRPC health check by dialing the host, calling the
// Health/Check RPC, and verifying the response status is SERVING.
func (check *GRPCHealthCheck) Run() (time.Duration, error) {
	start := time.Now()

	if check.Host == "" {
		return time.Since(start), errors.New(ErrEmptyHost)
	}

	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "grpc_health")
	}

	// Build transport credentials
	var creds grpc.DialOption
	if check.UseTLS {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.NewClient(check.Host, creds)
	if err != nil {
		check.Logger.WithError(err).Debugf("gRPC dial failed for %s", check.Host)
		return time.Since(start), fmt.Errorf("grpc dial error: %w", err)
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		check.Logger.WithError(err).Debugf("gRPC health check RPC failed for %s", check.Host)
		return time.Since(start), fmt.Errorf("grpc health check error: %w", err)
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return time.Since(start), fmt.Errorf("grpc health check: status is %s, expected SERVING", resp.GetStatus().String())
	}

	return time.Since(start), nil
}
