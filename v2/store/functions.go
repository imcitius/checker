package store

import (
	alerts "my/checker/models/alerts"

	tele "gopkg.in/telebot.v3"
)



func GetMessagesContextStorage() *MessagesContextStorage {
	return &MessagesContextStorage{
		data: make(map[int64]map[int]alerts.TAlertDetails),
	}
}

func (store *MessagesContextStorage) Update(m *tele.Message) {
	store.Lock()
	defer store.Unlock()

	if store.data[m.Chat.ID] == nil {
		store.data[m.Chat.ID] = make(map[int]alerts.TAlertDetails)
	}
	store.data[m.Chat.ID][m.ID] = alerts.TAlertDetails{}
}

func (store *MessagesContextStorage) GetData() interface{} {
	return store.data
}
