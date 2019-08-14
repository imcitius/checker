package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var timeout time.Duration
	var timer *time.Timer
	var ticker = time.NewTicker(time.Second)

	var err error

	fail := false
	if e, ok := os.LookupEnv("GUIDE_FAIL"); ok {
		fail = e == "true"
	}

	useTimer := false
	if e, ok := os.LookupEnv("GUIDE_RUN_TIME"); ok {
		useTimer = true
		timeout, err = time.ParseDuration(e)
		if err != nil {
			fmt.Println("Wrong value in GUIDE_RUN_TIME")
			os.Exit(1)
		}
	}

	if useTimer {
		timer = time.NewTimer(timeout)
	}

	stopSignal := make(chan os.Signal)
	signal.Notify(stopSignal, syscall.SIGTERM)
	signal.Notify(stopSignal, syscall.SIGINT)

	stopCh := make(chan bool, 1)

	if useTimer {
		for {
			select {
			case <-ticker.C:
				fmt.Println("working")
				break
			case <-timer.C:
				fmt.Println("time to exit")
				stopCh <- true
				break
			case <-stopSignal:
				fmt.Println("graceful exit")
				stopCh <- true
				break
			case <-stopCh:
				var status string
				if fail {
					status = "fail"
				} else {
					status = "success"
				}
				fmt.Printf("exit with %s\n", status)
				ticker.Stop()
				timer.Stop()
				if fail {
					os.Exit(1)
				} else {
					os.Exit(0)
				}
			}
		}
	} else {
		if fail {
			ticker.Stop()
			os.Exit(2)
		}

	}
}
