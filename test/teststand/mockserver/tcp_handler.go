// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"log"
	"net"
)

func startTCPPass(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("TCP pass listen failed: %v", err)
	}
	log.Printf("TCP pass listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP pass accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			c.Write([]byte("HELLO\n"))
		}(conn)
	}
}
