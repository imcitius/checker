package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckStatus is a record of an individual check's current state.
type CheckStatus struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UUID          string             `bson:"uuid" json:"uuid"`
	Project       string             `bson:"project" json:"project"`
	CheckGroup    string             `bson:"check_group" json:"check_group"`
	CheckName     string             `bson:"check_name" json:"check_name"`
	CheckType     string             `bson:"check_type" json:"check_type"`
	LastRun       time.Time          `bson:"last_run" json:"last_run"`
	IsHealthy     bool               `bson:"is_healthy" json:"is_healthy"`
	Message       string             `bson:"message" json:"message"`
	IsEnabled     bool               `bson:"enabled" json:"enabled"`
	LastAlertSent time.Time          `bson:"last_alert_sent" json:"last_alert_sent"`
	Host          string             `bson:"host" json:"host"`
	Periodicity   string             `bson:"periodicity" json:"periodicity"`
	URL           string             `bson:"url" json:"url"`
	IsSilenced    bool               `bson:"is_silenced" json:"is_silenced"`
}
