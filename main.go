package main

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version string

func main() {
	log.SetHandler(text.New(os.Stdout))
	if Version != "" {
		log.Infof("Start v%s", Version)
	}
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
			useTimer = false
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
				log.Debug("working")
				break
			case <-timer.C:
				log.Info("time to exit")
				stopCh <- true
				break
			case <-stopSignal:
				log.Info("graceful exit")
				stopCh <- true
				break
			case <-stopCh:
				var status string
				if fail {
					status = "fail"
				} else {
					status = "success"
				}
				log.Infof("exit with %s\n", status)
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
		for {
			select {
			case t := <-ticker.C:
				log.Debugf("working %d\n", t.Unix())
				break
			case <-stopSignal:
				log.Info("graceful exit")
				stopCh <- true
				break
			case <-stopCh:
				ticker.Stop()
				log.Info("exit")
				os.Exit(0)
			}
		}
	}
}
