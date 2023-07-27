package icmp

import (
	"errors"
	"fmt"
	"github.com/go-ping/ping"
	"time"
)

func (c *TICMPCheck) RealExecute() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
	)

	start := time.Now()

	errorHeader = fmt.Sprintf(ErrICMPError)

	if c.Host == "" {
		errorMessage = errorHeader + "empty host\n"
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	logger.Debugf("icmp ping test: %s", c.Host)
	pinger, err := ping.NewPinger(c.Host)
	pinger.Count = c.Count
	pinger.Timeout, _ = time.ParseDuration(c.Timeout)
	err = pinger.Run()
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrPingError, err)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	stats := pinger.Statistics()
	//logger.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", c.Host, pinger, err, stats)

	if err != nil || stats.PacketLoss != 0 {
		switch {
		case stats.PacketLoss > 0:
			//logger.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf(ErrPacketLoss, stats.PacketLoss)
		default:
			//logger.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf(ErrOther, err.Error())
		}
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	//logger.Println(errorMessage)
	return time.Now().Sub(start), nil
}
