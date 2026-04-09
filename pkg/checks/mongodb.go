// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	MongoDBErrEmptyURI = "empty URI"
)

// MongoDBCheck represents a MongoDB connectivity health check.
type MongoDBCheck struct {
	URI     string
	Timeout string
	Logger  *logrus.Entry
}

// Run executes the MongoDB health check by connecting and pinging the server.
func (check *MongoDBCheck) Run() (time.Duration, error) {
	start := time.Now()

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "mongodb")
	}

	errorHeader := fmt.Sprintf("MongoDB check error for URI %s: ", check.URI)

	if check.URI == "" {
		return time.Since(start), errors.New(errorHeader + MongoDBErrEmptyURI)
	}

	// Parse timeout
	connectTimeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), fmt.Errorf("%s%v", errorHeader, err)
	}

	// Create a context with the connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	// Build client options from URI with the connection timeout
	clientOpts := options.Client().
		ApplyURI(check.URI).
		SetConnectTimeout(connectTimeout).
		SetServerSelectionTimeout(connectTimeout)

	check.Logger.Debugf("Connecting to MongoDB with URI: %s", check.URI)

	// Create client
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not create MongoDB client")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			check.Logger.WithError(disconnectErr).Warn("Error disconnecting from MongoDB")
		}
	}()

	// Ping the server
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		check.Logger.WithError(err).Error("Error: Could not ping MongoDB")
		return time.Since(start), fmt.Errorf(errorHeader+"%v", err)
	}

	return time.Since(start), nil
}
