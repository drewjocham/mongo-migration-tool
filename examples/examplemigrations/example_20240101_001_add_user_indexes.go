package examplemigrations

// This package contains example migration definitions.

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AddUserIndexesMigration adds indexes to the users collection
type AddUserIndexesMigration struct{}

// Version returns the unique version identifier for this migration
func (m *AddUserIndexesMigration) Version() string {
	return "example_20240101_001"
}

// Description returns a human-readable description of what this migration does
func (m *AddUserIndexesMigration) Description() string {
	return "Add indexes to users collection for email and created_at fields"
}

// Up executes the migration
func (m *AddUserIndexesMigration) Up(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("users")

	// Create index for email field (unique)
	emailIndexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
		},
		Options: options.Index().
			SetName("idx_users_email_unique").
			SetUnique(true),
		// SetBackground is deprecated in MongoDB 4.2+
	}

	// Create index for created_at field
	createdAtIndexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().
			SetName("idx_users_created_at"),
		// SetBackground is deprecated in MongoDB 4.2+
	}

	// Create compound index for status and created_at
	compoundIndexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().
			SetName("idx_users_status_created_at"),
		// SetBackground is deprecated in MongoDB 4.2+
	}

	// Create all indexes
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		emailIndexModel,
		createdAtIndexModel,
		compoundIndexModel,
	})

	return err
}

// Down rolls back the migration
func (m *AddUserIndexesMigration) Down(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("users")

	// Drop the indexes we created
	indexNames := []string{
		"idx_users_email_unique",
		"idx_users_created_at",
		"idx_users_status_created_at",
	}

	for _, indexName := range indexNames {
		// Ignore errors when dropping indexes - they might not exist
		_ = collection.Indexes().DropOne(ctx, indexName)
	}

	return nil
}
