package scheduler

import (
	"my/checker/checks"
	"sync"
	"time"
	//"time"
)

func runProjectTicker(t TTickerWithDuration, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case _ = <-t.Ticker.C:
			tickerDuration, _ := time.ParseDuration(t.Duration) //sendCritAlerts(period)
			checkCollection, _ := checks.GetChecksByDuration(tickerDuration.String())

			for _, c := range checkCollection.Checks[t.Duration] {
				//logger.Infof("runProjectTicker: %+v", c)

				logger.Infof("(%s) Checking (%s/%s/%s): %s", c.Check.GetSID(), c.Check.GetProject(), c.Check.GetHealthcheck(), c.Check.GetType(), c.Check.GetHost())

				d, err := c.Check.Execute()

				if err != nil {
					logger.Errorf("(%s) Failed, took %s\nError: %s", c.Check.GetSID(), d, err.Error())

				} else {
					logger.Infof("(%s) Success, took %s", c.Check.GetSID(), d)
				}
			}
		}
	}
}
