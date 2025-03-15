package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"checker/internal/config"
	"checker/internal/models"
)

const checkDefinitionsCollection = "check_definitions"

// GetAllCheckDefinitions retrieves all check definitions from the database
func (m *MongoDB) GetAllCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	collection := m.Collection(checkDefinitionsCollection)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query check definitions: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.CheckDefinition
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode check definitions: %w", err)
	}

	return results, nil
}

// GetEnabledCheckDefinitions retrieves only enabled check definitions
func (m *MongoDB) GetEnabledCheckDefinitions(ctx context.Context) ([]models.CheckDefinition, error) {
	collection := m.Collection(checkDefinitionsCollection)

	cursor, err := collection.Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled check definitions: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.CheckDefinition
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode check definitions: %w", err)
	}

	return results, nil
}

// GetCheckDefinitionByUUID retrieves a single check definition by UUID
func (m *MongoDB) GetCheckDefinitionByUUID(ctx context.Context, uuid string) (models.CheckDefinition, error) {
	collection := m.Collection(checkDefinitionsCollection)

	var result models.CheckDefinition
	err := collection.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, fmt.Errorf("check definition not found: %w", err)
		}
		return result, fmt.Errorf("error retrieving check definition: %w", err)
	}

	return result, nil
}

// CreateCheckDefinition creates a new check definition
func (m *MongoDB) CreateCheckDefinition(ctx context.Context, def models.CheckDefinition) (string, error) {
	collection := m.Collection(checkDefinitionsCollection)

	// Generate UUID if not provided
	if def.UUID == "" {
		def.UUID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	def.CreatedAt = now
	def.UpdatedAt = now

	result, err := collection.InsertOne(ctx, def)
	if err != nil {
		return "", fmt.Errorf("failed to insert check definition: %w", err)
	}

	// Get the inserted ID
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return def.UUID, nil
	}

	return id.Hex(), nil
}

// UpdateCheckDefinition updates an existing check definition
func (m *MongoDB) UpdateCheckDefinition(ctx context.Context, def models.CheckDefinition) error {
	collection := m.Collection(checkDefinitionsCollection)

	// Set update timestamp
	def.UpdatedAt = time.Now()

	filter := bson.M{"uuid": def.UUID}
	update := bson.M{"$set": def}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update check definition: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("check definition with UUID %s not found", def.UUID)
	}

	return nil
}

// DeleteCheckDefinition deletes a check definition by UUID
func (m *MongoDB) DeleteCheckDefinition(ctx context.Context, uuid string) error {
	collection := m.Collection(checkDefinitionsCollection)

	result, err := collection.DeleteOne(ctx, bson.M{"uuid": uuid})
	if err != nil {
		return fmt.Errorf("failed to delete check definition: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("check definition with UUID %s not found", uuid)
	}

	logrus.Infof("Check definition %s deleted", uuid)
	return nil
}

// ToggleCheckDefinition enables or disables a check definition
func (m *MongoDB) ToggleCheckDefinition(ctx context.Context, uuid string, enabled bool) error {
	collection := m.Collection(checkDefinitionsCollection)

	filter := bson.M{"uuid": uuid}
	update := bson.M{
		"$set": bson.M{
			"enabled":    enabled,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to toggle check definition: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("check definition with UUID %s not found", uuid)
	}

	logrus.Infof("Check definition %s toggled to enabled=%v", uuid, enabled)
	return nil
}

// GetAllProjects returns a list of all unique project names
func (m *MongoDB) GetAllProjects(ctx context.Context) ([]string, error) {
	collection := m.Collection(checkDefinitionsCollection)

	// Use MongoDB aggregation to get distinct project values
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{{"_id", "$project"}}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID string `bson:"_id"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}

	projects := make([]string, 0, len(results))
	for _, result := range results {
		projects = append(projects, result.ID)
	}

	return projects, nil
}

// GetAllCheckTypes returns a list of all unique check types
func (m *MongoDB) GetAllCheckTypes(ctx context.Context) ([]string, error) {
	collection := m.Collection(checkDefinitionsCollection)

	// Use MongoDB aggregation to get distinct check type values
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{{"_id", "$type"}}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to query check types: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID string `bson:"_id"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode check types: %w", err)
	}

	types := make([]string, 0, len(results))
	for _, result := range results {
		types = append(types, result.ID)
	}

	return types, nil
}

// ConvertConfigToCheckDefinitions is a utility function to migrate from configuration file to database
func (m *MongoDB) ConvertConfigToCheckDefinitions(ctx context.Context, config *config.Config) error {
	batch := make([]interface{}, 0)
	now := time.Now()

	// Process each project and its health checks
	for projectName, project := range config.Projects {
		for groupName, group := range project.HealthChecks {
			for checkName, check := range group.Checks {
				// Create check definition from config
				checkDef := models.CheckDefinition{
					UUID:        check.UUID,
					Name:        checkName,
					Project:     projectName,
					GroupName:   groupName,
					Type:        check.Type,
					Description: check.Description,
					Enabled:     true, // Default to enabled
					CreatedAt:   now,
					UpdatedAt:   now,
					Duration:    check.Parameters.Duration.String(),

					// Copy check-specific fields
					URL:                 check.URL,
					Timeout:             check.Timeout,
					Answer:              check.Answer,
					AnswerPresent:       check.AnswerPresent,
					Code:                check.Code,
					Host:                check.Host,
					Port:                check.Port,
					Count:               check.Count,
					Headers:             check.Headers,
					SkipCheckSSL:        check.SkipCheckSSL,
					SSLExpirationPeriod: check.SSLExpirationPeriod,
					StopFollowRedirects: check.StopFollowRedirects,
					ActorType:           check.ActorType,
					AlertType:           check.AlertType,
				}

				// Copy auth if present
				if check.Auth.User != "" || check.Auth.Password != "" {
					checkDef.Auth.User = check.Auth.User
					checkDef.Auth.Password = check.Auth.Password
				}

				// Generate UUID if not present
				if checkDef.UUID == "" {
					checkDef.UUID = uuid.New().String()
				}

				batch = append(batch, checkDef)
			}
		}
	}

	// If we have definitions to insert
	if len(batch) > 0 {
		collection := m.Collection(checkDefinitionsCollection)
		opts := options.InsertMany().SetOrdered(false)

		_, err := collection.InsertMany(ctx, batch, opts)
		if err != nil {
			return fmt.Errorf("failed to insert check definitions: %w", err)
		}

		logrus.Infof("Successfully imported %d check definitions from config", len(batch))
	}

	return nil
}
