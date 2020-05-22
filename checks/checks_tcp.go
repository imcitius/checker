package check

import (
	"errors"
	"fmt"
	"my/checker/config"
	"net"
	"time"
)

func init() {
	config.Checks["tcp"] = func (c *config.Check, p *config.Project) error {
		var (
			errorMessage string
			checkAttempts int = 3
			checkAttempt int
		)

		if c.Attempts != 0 {
			checkAttempts = c.Attempts
		}
		//config.Log.Panicf("%+v", c)

		address := fmt.Sprintf("%s:%d", c.Host, c.Port)
		errorHeader := fmt.Sprintf("TCP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, address, c.UUid)

		fmt.Printf("tcp ping test: %s\n`", address)

		timeout, _ := time.ParseDuration(c.Timeout)

		//config.Log.Panic(timeout)

		for checkAttempt < checkAttempts {
			//startTime := time.Now()
			conn, err := net.DialTimeout("tcp", address, timeout)
			//endTime := time.Now()

			if err == nil {
				defer conn.Close()
				//t := float64(endTime.Sub(startTime)) / float64(time.Millisecond)
				//config.Log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
				return nil
			}

			errorMessage = errorHeader + fmt.Sprintf("connection to host %s failed: %v (attempt %d)\n", c.Host+":"+string(c.Port), err, checkAttempts)
			config.Log.Printf(errorMessage)
			checkAttempt++
		}
		fmt.Println(errorMessage)
		return errors.New(errorMessage)

	}
}
