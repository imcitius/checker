package store

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
	return Store, nil
}
