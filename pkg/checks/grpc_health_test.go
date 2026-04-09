// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// mockHealthServer implements the gRPC Health/Check service.
type mockHealthServer struct {
	healthpb.UnimplementedHealthServer
	status healthpb.HealthCheckResponse_ServingStatus
}

func (s *mockHealthServer) Check(_ context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: s.status}, nil
}

func startMockGRPCHealthServer(t *testing.T, status healthpb.HealthCheckResponse_ServingStatus) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, &mockHealthServer{status: status})

	go func() {
		if err := srv.Serve(ln); err != nil {
			// server stopped
		}
	}()

	return ln.Addr().String(), func() { srv.GracefulStop() }
}

func TestGRPCHealthCheck_Serving(t *testing.T) {
	addr, cleanup := startMockGRPCHealthServer(t, healthpb.HealthCheckResponse_SERVING)
	defer cleanup()

	check := &GRPCHealthCheck{
		Host:    addr,
		UseTLS:  false,
		Timeout: "5s",
	}

	dur, err := check.Run()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if dur <= 0 {
		t.Error("expected positive duration")
	}
}

func TestGRPCHealthCheck_NotServing(t *testing.T) {
	addr, cleanup := startMockGRPCHealthServer(t, healthpb.HealthCheckResponse_NOT_SERVING)
	defer cleanup()

	check := &GRPCHealthCheck{
		Host:    addr,
		UseTLS:  false,
		Timeout: "5s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for NOT_SERVING status")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestGRPCHealthCheck_EmptyHost(t *testing.T) {
	check := &GRPCHealthCheck{
		Host:    "",
		Timeout: "5s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestGRPCHealthCheck_InvalidTimeout(t *testing.T) {
	check := &GRPCHealthCheck{
		Host:    "127.0.0.1:50051",
		Timeout: "not-a-duration",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestGRPCHealthCheck_ConnectionRefused(t *testing.T) {
	check := &GRPCHealthCheck{
		Host:    "127.0.0.1:1",
		UseTLS:  false,
		Timeout: "1s",
	}

	_, err := check.Run()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
