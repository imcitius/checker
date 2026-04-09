// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCheckImportPayload_JSONRoundTrip(t *testing.T) {
	enabled := true
	payload := CheckImportPayload{
		Project:     "my-service",
		Environment: "prod",
		Source:      "ci",
		Prune:       true,
		Defaults: CheckImportDefaults{
			Duration:  "30s",
			Timeout:   "10s",
			Enabled:   &enabled,
			ActorType: "webhook",
			Severity:  "critical",
		},
		Checks: []CheckImportItem{
			{
				Name:    "API Health",
				Project: "my-service",
				Type:    "http",
				URL:     "https://api.example.com/healthz",
			},
		},
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var got CheckImportPayload
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, payload.Project, got.Project)
	assert.Equal(t, payload.Environment, got.Environment)
	assert.Equal(t, payload.Source, got.Source)
	assert.True(t, got.Prune)
	assert.Equal(t, "30s", got.Defaults.Duration)
	assert.Len(t, got.Checks, 1)
	assert.Equal(t, "API Health", got.Checks[0].Name)
}

func TestCheckImportPayload_YAMLRoundTrip(t *testing.T) {
	yamlData := `
project: my-service
environment: prod
prune: true
defaults:
  duration: 30s
  timeout: 10s
checks:
  - name: API Health
    type: http
    url: https://api.example.com/healthz
    duration: 30s
  - name: DB Check
    type: pgsql_query
    host: db.example.com
    port: 5432
`
	var payload CheckImportPayload
	err := yaml.Unmarshal([]byte(yamlData), &payload)
	assert.NoError(t, err)
	assert.Equal(t, "my-service", payload.Project)
	assert.Equal(t, "prod", payload.Environment)
	assert.True(t, payload.Prune)
	assert.Equal(t, "30s", payload.Defaults.Duration)
	assert.Len(t, payload.Checks, 2)
	assert.Equal(t, "API Health", payload.Checks[0].Name)
	assert.Equal(t, "http", payload.Checks[0].Type)
	assert.Equal(t, "DB Check", payload.Checks[1].Name)
	assert.Equal(t, 5432, payload.Checks[1].Port)
}

func TestCheckImportPayload_ZeroValue(t *testing.T) {
	var p CheckImportPayload
	assert.Empty(t, p.Project)
	assert.Empty(t, p.Environment)
	assert.Empty(t, p.Source)
	assert.False(t, p.Prune)
	assert.Nil(t, p.Checks)
}

func TestCheckImportDefaults_ZeroValue(t *testing.T) {
	var d CheckImportDefaults
	assert.Empty(t, d.Duration)
	assert.Empty(t, d.Timeout)
	assert.Nil(t, d.Enabled)
	assert.Empty(t, d.ActorType)
	assert.Empty(t, d.Severity)
	assert.Nil(t, d.AlertChannels)
	assert.Empty(t, d.ReAlertInterval)
	assert.Equal(t, 0, d.RetryCount)
	assert.Empty(t, d.RetryInterval)
}

func TestCheckImportItem_HTTPCheck(t *testing.T) {
	enabled := true
	answerPresent := true
	item := CheckImportItem{
		Name:          "HTTP Check",
		Project:       "my-service",
		GroupName:     "prod",
		Type:          "http",
		Enabled:       &enabled,
		Duration:      "30s",
		URL:           "https://api.example.com",
		Timeout:       "10s",
		Answer:        "ok",
		AnswerPresent: &answerPresent,
		Code:          []int{200, 201},
		Auth:          &AuthImportConfig{User: "admin", Password: "secret"},
	}

	assert.Equal(t, "HTTP Check", item.Name)
	assert.Equal(t, "http", item.Type)
	assert.NotNil(t, item.AnswerPresent)
	assert.True(t, *item.AnswerPresent)
	assert.Equal(t, []int{200, 201}, item.Code)
	assert.Equal(t, "admin", item.Auth.User)
}

func TestCheckImportItem_DatabaseCheck(t *testing.T) {
	item := CheckImportItem{
		Name: "PG Query",
		Type: "pgsql_query",
		Host: "db.example.com",
		Port: 5432,
		PgSQL: &DBImportConfig{
			UserName:   "postgres",
			DBName:     "mydb",
			Query:      "SELECT 1",
			ServerList: []string{"replica1", "replica2"},
		},
	}
	assert.NotNil(t, item.PgSQL)
	assert.Equal(t, "postgres", item.PgSQL.UserName)
	assert.Len(t, item.PgSQL.ServerList, 2)
}

