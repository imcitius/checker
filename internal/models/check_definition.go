package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

	// Polymorphic configuration
	Config CheckConfig `bson:"-" json:"config"`

	// Alert/Actor configuration
	ActorType        string      `bson:"actor_type,omitempty" json:"actor_type,omitempty"`
	AlertType        string      `bson:"alert_type,omitempty" json:"alert_type,omitempty"`
	AlertDestination string      `bson:"alert_destination,omitempty" json:"alert_destination,omitempty"`
	ActorConfig      interface{} `bson:"-" json:"actor_config,omitempty"` // Polymorphic Actor Config
}

// UnmarshalBSON implements a custom BSON unmarshaler to handle polymorphism
func (cd *CheckDefinition) UnmarshalBSON(data []byte) error {
	// 1. Decode common fields into a temporary struct
	type Alias CheckDefinition
	aux := &struct {
		*Alias `bson:",inline"`
	}{
		Alias: (*Alias)(cd),
	}

	if err := bson.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 2. Decode Check Config
	var raw bson.Raw
	if err := bson.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch cd.Type {
	case "http":
		var conf HTTPCheckConfig
		if err := bson.Unmarshal(data, &conf); err != nil {
			return err
		}
		cd.Config = &conf
	case "tcp":
		var conf TCPCheckConfig
		if err := bson.Unmarshal(data, &conf); err != nil {
			return err
		}
		cd.Config = &conf
	case "icmp":
		var conf ICMPCheckConfig
		if err := bson.Unmarshal(data, &conf); err != nil {
			return err
		}
		cd.Config = &conf
	case "passive":
		var conf PassiveCheckConfig
		if err := bson.Unmarshal(data, &conf); err != nil {
			return err
		}
		cd.Config = &conf
	case "mysql_query", "mysql_query_unixtime", "mysql_replication":
		// Handle MySQL flattened structure
		type MySQLFlatAndNested struct {
			Host        string `bson:"host"`
			Port        int    `bson:"port"`
			Timeout     string `bson:"timeout"`
			MySQLNested struct {
				UserName   string   `bson:"username"`
				Password   string   `bson:"password"`
				DBName     string   `bson:"dbname"`
				Query      string   `bson:"query"`
				Response   string   `bson:"response"`
				Difference string   `bson:"difference"`
				TableName  string   `bson:"table_name"`
				Lag        string   `bson:"lag"`
				ServerList []string `bson:"server_list"`
			} `bson:"mysql"`
		}
		var mfn MySQLFlatAndNested
		if err := bson.Unmarshal(data, &mfn); err != nil {
			return err
		}
		cd.Config = &MySQLCheckConfig{
			Host:       mfn.Host,
			Port:       mfn.Port,
			Timeout:    mfn.Timeout,
			UserName:   mfn.MySQLNested.UserName,
			Password:   mfn.MySQLNested.Password,
			DBName:     mfn.MySQLNested.DBName,
			Query:      mfn.MySQLNested.Query,
			Response:   mfn.MySQLNested.Response,
			Difference: mfn.MySQLNested.Difference,
			TableName:  mfn.MySQLNested.TableName,
			Lag:        mfn.MySQLNested.Lag,
			ServerList: mfn.MySQLNested.ServerList,
		}

	case "pgsql_query", "pgsql_query_unixtime", "pgsql_query_timestamp", "pgsql_replication", "pgsql_replication_status":
		type PgSQLFlatAndNested struct {
			Host        string `bson:"host"`
			Port        int    `bson:"port"`
			Timeout     string `bson:"timeout"`
			PgSQLNested struct {
				UserName         string   `bson:"username"`
				Password         string   `bson:"password"`
				DBName           string   `bson:"dbname"`
				SSLMode          string   `bson:"sslmode"`
				Query            string   `bson:"query"`
				Response         string   `bson:"response"`
				Difference       string   `bson:"difference"`
				TableName        string   `bson:"table_name"`
				Lag              string   `bson:"lag"`
				ServerList       []string `bson:"server_list"`
				AnalyticReplicas []string `bson:"analytic_replicas"`
			} `bson:"pgsql"`
		}
		var pfn PgSQLFlatAndNested
		if err := bson.Unmarshal(data, &pfn); err != nil {
			return err
		}
		cd.Config = &PostgreSQLCheckConfig{
			Host:             pfn.Host,
			Port:             pfn.Port,
			Timeout:          pfn.Timeout,
			UserName:         pfn.PgSQLNested.UserName,
			Password:         pfn.PgSQLNested.Password,
			DBName:           pfn.PgSQLNested.DBName,
			SSLMode:          pfn.PgSQLNested.SSLMode,
			Query:            pfn.PgSQLNested.Query,
			Response:         pfn.PgSQLNested.Response,
			Difference:       pfn.PgSQLNested.Difference,
			TableName:        pfn.PgSQLNested.TableName,
			Lag:              pfn.PgSQLNested.Lag,
			ServerList:       pfn.PgSQLNested.ServerList,
			AnalyticReplicas: pfn.PgSQLNested.AnalyticReplicas,
		}
	}

	// 3. Decode Actor Config
	if cd.ActorType == "webhook" {
		type WebhookNested struct {
			Webhook struct {
				URL     string            `bson:"url"`
				Method  string            `bson:"method"`
				Payload string            `bson:"payload"`
				Headers map[string]string `bson:"headers"`
			} `bson:"webhook"`
		}
		var wn WebhookNested
		if err := bson.Unmarshal(data, &wn); err != nil {
			return err
		}
		cd.ActorConfig = &WebhookConfig{
			URL:     wn.Webhook.URL,
			Method:  wn.Webhook.Method,
			Payload: wn.Webhook.Payload,
			Headers: wn.Webhook.Headers,
		}
	}

	return nil
}

