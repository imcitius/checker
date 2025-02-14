package scheduler

import (
	"context"
	"fmt"
	"my/checker/checks"
	"sync"
	"time"
	//"time"
)

func getCheckCollectionByDuration(duration string) checks.TChecksCollection {
	checkCollection, err := checks.GetChecksByDuration(duration)
	if err != nil {
		logger.Errorf("Error getting checks by duration: %s", err.Error())
	}

	return checkCollection
}

func runProjectTicker(t TTickerWithDuration, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	tickerDuration, _ := time.ParseDuration(t.Duration) //sendCritAlerts(period)
	// checkCollection, _ := checks.GetChecksByDuration(tickerDuration.String())


	for range t.Ticker.C {
		for _, c := range getCheckCollectionByDuration(tickerDuration.String()).Checks {
			// logger.Infof("\t %d checks for duration %s", checkCollection.Len(), tickerDuration.String())
			if !c.Check.GetEnabled() {
				logger.Infof("Check %s is disabled, skipping", c.Check.GetUUID())
				continue
			}

			res := c.Check.Execute()

			header := fmt.Sprintf("(%s) %s/%s/%s, %s ", c.Check.GetSID(), c.Check.GetProject(), c.Check.GetHealthcheck(), c.Check.GetType(), c.Check.GetHost())
			if res.Result.Error != nil {
				message := fmt.Sprintf("%s Failed: %s", header, res.Result.Error.Error())
				logger.Errorf("Check error: %s", message)
				c.Check.SetStatus(false)
				res.Alert(ctx, message)
			} else {
				logger.Infof("%s Success, took %s", header, res.Result.Duration)
				c.Check.SetStatus(true)
			}

			if configurer.DB.Connected {
				err := checks.UpdateChecksByCollectioninDB(getCheckCollectionByDuration(tickerDuration.String()).Checks)
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

	for range t.Ticker.C {
		for _, c := range maintenanceTasksCollection.Tickers {
			res := c.exec()
			if res != nil {
				logger.Errorf("Failed to execute maintenance task: %s", res.Error())
			}
		}
	}
}
