package checks

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMongoDBCheck_Run(t *testing.T) {
	logger := logrus.WithField("test", "TestMongoDBCheck_Run")

	t.Run("Empty URI", func(t *testing.T) {
		check := MongoDBCheck{
			URI:    "",
			Logger: logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), MongoDBErrEmptyURI)
	})

	t.Run("Invalid timeout", func(t *testing.T) {
		check := MongoDBCheck{
			URI:     "mongodb://localhost:27017",
			Timeout: "invalid",
			Logger:  logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timeout")
	})

	t.Run("Invalid URI fails fast", func(t *testing.T) {
		check := MongoDBCheck{
			URI:     "mongodb://invalid-host-that-does-not-exist:27017",
			Timeout: "2s",
			Logger:  logger,
		}
		_, err := check.Run()
		assert.Error(t, err)
	})
}

func TestMongoDBCheck_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	logger := logrus.WithField("test", "TestMongoDBCheck_Integration")

	t.Run("Ping real MongoDB", func(t *testing.T) {
		check := MongoDBCheck{
			URI:     mongoURI,
			Timeout: "5s",
			Logger:  logger,
		}
		elapsed, err := check.Run()
		assert.NoError(t, err)
		assert.Greater(t, elapsed.Nanoseconds(), int64(0))
		t.Logf("MongoDB ping succeeded in %s", elapsed)
	})
}
