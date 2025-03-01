package checks

import (
	"errors"
	"fmt"
	"github.com/go-ping/ping"
	"github.com/sirupsen/logrus"
	"time"
)

// PingCheck represents a Ping health check.
type PingCheck struct {
	Host    string
	Count   int
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the Ping health check.
func (pc *PingCheck) Run() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
		start                     = time.Now()
	)

	if pc.Host == "" {
		errorMessage = errorHeader + ErrEmptyHost
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	logrus.Debugf("icmp ping test: %s", pc.Host)
	pinger, err := ping.NewPinger(pc.Host)
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrICMPError, err)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	pinger.Count = pc.Count
	pinger.Timeout, _ = time.ParseDuration(pc.Timeout)
	err = pinger.Run()
	stats := pinger.Statistics()

	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrICMPError, err)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	pc.Logger.WithError(err).Debugf("Ping host %s, res: %+v (err: %+v, stats: %+v)", pc.Host, pinger, err, stats)

	if stats.PacketLoss != 0 {
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

	return time.Now().Sub(start), nil
}
