package checks

import (
	"my/checker/store"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateChecksByCollectioninDB(checks []TCheckWithDuration) error {
	var models []mongo.WriteModel
	opts := options.BulkWrite().SetOrdered(false)

	for _, c := range checks {
		details := configurer.GetCheckDetails(c.Check.GetUUID())

		models = append(models,
			mongo.NewUpdateOneModel().SetFilter(bson.D{{Key: "UUID", Value: details.UUID}}).
				SetUpdate(bson.D{{Key: "$set", Value: bson.D{
					{Key: "Project", Value: details.Project},
					{Key: "Healthcheck", Value: details.Healthcheck},
					{Key: "Name", Value: details.Name},
					{Key: "UUID", Value: details.UUID},
					{Key: "LastResult", Value: details.LastResult},
					{Key: "LastExec", Value: details.LastExec},
					{Key: "LastPing", Value: details.LastPing},
					{Key: "Enabled", Value: details.Enabled},
				}}}).SetUpsert(true),
		)
	}

	results, err := store.Store.BulkWriteChecks(models, opts)

	if err != nil {
		logger.Errorf("Error while inserting checks to MongoDB: %s", err.Error())
	}

	// When you run this file for the first time, it should print:
	// Number of documents replaced or modified: 2
	logger.Infof("MongoDB updated (checks status), replaced or modified, upserted: %d, %d", results.ModifiedCount, results.UpsertedCount)

	return err
}
