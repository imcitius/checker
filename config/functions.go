package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/kofalt/go-memoize"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	store "my/checker/store/mongodb"
)

var (
	logger = logrus.New()
	cache  *memoize.Memoizer
	wg     sync.WaitGroup
)

func InitConfig(cfgFile string) {
	initConfig(cfgFile)
	cache = memoize.NewMemoizer(24*time.Hour, 24*time.Hour)
	config.Defaults.DefaultCheckParameters.Duration = config.Defaults.Duration
	config.refineProjects()
}

func InitLog(logLevel string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(level)
	if logger.GetLevel() == 5 {
		logger.SetReportCaller(true)
	}
}

func GetConfig() *TConfig {
	return &config
}

func GetLog() *logrus.Logger {
	return logger
}

func GetProjectByName(name string) (TProject, error) {
	result, err, _ := cache.Memoize(fmt.Sprintf("projectByName-%s", name), func() (interface{}, error) {
		return findProjectByName(name)
	})
	return result.(TProject), err
}

func (c *TConfig) SetDBConnected(store store.IStore) {
	c.DB.Connected = true
	c.Store = store
}

func findProjectByName(name string) (TProject, error) {
	for _, p := range config.Projects {
		if p.Name == name {
			return p, nil
		}
	}
	return TProject{}, fmt.Errorf("project not found")
}

func GetWG() *sync.WaitGroup {
	return &wg
}

func (c *TConfig) GetCheckByUUid(uuid string) (TCheckConfig, error) {
	for _, p := range c.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if c.UUID == uuid {
					return c, nil
				}
			}
		}
	}
	return TCheckConfig{}, fmt.Errorf("check not found")
}

func (c *TConfig) ListChecks() (interface{}, error) {
	type listObject struct {
		UUID       string
		LastResult bool
		LastExec   time.Time
		LastPing   time.Time
	}

	var list map[string]map[string]map[string]listObject

	list = make(map[string]map[string]map[string]listObject)
	for _, p := range c.Projects {
		list[p.Name] = make(map[string]map[string]listObject)
		for _, h := range p.Healthchecks {
			list[p.Name][h.Name] = make(map[string]listObject)
			for _, c := range h.Checks {
				list[p.Name][h.Name][c.Name] = listObject{
					UUID:       c.UUID,
					LastResult: c.LastResult,
					LastExec:   c.LastExec,
					LastPing:   c.LastPing,
				}
			}
		}
	}
	return list, nil
}

func (c *TConfig) GetAllChecks() ([]TCheckConfig, error) {

	list := make([]TCheckConfig, 0)

	for _, p := range c.Projects {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				list = append(list, c)
			}
		}
	}
	return list, nil
}

func (c *TConfig) Ping(uuid string) (TCheckConfig, error) {

	check, _ := c.GetCheckByUUid(uuid)
	check.LastPing = time.Now()
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check
	return check, nil
}

func (c *TConfig) SetStatus(uuid string, status bool) {
	check, _ := c.GetCheckByUUid(uuid)
	check.LastExec = time.Now()
	check.LastResult = status
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check

	if err := c.Save(); err != nil {
		logger.Errorf("Failed to save config: %v", err)
	}
}

func (c *TConfig) GetDBConnectionString() string {
	if c.DB.Protocol == "mongodb" {
		return fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority",
			c.DB.Username, c.DB.Password, c.DB.Host)
	}

	return ""
}

func (c *TConfig) SetDBPassword(password string) {
	c.DB.Password = password
}

func (c *TConfig) GetCheckDetails(uuid string) TCheckDetails {
	check, _ := c.GetCheckByUUid(uuid)
	return TCheckDetails{
		Project:     check.Project,
		Healthcheck: check.Healthcheck,
		Name:        check.Name,
		UUID:        check.UUID,
		LastExec:    check.LastExec,
		LastResult:  check.LastResult,
		LastPing:    check.LastPing,
		Enabled:     check.Enabled,
	}
}

func (c *TConfig) UpdateCheckByUUID(check TCheckConfig) error {
	p := c.Projects
	p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name] = check

	logger.Infof("Check state: %+v", p[check.Project].Healthchecks[check.Healthcheck].Checks[check.Name])

	if err := c.Save(); err != nil {
		logger.Errorf("Failed to save config: %v", err)
		return err
	}

	return nil
}

func (c *TConfig) ToggleCheck(uuid string, enabled bool) error {
	check, _ := c.GetCheckByUUid(uuid)
	check.Enabled = enabled
	c.UpdateCheckByUUID(check)

	return nil
}

func (c *TConfig) Save() error {
	if !c.DB.Connected {
		return fmt.Errorf("database not connected")
	}
	return c.UpdateChecks()
}

// func GetMessagesContextStorage() *MessagesContextStorage {
// 	return &MessagesContextStorage{
// 		data: make(map[int64]map[int]config.TAlertDetails),
// 	}
// }

// func (store *MessagesContextStorage) Update(m *tele.Message) {
// 	store.Lock()
// 	defer store.Unlock()

