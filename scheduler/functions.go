package scheduler

import (
	"my/checker/store"
	"time"
)

// GetTickers function to return all tickers
// Caching not needed, because it's called only once
func GetTickers() (TTickersCollection, error) {
	tickers := TTickersCollection{
		make(map[string]TTickerWithDuration),
	}

	defaultDuration, err := time.ParseDuration(configurer.Defaults.Duration)
	if err != nil {
		logger.Fatalf("Failed to parse duration: %s", err.Error())
	}
	tickers.Tickers[configurer.Defaults.Duration] = TTickerWithDuration{
		time.NewTicker(defaultDuration),
		defaultDuration.String(),
	}

	for _, project := range configurer.Projects {
		for _, healthcheck := range project.Healthchecks {
			logger.Debugf("Healthcheck %s", healthcheck.Name)
			if healthcheck.Parameters.Duration != "" {
				logger.Debugf("Healthcheck %s duration: %s", healthcheck.Name, healthcheck.Parameters.Duration)
				duration, err := time.ParseDuration(healthcheck.Parameters.Duration)
				if err != nil {
					logger.Fatalf("Failed to parse duration: %s", err.Error())
				}

				if _, ok := tickers.Tickers[healthcheck.Parameters.Duration]; !ok {
					tickers.Tickers[healthcheck.Parameters.Duration] = TTickerWithDuration{
						time.NewTicker(duration),
						duration.String(),
					}
				}
			}

			for _, check := range healthcheck.Checks {
				if check.Parameters.Duration != "" {
					logger.Debugf("Check %s duration: %s", check.Name, check.Parameters.Duration)
					duration, err := time.ParseDuration(check.Parameters.Duration)
					if err != nil {
						logger.Fatalf("Failed to parse duration: %s", err.Error())
					}

					if _, ok := tickers.Tickers[check.Parameters.Duration]; !ok {
						tickers.Tickers[check.Parameters.Duration] = TTickerWithDuration{
							time.NewTicker(duration),
							duration.String(),
						}
					}
				}
			}
		}

		if project.Parameters.Duration != "" {
			logger.Debugf("Project %s duration: %s", project.Name, project.Parameters.Duration)
			duration, err := time.ParseDuration(project.Parameters.Duration)
			if err != nil {
				logger.Fatalf("Failed to parse duration: %s", err.Error())
			}
			if _, ok := tickers.Tickers[project.Parameters.Duration]; !ok {
				tickers.Tickers[project.Parameters.Duration] = TTickerWithDuration{
					time.NewTicker(duration),
					duration.String(),
				}
			}
		}
	}

	return tickers, nil
}

func GetMaintenanceTickers() TMaintenanceTickersCollection {
	// if DB is connected, persist status every minute
	tickers := TMaintenanceTickersCollection{
		make(map[string]TMaintenanceTicker),
	}

	if configurer.DB.Connected {
		tickers.Tickers[configurer.Defaults.MaintenanceDuration] = TMaintenanceTicker{
			Duration: configurer.Defaults.MaintenanceDuration,
			Ticker:   time.NewTicker(time.Minute),
			exec: func() error {
				err := store.Store.UpdateChecks()
				if err != nil {
					logger.Errorf("Failed to update checks in DB: %s", err.Error())
					return err
				}
				return nil
			},
		}
	}

	return tickers
}
