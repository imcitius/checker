package mongodb

import "go.mongodb.org/mongo-driver/mongo"

type mongoDbStore struct {
	client *mongo.Client
}