// 	if store.data[m.Chat.ID] == nil {
// 		store.data[m.Chat.ID] = make(map[int]config.TAlertDetails)
// 	}
// 	store.data[m.Chat.ID][m.ID] = config.TAlertDetails{}
// }

// func (store *MessagesContextStorage) GetData() interface{} {
// 	return store.data
// }

func (config *TConfig) UpdateChecks() error {
	collection := config.Store.client.Database().Collection("checks")

	checks, err := config.GetAllChecks()
	if err != nil {
		logrus.Errorf("Error while getting checks from config: %s", err.Error())
	}

	var models []mongo.WriteModel
	opts := options.BulkWrite().SetOrdered(false)
	for _, v := range checks {
		models = append(models,
			mongo.NewUpdateOneModel().SetFilter(bson.D{{Key: "UUID", Value: v.UUID}}).
				SetUpdate(bson.D{{Key: "$set", Value: bson.D{
					{Key: "Project", Value: v.Project},
					{Key: "Healthcheck", Value: v.Healthcheck},
					{Key: "Name", Value: v.Name},
					{Key: "UUID", Value: v.UUID},
					{Key: "LastResult", Value: v.LastResult},
					{Key: "LastExec", Value: v.LastExec},
					{Key: "LastPing", Value: v.LastPing},
					{Key: "Enabled", Value: v.Enabled},
				}}}).SetUpsert(true),
		)
	}

	results, err := collection.BulkWrite(DBContext, models, opts)
	if err != nil {
		logger.Errorf("Error while inserting checks to MongoDB: %s", err.Error())
	}

	// When you run this file for the first time, it should print:
	// Number of documents replaced or modified: 2
	logger.Infof("MongoDB updated, replaced or modified, upserted: %d, %d", results.ModifiedCount, results.UpsertedCount)

	return err
}

func UpdateAlerts() error {
	return nil
}

func GetCheckObjectByUUid(uuid string) (DbCheckObject, error) {
	collection := config.Store.client.Database().Collection("checks")
	res := collection.FindOne(DBContext, bson.M{"UUID": uuid})
	results := bson.M{}

	err := res.Decode(&results)
	if err != nil {
		logger.Errorf("Error decoding MongoDB collection: %s", err.Error())
		return DbCheckObject{}, err
	}

	return DbCheckObject{
		Project:     results["Project"].(string),
		Healthcheck: results["Healthcheck"].(string),
		Name:        results["Name"].(string),
		UUID:        results["UUID"].(string),
		LastResult:  results["LastResult"].(bool),
		LastExec:    time.Unix(results["LastExec"].(primitive.DateTime).Time().Unix(), 0),
		LastPing:    time.Unix(results["LastPing"].(primitive.DateTime).Time().Unix(), 0),
		Enabled:     results["Enabled"].(bool),
	}, err
}

func BulkWriteChecks(models []mongo.WriteModel, opts *options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	collection := config.Store.client.Database().Collection("checks")
	return collection.BulkWrite(DBContext, models, opts)
}

func BulkWriteAlerts(models []mongo.WriteModel, opts *options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	collection := config.Store.client.Database().Collection("alerts")
	return collection.BulkWrite(DBContext, models, opts)
}

func GetAllChecks() ([]DbCheckObject, error) {
	collection := config.Store.client.Database().Collection("checks")
	cur, err := collection.Find(DBContext, bson.M{})
	if err != nil {
		logger.Errorf("Error while getting checks from MongoDB: %s", err.Error())
	}

	var checks []DbCheckObject
	for cur.Next(DBContext) {
		var t DbCheckObject
		err := cur.Decode(&t)
		if err != nil {
			return checks, err
		}

		checks = append(checks, t)
	}

	if err := cur.Err(); err != nil {
		return checks, err
	}

	return checks, err
}

func GetAllAlerts() (*MessagesContextStorage, error) {
	collection := config.Store.client.Database().Collection("alerts")
	cur, err := collection.Find(DBContext, bson.M{})
	if err != nil {
		logger.Errorf("Error while getting checks from MongoDB: %s", err.Error())
	}

	var res MessagesContextStorage
	for cur.Next(DBContext) {
		var t map[int]config.TAlertDetails
		err := cur.Decode(&t)
		if err != nil {
			return nil, err
		}
	}

	if err := cur.Err(); err != nil {
		return &res, err
	}

	return &res, err
}

func UpdateSingleCheck(check DbCheckObject) error {
	collection := config.Store.client.Database().Collection("checks")

	_, err := collection.UpdateOne(
		DBContext,
		bson.M{"UUID": check.UUID},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "Project", Value: check.Project},
			{Key: "Healthcheck", Value: check.Healthcheck},
			{Key: "Name", Value: check.Name},
			{Key: "LastResult", Value: check.LastResult},
			{Key: "LastExec", Value: check.LastExec},
			{Key: "LastPing", Value: check.LastPing},
			{Key: "Enabled", Value: check.Enabled},
		}}},
	)

	return err
}
