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

	loadChecks()
	return Store, nil
}

func loadChecks() {
	checksFromDb, err := Store.GetAllChecks()
	if err != nil {
		logger.Fatalf("Cannot get checks from DB: %s", err)
	}
	for _, o := range checksFromDb {
		check, err := configurer.GetCheckByUUid(o.UUid)
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
