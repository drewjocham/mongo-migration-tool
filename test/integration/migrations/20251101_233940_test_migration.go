package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TestMigrationMigration Test migration for integration test
type TestMigrationMigration struct{}

func (m *TestMigrationMigration) Version() string {
	return "20251101_233940"
}

func (m *TestMigrationMigration) Description() string {
	return "Test migration for integration test"
}

func (m *TestMigrationMigration) Up(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement migration up logic
	// Example:
	// collection := db.Collection("your_collection")
	// _, err := collection.UpdateMany(ctx, bson.D{}, bson.D{{"$set", bson.D{{"new_field", "default_value"}}}})
	// return err
	return nil
}

func (m *TestMigrationMigration) Down(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement migration down logic (rollback)
	// Example:
	// collection := db.Collection("your_collection")
	// _, err := collection.UpdateMany(ctx, bson.D{}, bson.D{{"$unset", bson.D{{"new_field", ""}}}})
	// return err
	return nil
}
