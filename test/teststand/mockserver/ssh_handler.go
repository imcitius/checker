package main

import (
	"log"
	"net"
)

func startSSHPass(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("SSH pass listen failed: %v", err)
	}
	log.Printf("SSH pass listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("SSH pass accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			c.Write([]byte("SSH-2.0-MockSSH\r\n"))
		}(conn)
	}
}

func startSSHFail(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("SSH fail listen failed: %v", err)
	}
	log.Printf("SSH fail listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("SSH fail accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			c.Write([]byte{0xff, 0xfe, 0x00, 0x01, 0xde, 0xad})
		}(conn)
	}
}
