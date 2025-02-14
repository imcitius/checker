package store

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IStore interface {
	Init() (IStore, error)
	Disconnect() error

	GetCheckObjectByUUid(string) (DbCheckObject, error)
	BulkWriteChecks([]mongo.WriteModel, *options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	BulkWriteAlerts([]mongo.WriteModel, *options.BulkWriteOptions) (*mongo.BulkWriteResult, error)

	//GetData() (interface{}, error)
	//Save() error
	//Load() error

	GetAllChecks() ([]DbCheckObject, error)
	GetAllAlerts() (*MessagesContextStorage, error)
	UpdateChecks() error
	UpdateAlerts() error
	//UpdateChecksByCollection(checks checks.TChecksCollection) error
}
