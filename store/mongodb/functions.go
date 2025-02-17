package store

func InitDB(dbConfig DBConfig) (IStore, error) {
	switch dbConfig.Protocol {
	case "mongodb":
		client := &mongoDbStore{}
		store, err := client.Init(dbConfig)
		if err != nil {
			return nil, err
		}
		Store = store
	}
	return Store, nil
}