func TestCheckImportItem_MySQLCheck(t *testing.T) {
	item := CheckImportItem{
		Name: "MySQL Query",
		Type: "mysql_query",
		Host: "mysql.example.com",
		Port: 3306,
		MySQL: &DBImportConfig{
			UserName: "root",
			DBName:   "testdb",
			Query:    "SELECT 1",
		},
	}
	assert.NotNil(t, item.MySQL)
	assert.Equal(t, "root", item.MySQL.UserName)
}

func TestCheckImportItem_ZeroValue(t *testing.T) {
	var item CheckImportItem
	assert.Empty(t, item.Name)
	assert.Empty(t, item.Type)
	assert.Nil(t, item.Enabled)
	assert.Nil(t, item.AnswerPresent)
	assert.Nil(t, item.SkipCheckSSL)
	assert.Nil(t, item.StopFollowRedirects)
	assert.Nil(t, item.Auth)
	assert.Nil(t, item.PgSQL)
	assert.Nil(t, item.MySQL)
}

func TestAuthImportConfig_JSONRoundTrip(t *testing.T) {
	a := AuthImportConfig{User: "admin", Password: "secret"}
	data, err := json.Marshal(a)
	assert.NoError(t, err)

	var got AuthImportConfig
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, "admin", got.User)
	assert.Equal(t, "secret", got.Password)
}

func TestDBImportConfig_JSONRoundTrip(t *testing.T) {
	d := DBImportConfig{
		UserName:   "postgres",
		DBName:     "mydb",
		Query:      "SELECT count(*) FROM users",
		ServerList: []string{"replica1", "replica2"},
	}
	data, err := json.Marshal(d)
	assert.NoError(t, err)

	var got DBImportConfig
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, d.UserName, got.UserName)
	assert.Equal(t, d.DBName, got.DBName)
	assert.Equal(t, d.Query, got.Query)
	assert.Equal(t, d.ServerList, got.ServerList)
}

func TestDBImportConfig_ZeroValue(t *testing.T) {
	var d DBImportConfig
	assert.Empty(t, d.UserName)
	assert.Empty(t, d.DBName)
	assert.Empty(t, d.Query)
	assert.Nil(t, d.ServerList)
}

func TestCheckImportResult_JSONRoundTrip(t *testing.T) {
	result := CheckImportResult{
		Created: []CheckImportResultItem{
			{Name: "API Health", UUID: "uuid-1", Project: "svc"},
		},
		Updated: []CheckImportResultItem{
			{Name: "DB Check", UUID: "uuid-2", Project: "svc"},
		},
		Deleted: []CheckImportResultItem{},
		Errors: []CheckImportError{
			{Name: "Bad Check", Index: 2, Message: "invalid type"},
		},
		Summary: CheckImportSummary{
			Total:   3,
			Created: 1,
			Updated: 1,
			Deleted: 0,
			Errors:  1,
		},
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	var got CheckImportResult
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Len(t, got.Created, 1)
	assert.Len(t, got.Updated, 1)
	assert.Len(t, got.Deleted, 0)
	assert.Len(t, got.Errors, 1)
	assert.Equal(t, 3, got.Summary.Total)
	assert.Equal(t, 1, got.Summary.Created)
	assert.Equal(t, "invalid type", got.Errors[0].Message)
}

func TestCheckImportSummary_ZeroValue(t *testing.T) {
	var s CheckImportSummary
	assert.Equal(t, 0, s.Total)
	assert.Equal(t, 0, s.Created)
	assert.Equal(t, 0, s.Updated)
	assert.Equal(t, 0, s.Deleted)
	assert.Equal(t, 0, s.Errors)
}

func TestCheckImportResultItem_ZeroValue(t *testing.T) {
	var item CheckImportResultItem
	assert.Empty(t, item.Name)
	assert.Empty(t, item.UUID)
	assert.Empty(t, item.Project)
}

func TestCheckImportError_ZeroValue(t *testing.T) {
	var e CheckImportError
	assert.Empty(t, e.Name)
	assert.Equal(t, 0, e.Index)
	assert.Empty(t, e.Message)
}
