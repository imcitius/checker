package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func runTCPCheck(c *Check, p *Project) error {
	var (
		errorMessage  string
		checkAttempts int = 3
		checkAttempt  int
	)

	if c.Attempts != 0 {
		checkAttempts = c.Attempts
	}
	//log.Panicf("%+v", c)

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	errorHeader := fmt.Sprintf("TCP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, address, c.uuID)

	fmt.Printf("tcp ping test: %s\n`", address)

	timeout, _ := time.ParseDuration(c.Timeout)

	//log.Panic(timeout)

	for checkAttempt < checkAttempts {
		//startTime := time.Now()
		conn, err := net.DialTimeout("tcp", address, timeout)
		//endTime := time.Now()

		if err == nil {
			defer conn.Close()
			//t := float64(endTime.Sub(startTime)) / float64(time.Millisecond)
			//log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
			return nil
		}

		errorMessage = errorHeader + fmt.Sprintf("connection to host %s failed: %v (attempt %d)\n", c.Host+":"+string(c.Port), err, checkAttempts)
		log.Printf(errorMessage)
		checkAttempt++
	}
	fmt.Println(errorMessage)
	return errors.New(errorMessage)

}
