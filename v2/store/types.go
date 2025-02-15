package store

import (
	alerts "my/checker/models/alerts"
	"sync"
)

type MessagesContextStorage struct {
	sync.RWMutex

	data map[int64]map[int]alerts.TAlertDetails
}
