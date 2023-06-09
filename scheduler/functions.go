package scheduler

import (
	"time"
)

// GetTickers function to return all tickers
// Caching not needed, because it's called only once
func GetTickers() (TTickersCollection, error) {
	tickers := TTickersCollection{
		make(map[string]TTickerWithDuration),
	}

	if configurer.Defaults.Duration != "" {
		defaultDuration, err := time.ParseDuration(configurer.Defaults.Duration)
		if err != nil {
			logger.Fatalf("Failed to parse duration: %s", err.Error())
		}
		tickers.Tickers[configurer.Defaults.Duration] = TTickerWithDuration{
			time.NewTicker(defaultDuration),
			defaultDuration.String(),
		}
	}

	for _, project := range configurer.Projects {
		for _, healthcheck := range project.Healthchecks {
			for _, check := range healthcheck.Checks {
				if check.Parameters.Duration != "" {
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

	// add code to actual parse tickers from config

	return tickers, nil
}
