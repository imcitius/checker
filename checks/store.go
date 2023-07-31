package checks

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"my/checker/store"
)

func UpdateChecksByCollection(checks []TCheckWithDuration) error {

	models := []mongo.WriteModel{}
	opts := options.BulkWrite().SetOrdered(false)
	for _, c := range checks {
		details := configurer.GetCheckDetails(c.Check.GetUUID())

		models = append(models,
			mongo.NewUpdateOneModel().SetFilter(bson.D{{"UUid", details.UUid}}).
				SetUpdate(bson.D{{"$set", bson.D{
					{"Project", details.Project},
					{"Healthcheck", details.Healthcheck},
					{"Name", details.Name},
					{"UUid", details.UUid},
					{"LastResult", details.LastResult},
					{"LastExec", details.LastExec},
					{"LastPing", details.LastPing}},
				}}).SetUpsert(true),
		)
	}

	results, err := store.Store.BulkWrite(models, opts)

	if err != nil {
		logger.Errorf("Error while inserting checks to MongoDB: %s", err.Error())
	}

	// When you run this file for the first time, it should print:
	// Number of documents replaced or modified: 2
	logger.Infof("MongoDB updated, replaced or modified, upserted: %d, %d", results.ModifiedCount, results.UpsertedCount)

	return err
}
