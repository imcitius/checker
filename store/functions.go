package store

import (
	tele "gopkg.in/telebot.v3"
	"my/checker/config"
)

func InitDB() (IStore, error) {
	switch configurer.DB.Protocol {
	case "mongodb":
		client := &mongoDbStore{}
		store, err := client.Init()
		if err != nil {
			return nil, err
		}
		Store = store
	}
	configurer.SetDBConnected()

	loadChecks()
	loadAlerts()
	return Store, nil
}

func loadChecks() {
	checksFromDb, err := Store.GetAllChecks()
	if err != nil {
		logger.Fatalf("Cannot get checks from DB: %s", err)
	}
	for _, o := range checksFromDb {
		check, err := configurer.GetCheckByUUid(o.UUID)
		if err != nil {
			logger.Debugf("Cannot get check drom db: %s", err.Error())
			continue
		}
		check.LastExec = o.LastExec
		check.LastPing = o.LastPing
		check.LastResult = o.LastResult

		err = configurer.UpdateCheckByUUID(check)
		if err != nil {
			logger.Errorf("Cannot update check: %s", err.Error())
		}
	}
}

func loadAlerts() {
	//alertsFromDb, err := Store.GetAllAlerts()

	var err error = nil
	if err != nil {
		logger.Fatalf("Cannot get checks from DB: %s", err)
	}
	//for _, o := range alertsFromDb.data {
	//
	//}
}

func GetMessagesContextStorage() *MessagesContextStorage {
	return &MessagesContextStorage{
		data: make(map[int64]map[int]config.TAlertDetails),
	}
}

func (store *MessagesContextStorage) Update(m *tele.Message) {
	store.Lock()
	defer store.Unlock()

	if store.data[m.Chat.ID] == nil {
		store.data[m.Chat.ID] = make(map[int]config.TAlertDetails)
	}
	store.data[m.Chat.ID][m.ID] = config.TAlertDetails{}
}

func (store *MessagesContextStorage) GetData() interface{} {
	return store.data
}
