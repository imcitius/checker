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
