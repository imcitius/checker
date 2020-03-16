package main

import (
	"time"
)

func urlChecks(token string) {
	for {
		<-time.After(time.Duration(config.Secs) * time.Second)

		testurls(probes)
	}
}
