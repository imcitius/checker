package store

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IStore interface {
	Init() (IStore, error)
	Disconnect() error

	GetObjectByUUid(string) (DbCheckObject, error)
	BulkWrite([]mongo.WriteModel, *options.BulkWriteOptions) (*mongo.BulkWriteResult, error)

	//GetData() (interface{}, error)
	//Save() error
	//Load() error

	GetAllChecks() ([]DbCheckObject, error)
	UpdateChecks() error
	//UpdateChecksByCollection(checks checks.TChecksCollection) error
}
