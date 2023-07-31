package store

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"os"
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

func (store mongoDbStore) GetData() (interface{}, error) {
	collection := store.client.Database(configurer.DB.Database).Collection("users")
	res := collection.FindOne(DBContext, bson.M{"name": "zhopa"})
	//if err != nil {
	//	logger.Errorf("Error while getting data from MongoDB: %s", err.Error())
	//}

	results := bson.M{}

	err := res.Decode(&results)
	if err != nil {
		logger.Errorf("Error decoding MongoDB collection: %s", err.Error())
	}

	return &results, err
}

func (store mongoDbStore) UpdateChecks() error {
	collection := store.client.Database(configurer.DB.Database).Collection("checks")

	res, err := configurer.GetChecks()
	if err != nil {
		logger.Errorf("Error while getting checks from config: %s", err.Error())
	}

	models := []mongo.WriteModel{}
	opts := options.BulkWrite().SetOrdered(false)
	for _, v := range res {
		models = append(models,
			mongo.NewUpdateOneModel().SetFilter(bson.D{{"UUid", v.UUid}}).
				SetUpdate(bson.D{{"$set", bson.D{
					{"project", v.Project},
					{"healthcheck", v.Healthcheck},
					{"name", v.Name},
					{"UUid", v.UUid},
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
	logger.Debugf("Number of documents replaced or modified, upserted: %d, %d", results.ModifiedCount, results.UpsertedCount)

	return err
}
