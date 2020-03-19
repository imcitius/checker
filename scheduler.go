package main

import (
	"math"
	"time"
)

func runScheduler() {
	done := make(chan bool)
	StartTime := time.Now()
	Ticker := time.NewTicker(time.Duration(Config.Defaults.TimerStep) * time.Second)

	for {

		for {
			select {
			case <-done:
				return
			case t := <-Ticker.C:
				dif := float64(t.Sub(StartTime) / time.Second)
				for _, timeout := range Timeouts {
					if math.Remainder(dif, float64(timeout)) == 0 {
						// fmt.Printf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)
						checkProjects(timeout)
					}
				}
			}
		}
	}
}
