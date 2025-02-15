package store

import (
	"my/checker/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IStore interface {
	Init(connectionString string) (IStore, error)
	Disconnect() error

	GetCheckObjectByUUid(string) (models.DbCheckObject, error)
	BulkWriteChecks([]mongo.WriteModel, *options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	BulkWriteAlerts([]mongo.WriteModel, *options.BulkWriteOptions) (*mongo.BulkWriteResult, error)

	//GetData() (interface{}, error)
	//Save() error
	//Load() error

	GetAllChecks() ([]models.DbCheckObject, error)
	GetAllAlerts() (*MessagesContextStorage, error)
	UpdateChecks() error
	UpdateAlerts() error
	//UpdateChecksByCollection(checks checks.TChecksCollection) error
	UpdateSingleCheck(check models.DbCheckObject) error
}
