package db

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"checker/internal/config"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(cfg *config.Config) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var mongoURI string
	if cfg.DB.Host == "localhost:27017" || cfg.DB.Host == "127.0.0.1:27017" {
		// Local MongoDB connection without authentication
		mongoURI = "mongodb://" + cfg.DB.Host
		logrus.Debug("Using local MongoDB connection without authentication")
	} else {
		// Remote MongoDB connection with authentication
		dbPassword := cfg.DB.Password
		pass, ok := os.LookupEnv("CHECKER_DB_PASSWORD")
		if ok {
			dbPassword = pass
		}
		mongoURI = "mongodb+srv://" + cfg.DB.Username + ":" + dbPassword + "@" + cfg.DB.Host + "/" + cfg.DB.Database + "?retryWrites=true&w=majority"
		logrus.Debug("Using remote MongoDB connection with authentication")
	}
	clientOpts := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	logrus.Infof("Connected to MongoDB at %s", cfg.DB.Host)
	database := client.Database(cfg.DB.Database)
	return &MongoDB{Client: client, Database: database}, nil
}

// Collection returns a handle for a Mongo collection.
func (mdb *MongoDB) Collection(name string) *mongo.Collection {
	return mdb.Database.Collection(name)
}
