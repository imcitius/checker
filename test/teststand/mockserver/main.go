package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting mock server...")

	// Health check on :8080
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Health check listening on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("health check server failed: %v", err)
		}
	}()

	// HTTP
	go startHTTPPass(":8081")
	go startHTTPFail(":8082")

	// TCP
	go startTCPPass(":9001")
	// TCP fail: intentionally not listening on 9002

	// DNS
	go startDNS(":5353")

	// SSH
	go startSSHPass(":2222")
	go startSSHFail(":2223")

	// SMTP
	go startSMTPPass(":2525")
	go startSMTPFail(":2526")

	// gRPC
	go startGRPCPass(":50051")
	go startGRPCFail(":50052")

	// WebSocket
	go startWebSocketPass(":8091")
	go startWebSocketFail(":8092")

	// TLS/SSL
	passCert, passErr := generateValidCert()
	failCert, failErr := generateExpiredCert()
	go startTLSServer(":8443", passCert, passErr, "TLS-pass")
	go startTLSServer(":8444", failCert, failErr, "TLS-fail")

	log.Println("All handlers started. Waiting for shutdown signal...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down.")
}
