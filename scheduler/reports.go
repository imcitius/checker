package scheduler

import (
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
	"my/checker/status"
	"sync"
	"time"
)

func runReportsTicker(t *time.Ticker, desc string, wg *sync.WaitGroup, ch chan bool) {
	config.Log.Infof("Starting report ticker %s", desc)
	config.Wg.Add(1)
	defer wg.Done()

	config.Log.Debugf("Waiting for ticker %s", desc)
	defer config.Log.Debugf("Finished ticker %s", desc)
	for {
		select {
		case <-ch:
			config.Log.Infof("Exit reports ticker")
			return
		case tick := <-t.C:
			uptime := tick.Round(time.Second).Sub(config.StartTime.Round(time.Second))
			config.Log.Debugf("ticker.Description: %s", desc)
			period := desc
			config.Log.Infof("Uptime: %s (%s ticker)", uptime, desc)
			reportsDuration := runReports(period)
			config.Log.Debugf("Reports duration: %v msec", reportsDuration.Milliseconds())
			metrics.SchedulerReportsDuration.Set(float64(reportsDuration.Milliseconds()))
		}
	}
}

func runReports(period string) time.Duration {
	startTime := time.Now()
	config.Log.Debugf("runReports: %s", period)
	for _, p := range Config.Projects {
		config.Log.Debugf("runReports 0: %s\n", period)
		config.Log.Debugf("runReports 1: %s\n", p.Name)
		config.Log.Debugf("runReports 2: %s\n", p.Parameters.Mode)
		config.Log.Debugf("runReports 4: %s\n", status.Statuses.Projects[p.Name].Mode)
		config.Log.Debugf("runReports 6: %s\n", p.Parameters.ReportPeriod)

		schedPeriod, err := time.ParseDuration(period)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse SchedPeriod duration %s", err)
		} else {
			config.Log.Debugf("schedPeriod: %s\n", schedPeriod)
		}
		reportsPeriod, err := time.ParseDuration(p.Parameters.ReportPeriod)
		if err != nil {
			config.Log.Errorf("runReports Cannot parse ReportPeriod duration %s", err)
		}
		config.Log.Debugf("reportsPeriod: %s\n", reportsPeriod)

		if schedPeriod >= reportsPeriod {
			project := projects.Project{Project: p}
			config.Log.Debugf("runReports 10: %s", project.GetMode())
			err := project.ProjectSendReport()
			if err != nil {
				config.Log.Errorf("Cannot send report for project %s: %+v", project.Name, err)
			}
		}
	}

	if config.Config.Defaults.Parameters.ReportPeriod == period {
		if status.MainStatus == "quiet" {
			reportMessage := "All messages ceased"
			alerts.SendChatOps(reportMessage)
		}
	}
	return time.Since(startTime)
}