// MarshalBSON implements custom BSON marshaling to flatten the structure
func (cd *CheckDefinition) MarshalBSON() ([]byte, error) {
	doc := bson.M{
		"_id":               cd.ID,
		"uuid":              cd.UUID,
		"name":              cd.Name,
		"project":           cd.Project,
		"group_name":        cd.GroupName,
		"type":              cd.Type,
		"description":       cd.Description,
		"enabled":           cd.Enabled,
		"created_at":        cd.CreatedAt,
		"updated_at":        cd.UpdatedAt,
		"last_run":          cd.LastRun,
		"is_healthy":        cd.IsHealthy,
		"last_message":      cd.LastMessage,
		"last_alert_sent":   cd.LastAlertSent,
		"duration":          cd.Duration,
		"actor_type":        cd.ActorType,
		"alert_type":        cd.AlertType,
		"alert_destination": cd.AlertDestination,
	}

	// Flatten Check Config
	if cd.Config != nil {
		switch c := cd.Config.(type) {
		case *HTTPCheckConfig:
			doc["url"] = c.URL
			doc["timeout"] = c.Timeout
			doc["answer"] = c.Answer
			doc["answer_present"] = c.AnswerPresent
			doc["code"] = c.Code
			doc["headers"] = c.Headers
			doc["cookies"] = c.Cookies
			doc["skip_check_ssl"] = c.SkipCheckSSL
			doc["ssl_expiration_period"] = c.SSLExpirationPeriod
			doc["stop_follow_redirects"] = c.StopFollowRedirects
			doc["auth"] = c.Auth
		case *TCPCheckConfig:
			doc["host"] = c.Host
			doc["port"] = c.Port
			doc["timeout"] = c.Timeout
		case *ICMPCheckConfig:
			doc["host"] = c.Host
			doc["count"] = c.Count
			doc["timeout"] = c.Timeout
		case *PassiveCheckConfig:
			doc["timeout"] = c.Timeout
		case *MySQLCheckConfig:
			doc["host"] = c.Host
			doc["port"] = c.Port
			doc["timeout"] = c.Timeout
			doc["mysql"] = bson.M{
				"username":    c.UserName,
				"password":    c.Password,
				"dbname":      c.DBName,
				"query":       c.Query,
				"response":    c.Response,
				"difference":  c.Difference,
				"table_name":  c.TableName,
				"lag":         c.Lag,
				"server_list": c.ServerList,
			}
		case *PostgreSQLCheckConfig:
			doc["host"] = c.Host
			doc["port"] = c.Port
			doc["timeout"] = c.Timeout
			doc["pgsql"] = bson.M{
				"username":          c.UserName,
				"password":          c.Password,
				"dbname":            c.DBName,
				"sslmode":           c.SSLMode,
				"query":             c.Query,
				"response":          c.Response,
				"difference":        c.Difference,
				"table_name":        c.TableName,
				"lag":               c.Lag,
				"server_list":       c.ServerList,
				"analytic_replicas": c.AnalyticReplicas,
			}
		}
	}

	// Flatten Actor Config
	if cd.ActorConfig != nil {
		switch c := cd.ActorConfig.(type) {
		case *WebhookConfig:
			doc["webhook"] = bson.M{
				"url":     c.URL,
				"method":  c.Method,
				"payload": c.Payload,
				"headers": c.Headers,
			}
		}
	}

	return bson.Marshal(doc)
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

	// Config fields — shared
	URL     string `json:"url,omitempty"`
	Timeout string `json:"timeout,omitempty"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`

	// HTTP advanced fields
	Answer              string              `json:"answer,omitempty"`
	AnswerPresent       bool                `json:"answer_present,omitempty"`
	Code                []int               `json:"code,omitempty"`
	Headers             []map[string]string `json:"headers,omitempty"`
	Cookies             []map[string]string `json:"cookies,omitempty"`
	SkipCheckSSL        bool                `json:"skip_check_ssl,omitempty"`
	SSLExpirationPeriod string              `json:"ssl_expiration_period,omitempty"`
	StopFollowRedirects bool                `json:"stop_follow_redirects,omitempty"`
	Auth                struct {
		User     string `json:"user,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"auth,omitempty"`

	// ICMP fields
	Count int `json:"count,omitempty"`

	// Database config fields
	PgSQL struct {
		UserName         string   `json:"username,omitempty"`
		Password         string   `json:"password,omitempty"`
		DBName           string   `json:"dbname,omitempty"`
		SSLMode          string   `json:"sslmode,omitempty"`
		Query            string   `json:"query,omitempty"`
		Response         string   `json:"response,omitempty"`
		Difference       string   `json:"difference,omitempty"`
		TableName        string   `json:"table_name,omitempty"`
		Lag              string   `json:"lag,omitempty"`
		ServerList       []string `json:"server_list,omitempty"`
		AnalyticReplicas []string `json:"analytic_replicas,omitempty"`
	} `json:"pgsql,omitempty"`
	MySQL struct {
		UserName   string   `json:"username,omitempty"`
		Password   string   `json:"password,omitempty"`
		DBName     string   `json:"dbname,omitempty"`
		Query      string   `json:"query,omitempty"`
		Response   string   `json:"response,omitempty"`
		Difference string   `json:"difference,omitempty"`
		TableName  string   `json:"table_name,omitempty"`
		Lag        string   `json:"lag,omitempty"`
		ServerList []string `json:"server_list,omitempty"`
	} `json:"mysql,omitempty"`

	ActorType        string `json:"actor_type,omitempty"`
	AlertType        string `json:"alert_type,omitempty"`
	AlertDestination string `json:"alert_destination,omitempty"`
}
