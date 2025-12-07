package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateUsersCollectionMigration creates the users collection
type CreateUsersCollectionMigration struct{}

func (m *CreateUsersCollectionMigration) Version() string {
	return "20251207_100000"
}

func (m *CreateUsersCollectionMigration) Description() string {
	return "Create users collection with schema validation and indexes"
}

func (m *CreateUsersCollectionMigration) Up(ctx context.Context, db *mongo.Database) error {
	validator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"email", "username", "password_hash", "created_at", "updated_at"},
			"properties": bson.M{
				"email":         bson.M{"bsonType": "string", "description": "must be a string and is required"},
				"username":      bson.M{"bsonType": "string", "description": "must be a string and is required"},
				"password_hash": bson.M{"bsonType": "string", "description": "must be a string and is required"},
				"first_name":    bson.M{"bsonType": "string"},
				"last_name":     bson.M{"bsonType": "string"},
				"is_active":     bson.M{"bsonType": "bool"},
				"created_at":    bson.M{"bsonType": "date", "description": "must be a date and is required"},
				"updated_at":    bson.M{"bsonType": "date", "description": "must be a date and is required"},
			},
		},
	}

	opts := options.CreateCollection().SetValidator(validator)
	if err := db.CreateCollection(ctx, "users", opts); err != nil {
		return err
	}

	collection := db.Collection("users")
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("idx_users_email").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetName("idx_users_username").SetUnique(true),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (m *CreateUsersCollectionMigration) Down(ctx context.Context, db *mongo.Database) error {
	return db.Collection("users").Drop(ctx)
}
