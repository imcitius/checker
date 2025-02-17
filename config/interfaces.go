package config

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IStore interface {
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
	UpdateSingleCheck(check DbCheckObject) error
}
