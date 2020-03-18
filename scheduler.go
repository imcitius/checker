package main

import (
	"math"
	"time"
)

func schedule(ticker *time.Ticker, starttime time.Time) {
	done := make(chan bool)

	for {

		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				dif := float64(t.Sub(starttime) / time.Second)
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
