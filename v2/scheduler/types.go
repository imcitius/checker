package scheduler

import (
	"time"
)

type TTimeoutsCollection struct {
	Periods []string
}

type TTickersCollection struct {
	Tickers map[string]TTickerWithDuration
}

type TTickerWithDuration struct {
	Ticker   *time.Ticker
	Duration string
}

type TMaintenanceTickersCollection struct {
	Tickers map[string]TMaintenanceTicker
}

type TMaintenanceTicker struct {
	Duration string
	Ticker   *time.Ticker
	exec     func() error
}
