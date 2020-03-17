package main

import (
	"fmt"
	"time"
)

func runTimer() {
	for {
		ticker := time.NewTicker(time.Duration(CheckData.Secs) * time.Second)
		done := make(chan bool)

		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)
				checkProjects()
			}
		}
	}
}
