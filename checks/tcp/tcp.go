package tcp

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func (c TTCPCheck) RealExecute() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
		checkAttempts             = c.Count
		checkAttempt              int
	)

	start := time.Now()

	errorHeader = fmt.Sprintf(ErrTCPError)

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	logger.Debugf("tcp ping test: %s\n", address)

	timeout, _ := time.ParseDuration(c.Timeout)

	for checkAttempt < checkAttempts {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			errorMessage = errorHeader + fmt.Sprintf(ErrConnectError, address, checkAttempt, checkAttempts, err)
			return time.Now().Sub(start), errors.New(errorMessage)
		}

		defer func() {
			err = conn.Close()
		}()

		if err != nil {
			errorMessage = errorHeader + fmt.Sprintf(ErrConnectError, address, checkAttempt, checkAttempts, err)
			return time.Now().Sub(start), errors.New(errorMessage)
		}
		checkAttempt++
	}

	return time.Now().Sub(start), nil
}
