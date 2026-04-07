package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func startGRPCPass(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("gRPC pass listen failed: %v", err)
	}
	srv := grpc.NewServer()
	hsrv := health.NewServer()
	hsrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hsrv)

	log.Printf("gRPC pass listening on %s", addr)
	if err := srv.Serve(ln); err != nil {
		log.Fatalf("gRPC pass serve failed: %v", err)
	}
}

func startGRPCFail(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("gRPC fail listen failed: %v", err)
	}
	srv := grpc.NewServer()
	hsrv := health.NewServer()
	hsrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	healthpb.RegisterHealthServer(srv, hsrv)

	log.Printf("gRPC fail listening on %s", addr)
	if err := srv.Serve(ln); err != nil {
		log.Fatalf("gRPC fail serve failed: %v", err)
	}
}
