package models

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCheckDefinition_BSON(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond) // Truncate for comparison

	tests := []struct {
		name     string
		checkDef CheckDefinition
	}{
		{
			name: "HTTP Check",
			checkDef: CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "http-uuid",
				Name:      "HTTP Test",
				Type:      "http",
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
				Config: &HTTPCheckConfig{
					URL:           "https://example.com",
					Timeout:       "10s",
					AnswerPresent: true,
					Answer:        "ok",
					Code:          []int{200, 201},
				},
			},
		},
		{
			name: "TCP Check",
			checkDef: CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "tcp-uuid",
				Name:      "TCP Test",
				Type:      "tcp",
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
				Config: &TCPCheckConfig{
					Host:    "example.com",
					Port:    8080,
					Timeout: "5s",
				},
			},
		},
		{
			name: "MySQL Check",
			checkDef: CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "mysql-uuid",
				Name:      "MySQL Test",
				Type:      "mysql_query",
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
				Config: &MySQLCheckConfig{
					Host:     "db.example.com",
					Port:     3306,
					UserName: "root",
					Password: "password",
					DBName:   "test_db",
					Query:    "SELECT 1",
				},
			},
		},
		{
			name: "PostgreSQL Check",
			checkDef: CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "pgsql-uuid",
				Name:      "PgSQL Test",
				Type:      "pgsql_query",
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
				Config: &PostgreSQLCheckConfig{
					Host:     "db.example.com",
					Port:     5432,
					UserName: "postgres",
					Password: "password",
					DBName:   "test_db",
					Query:    "SELECT 1",
				},
			},
		},
		{
			name: "Webhook Actor",
			checkDef: CheckDefinition{
				ID:        primitive.NewObjectID(),
				UUID:      "webhook-uuid",
				Name:      "Webhook Test",
				Type:      "http",
				ActorType: "webhook",
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
				Config: &HTTPCheckConfig{ // Needs a check config too
					URL: "https://example.com",
				},
				ActorConfig: &WebhookConfig{
					URL:     "https://hook.example.com",
					Method:  "POST",
					Payload: `{"text": "hello"}`,
					Headers: map[string]string{"Content-Type": "application/json"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := bson.Marshal(&tt.checkDef)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal
			var result CheckDefinition
			if err := bson.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Compare Basic Fields
			if result.UUID != tt.checkDef.UUID {
				t.Errorf("UUID mismatch: got %v, want %v", result.UUID, tt.checkDef.UUID)
			}
			if result.Type != tt.checkDef.Type {
				t.Errorf("Type mismatch: got %v, want %v", result.Type, tt.checkDef.Type)
			}

			// Compare Config
			if !reflect.DeepEqual(result.Config, tt.checkDef.Config) {
				t.Errorf("Config mismatch:\nGot:  %+v\nWant: %+v", result.Config, tt.checkDef.Config)
			}

			// Compare Actor Config
			if !reflect.DeepEqual(result.ActorConfig, tt.checkDef.ActorConfig) {
				t.Errorf("ActorConfig mismatch:\nGot:  %+v\nWant: %+v", result.ActorConfig, tt.checkDef.ActorConfig)
			}
		})
	}
}
