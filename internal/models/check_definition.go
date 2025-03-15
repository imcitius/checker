package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckDefinition stores all configuration for a check
// This replaces the config file-based definition
type CheckDefinition struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UUID        string             `bson:"uuid" json:"uuid"`
	Name        string             `bson:"name" json:"name"`
	Project     string             `bson:"project" json:"project"`
	GroupName   string             `bson:"group_name" json:"group_name"`
	Type        string             `bson:"type" json:"type"`
	Description string             `bson:"description" json:"description"`
	Enabled     bool               `bson:"enabled" json:"enabled"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`

	// Status fields
	LastRun       time.Time `bson:"last_run" json:"last_run"`
	IsHealthy     bool      `bson:"is_healthy" json:"is_healthy"`
	LastMessage   string    `bson:"last_message" json:"last_message"`
	LastAlertSent time.Time `bson:"last_alert_sent" json:"last_alert_sent"`

	// Scheduling
	Duration string `bson:"duration" json:"duration"` // e.g. "1m", "5m", "1h"

	// Check-specific configuration
	URL                 string              `bson:"url,omitempty" json:"url,omitempty"`
	Timeout             string              `bson:"timeout,omitempty" json:"timeout,omitempty"`
	Answer              string              `bson:"answer,omitempty" json:"answer,omitempty"`
	AnswerPresent       bool                `bson:"answer_present,omitempty" json:"answer_present,omitempty"`
	Code                []int               `bson:"code,omitempty" json:"code,omitempty"`
	Host                string              `bson:"host,omitempty" json:"host,omitempty"`
	Port                int                 `bson:"port,omitempty" json:"port,omitempty"`
	Count               int                 `bson:"count,omitempty" json:"count,omitempty"`
	Headers             []map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Cookies             []map[string]string `bson:"cookies,omitempty" json:"cookies,omitempty"`
	SkipCheckSSL        bool                `bson:"skip_check_ssl,omitempty" json:"skip_check_ssl,omitempty"`
	SSLExpirationPeriod string              `bson:"ssl_expiration_period,omitempty" json:"ssl_expiration_period,omitempty"`
	StopFollowRedirects bool                `bson:"stop_follow_redirects,omitempty" json:"stop_follow_redirects,omitempty"`

	// Alert configuration
	ActorType        string `bson:"actor_type,omitempty" json:"actor_type,omitempty"`
	AlertType        string `bson:"alert_type,omitempty" json:"alert_type,omitempty"`
	AlertDestination string `bson:"alert_destination,omitempty" json:"alert_destination,omitempty"`

	// Authentication
	Auth struct {
		User     string `bson:"user,omitempty" json:"user,omitempty"`
		Password string `bson:"password,omitempty" json:"password,omitempty"`
	} `bson:"auth,omitempty" json:"auth,omitempty"`
}

// CheckDefinitionViewModel is used for the web UI
type CheckDefinitionViewModel struct {
	ID          string `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Project     string `json:"project"`
	GroupName   string `json:"group_name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Duration    string `json:"duration"`

	// Config fields (simplified for UI)
	URL              string `json:"url,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"`
	ActorType        string `json:"actor_type,omitempty"`
	AlertType        string `json:"alert_type,omitempty"`
	AlertDestination string `json:"alert_destination,omitempty"`
}

// ToConfigFormat converts the database model to the format expected by the checker factory
func (cd *CheckDefinition) ToConfigFormat() map[string]interface{} {
	// This function converts the database model to the format expected by the checker factory
	config := map[string]interface{}{
		"type":                  cd.Type,
		"url":                   cd.URL,
		"timeout":               cd.Timeout,
		"answer":                cd.Answer,
		"answer_present":        cd.AnswerPresent,
		"code":                  cd.Code,
		"host":                  cd.Host,
		"port":                  cd.Port,
		"count":                 cd.Count,
		"headers":               cd.Headers,
		"cookies":               cd.Cookies,
		"skip_check_ssl":        cd.SkipCheckSSL,
		"ssl_expiration_period": cd.SSLExpirationPeriod,
		"stop_follow_redirects": cd.StopFollowRedirects,
		"actor_type":            cd.ActorType,
		"alert_type":            cd.AlertType,
		"auth": map[string]string{
			"user":     cd.Auth.User,
			"password": cd.Auth.Password,
		},
	}

	return config
}
