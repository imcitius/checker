package scheduler

import (
	"fmt"
	"my/checker/checks"
	"sync"
	"time"
	//"time"
)

func runProjectTicker(t TTickerWithDuration, wg *sync.WaitGroup) {
	defer wg.Done()
	tickerDuration, _ := time.ParseDuration(t.Duration) //sendCritAlerts(period)
	checkCollection, _ := checks.GetChecksByDuration(tickerDuration.String())

	logger.Infof("\t %d checks for duration %s", checkCollection.Len(), tickerDuration.String())

	for {
		select {
		case _ = <-t.Ticker.C:

			for _, c := range checkCollection.Checks {
				res := c.Check.Execute()

				header := fmt.Sprintf("(%s) %s/%s/%s, %s ", c.Check.GetSID(), c.Check.GetProject(), c.Check.GetHealthcheck(), c.Check.GetType(), c.Check.GetHost())
				if res.Result.Error != nil {
					message := fmt.Sprintf("%s Failed: %s", header, res.Result.Error.Error())
					logger.Errorf(message)
					c.Check.SetStatus(false)
					res.Alert(message)
				} else {
					logger.Infof("%s Success, took %s", header, res.Result.Duration)
					c.Check.SetStatus(true)
				}

				err := checks.UpdateChecksByCollection(checkCollection.Checks)
				if err != nil {
					logger.Errorf("Scheduler failed to update checks in DB: %s", err.Error())
				}
			}
		}
	}
}

func runMaintenanceTicker(t TMaintenanceTicker, wg *sync.WaitGroup) {
	defer wg.Done()
	tickerDuration, _ := time.ParseDuration(t.Duration) //sendCritAlerts(period)
	maintenanceTasksCollection := GetMaintenanceTickers()

	logger.Infof("\t %d maintenances for duration %s", len(maintenanceTasksCollection.Tickers), tickerDuration.String())

	for {
		select {
		case _ = <-t.Ticker.C:
			for _, c := range maintenanceTasksCollection.Tickers {
				res := c.exec()
				if res != nil {
					logger.Errorf("Failed to execute maintenance task: %s", res.Error())
				}
			}
		}
	}
}
