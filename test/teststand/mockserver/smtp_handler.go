package main

import (
	"bufio"
	"log"
	"net"
	"strings"
)

func startSMTPPass(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("SMTP pass listen failed: %v", err)
	}
	log.Printf("SMTP pass listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("SMTP pass accept error: %v", err)
			continue
		}
		go handleSMTP(conn)
	}
}

func handleSMTP(conn net.Conn) {
	defer conn.Close()
	conn.Write([]byte("220 mock ESMTP\r\n"))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			conn.Write([]byte("250 OK\r\n"))
		case strings.HasPrefix(cmd, "QUIT"):
			conn.Write([]byte("221 Bye\r\n"))
			return
		default:
			conn.Write([]byte("250 OK\r\n"))
		}
	}
}

func startSMTPFail(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("SMTP fail listen failed: %v", err)
	}
	log.Printf("SMTP fail listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("SMTP fail accept error: %v", err)
			continue
		}
		// Immediately close - simulates rejection
		conn.Close()
	}
}
