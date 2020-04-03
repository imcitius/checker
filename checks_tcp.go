package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func runTCPCheck(c *Check, p *Project) error {
	var (
		errorHeader, errorMessage string
		checkAttempts             int = 3
	)

	//log.Panic(projectName)

	errorHeader = fmt.Sprintf("TCP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	fmt.Println("tcp ping test: ", c.Host)

	timeout := c.Timeout * time.Millisecond

	for checkAttempts < c.Attempts {
		//startTime := time.Now()
		conn, err := net.DialTimeout("tcp", c.Host+":"+string(c.Port), timeout)
		//endTime := time.Now()

		if err == nil {
			conn.Close()
			//t := float64(endTime.Sub(startTime)) / float64(time.Millisecond)
			//log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
			return nil
		}

		errorMessage = errorHeader + fmt.Sprintf("connection to host %s failed: %v (attempt %d)\n", c.Host+":"+string(c.Port), err, checkAttempts)
		//log.Printf(errorMessage)
		checkAttempts++
	}

	fmt.Println(errorMessage)
	return errors.New(errorMessage)

}
