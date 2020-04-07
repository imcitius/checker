package main

import (
	"errors"
	"fmt"
	"github.com/sparrc/go-ping"
	"time"
)

func runICMPCheck(c *Check, p *Project) error {
	var (
		errorHeader, errorMessage string
	)

	errorHeader = fmt.Sprintf("ICMP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	fmt.Println("icmp ping test: ", c.Host)
	pinger, err := ping.NewPinger(c.Host)
	pinger.Count = c.Count
	pinger.Timeout, _ = time.ParseDuration(c.Timeout)
	pinger.Run()
	stats := pinger.Statistics()

	//log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", c.Host, pinger, err, stats)

	if err == nil && stats.PacketLoss == 0 {
		return nil
	} else {
		switch {
		case stats.PacketLoss > 0:
			//log.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf("ping error: %v percent packet loss\n", stats.PacketLoss)
		default:
			//log.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf("other ping error: %+v\n", err)
		}
	}

	//log.Println(errorMessage)
	return errors.New(errorMessage)

}
