package store

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type mongoDbStore struct {
	client *mongo.Client
}

func (store mongoDbStore) Init(dbConfig DBConfig) (IStore, error) {
	DBContext := context.TODO()

	dbPassword := dbConfig.Password
	if dbConfig.Password == "" {
		dbPassword = os.Getenv("CHECKER_DB_PASSWORD")
	}
	// construct connection string
	connectionString := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority",
		dbConfig.Username, dbPassword, dbConfig.Host)
	
	mongoconn := options.Client().ApplyURI(connectionString)
	mongoclient, err := mongo.Connect(DBContext, mongoconn)

	if err != nil {
		logrus.Fatalf("Cannot connect to MongoDB: %s", err.Error())
	}

	if err := mongoclient.Ping(DBContext, readpref.Primary()); err != nil {
		logrus.Fatalf("Cannot connect to MongoDB: %s", err.Error())
	}
	fmt.Println("MongoDB successfully connected...")

	return mongoDbStore{client: mongoclient}, err
}

func (store mongoDbStore) Disconnect() error {
	return store.client.Disconnect(context.Background())
}
