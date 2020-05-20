package cmd

import (
	"github.com/spf13/viper"
	"go-boilerplate/config"
	"math"
	"sync"
	"time"
)

func fillTimeouts(c *config.ConfigFile, t *config.TimeoutsCollection) error {
	t.Add(config.Config.Defaults.Parameters.RunEvery)

	for _, project := range c.Projects {

		//config.Log.Debugf("Project name: %s", project.Name)
		//config.Log.Debugf("Parameters: %+v", project.Parameters)

		if project.Parameters.RunEvery != c.Defaults.Parameters.RunEvery {
			t.Add(project.Parameters.RunEvery)
		}
		for _, healthcheck := range project.Healtchecks {
			if healthcheck.Parameters.RunEvery != c.Defaults.Parameters.RunEvery {
				t.Add(healthcheck.Parameters.RunEvery)
				project.Timeouts.Add(healthcheck.Parameters.RunEvery)
			}
			config.Log.Debugf("Project %s timeouts found: %+v\n", project.Name, project.Timeouts)
		}
	}
	config.Log.Debugf("Total timeouts found: %+v\n\n", t)

	return nil
}

func runScheduler(signalCh chan bool, wg *sync.WaitGroup) {

	Timeouts := new(config.TimeoutsCollection)
	err := fillTimeouts(&config.Config, Timeouts)

	StartTime := time.Now()

	timerStep, err := time.ParseDuration(viper.GetString("defaults.timer_step"))
	if err != nil {
		config.Log.Fatal(err)
	}

	Ticker := time.NewTicker(timerStep)

	config.Log.Debug("Scheduler started")
	config.Log.Debugf("Timeouts: %+v", Timeouts.Periods)

	for {
		config.Log.Debugf("Scheduler loop #: %d", config.ScheduleLoop)
		select {
		case <-signalCh:
			config.Log.Infof("Exit scheduler")
			wg.Done()
		case t := <-Ticker.C:
			dif := float64(t.Sub(StartTime) / time.Second)

			for i, timeout := range Timeouts.Periods {
				config.Log.Debugf("Got timeout #%d: %s", i, timeout)

				tf, err := time.ParseDuration(timeout)
				if err != nil {
					config.Log.Errorf("Cannot parse timeout: %s", err)
				}
				config.Log.Debugf("Parsed timeout #%d: %+v", i, tf)

				if math.Remainder(dif, tf.Seconds()) == 0 {
					config.Log.Debugf("Time: %v\nTimeout: %v\n===\n\n", t, timeout)

					config.Log.Infof("Schedule: %s", timeout)

					//go runChecks(timeout)
					//go runReports(timeout)
					//runAlerts(timeout)
				}
			}
		}
		config.ScheduleLoop++
	}
}
