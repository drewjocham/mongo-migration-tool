package examplemigrations

// This package contains example migration definitions.

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TransformUserDataMigration demonstrates data transformation operations
type TransformUserDataMigration struct{}

// Version returns the unique version identifier for this migration
func (m *TransformUserDataMigration) Version() string {
	return "example_20240101_002"
}

// Description returns a human-readable description of what this migration does
func (m *TransformUserDataMigration) Description() string {
	return "Transform user data: normalize email case, add full_name field, and update timestamps"
}

// Up executes the migration
func (m *TransformUserDataMigration) Up(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("users")

	// Find all users and transform their data
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to find users: %w", err)
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			log.Printf("Error closing cursor: %v", closeErr)
		}
	}()

	for cursor.Next(ctx) {
		var user bson.M
		if err := cursor.Decode(&user); err != nil {
			return fmt.Errorf("failed to decode user: %w", err)
		}

		if err := m.transformSingleUser(ctx, collection, user); err != nil {
			return fmt.Errorf("failed to transform user %v: %w", user["_id"], err)
		}
	}

	return cursor.Err()
}

func (m *TransformUserDataMigration) transformSingleUser(
	ctx context.Context, collection *mongo.Collection, user bson.M,
) error {
	update := bson.M{"$set": bson.M{}}

	// Normalize email to lowercase
	if email, exists := user["email"].(string); exists {
		update["$set"].(bson.M)["email"] = strings.ToLower(email)
	}

	// Create full_name from first_name and last_name
	firstName, hasFirst := user["first_name"].(string)
	lastName, hasLast := user["last_name"].(string)
	if hasFirst || hasLast {
		fullName := strings.TrimSpace(firstName + " " + lastName)
		if fullName != "" {
			update["$set"].(bson.M)["full_name"] = fullName
		}
	}

	// Add updated_at timestamp if it doesn't exist
	if _, hasUpdated := user["updated_at"]; !hasUpdated {
		update["$set"].(bson.M)["updated_at"] = time.Now()
	}

	// Only update if we have changes to make
	if len(update["$set"].(bson.M)) > 0 {
		filter := bson.M{"_id": user["_id"]}
		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}
	return nil
}

// Down rolls back the migration
func (m *TransformUserDataMigration) Down(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("users")

	// Remove the full_name field that we added
	update := bson.M{
		"$unset": bson.M{
			"full_name": "",
		},
	}

	_, err := collection.UpdateMany(ctx, bson.D{}, update)
	return err
}
