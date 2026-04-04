package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUser_JSONRoundTrip(t *testing.T) {
	id := primitive.NewObjectID()
	u := User{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	data, err := json.Marshal(u)
	assert.NoError(t, err)

	var got User
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, u.Name, got.Name)
	assert.Equal(t, u.Email, got.Email)
}

func TestUser_BSONRoundTrip(t *testing.T) {
	id := primitive.NewObjectID()
	u := User{
		ID:    id,
		Name:  "Jane Smith",
		Email: "jane@example.com",
	}

	data, err := bson.Marshal(u)
	assert.NoError(t, err)

	var got User
	err = bson.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Name, got.Name)
	assert.Equal(t, u.Email, got.Email)
}

func TestUser_ZeroValue(t *testing.T) {
	var u User
	assert.True(t, u.ID.IsZero())
	assert.Empty(t, u.Name)
	assert.Empty(t, u.Email)
}

func TestUser_EmptyFields(t *testing.T) {
	u := User{
		ID:    primitive.NewObjectID(),
		Name:  "",
		Email: "",
	}
	assert.Empty(t, u.Name)
	assert.Empty(t, u.Email)
	assert.False(t, u.ID.IsZero())
}
