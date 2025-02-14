package store

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"my/checker/config"
	"os"
	"time"
)

type mongoDbStore struct {
	client *mongo.Client
}

func (store mongoDbStore) Init() (IStore, error) {
	DBContext := context.TODO()

	if os.Getenv("CHECKER_DB_PASSWORD") != "" {
		configurer.SetDBPassword(os.Getenv("CHECKER_DB_PASSWORD"))
	}

	mongoconn := options.Client().ApplyURI(configurer.GetDBConnectionString())
	mongoclient, err := mongo.Connect(DBContext, mongoconn)

	if err != nil {
		logger.Fatalf("Cannot connect to MongoDB: %s", err.Error())
	}

	if err := mongoclient.Ping(DBContext, readpref.Primary()); err != nil {
		logger.Fatalf("Cannot connect to MongoDB: %s", err.Error())
	}
	fmt.Println("MongoDB successfully connected...")

	return mongoDbStore{client: mongoclient}, err
}

func (store mongoDbStore) Disconnect() error {
	return store.client.Disconnect(context.Background())
}

func (store mongoDbStore) UpdateChecks() error {
	collection := store.client.Database(configurer.DB.Database).Collection("checks")

	checks, err := configurer.GetAllChecks()
	if err != nil {
		logger.Errorf("Error while getting checks from config: %s", err.Error())
	}

	var models []mongo.WriteModel
	opts := options.BulkWrite().SetOrdered(false)
	for _, v := range checks {
		models = append(models,
			mongo.NewUpdateOneModel().SetFilter(bson.D{{"UUID", v.UUID}}).
				SetUpdate(bson.D{{"$set", bson.D{
					{"Project", v.Project},
					{"Healthcheck", v.Healthcheck},
					{"Name", v.Name},
					{"UUID", v.UUID},
					{"LastResult", v.LastResult},
					{"LastExec", v.LastExec},
					{"LastPing", v.LastPing}},
				}}).SetUpsert(true),
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

func (store mongoDbStore) UpdateAlerts() error {

	return nil
}

func (store mongoDbStore) GetCheckObjectByUUid(uuid string) (DbCheckObject, error) {

	collection := store.client.Database(configurer.DB.Database).Collection("checks")
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
	}, err
}

func (store mongoDbStore) BulkWriteChecks(models []mongo.WriteModel, opts *options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	collection := store.client.Database(configurer.DB.Database).Collection("checks")
	return collection.BulkWrite(DBContext, models, opts)
}

func (store mongoDbStore) BulkWriteAlerts(models []mongo.WriteModel, opts *options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	collection := store.client.Database(configurer.DB.Database).Collection("alerts")
	return collection.BulkWrite(DBContext, models, opts)
}

func (store mongoDbStore) GetAllChecks() ([]DbCheckObject, error) {
	collection := store.client.Database(configurer.DB.Database).Collection("checks")
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

func (store mongoDbStore) GetAllAlerts() (*MessagesContextStorage, error) {
	collection := store.client.Database(configurer.DB.Database).Collection("alerts")
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
