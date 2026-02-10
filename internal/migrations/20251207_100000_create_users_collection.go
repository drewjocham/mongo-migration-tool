package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CreateUsersCollectionMigration creates the users collection
type CreateUsersCollectionMigration struct{}

// Version returns the unique version identifier for this migration
func (m *CreateUsersCollectionMigration) Version() string {
	return "20251207_100000"
}

// Description returns a human-readable description of what this migration does
func (m *CreateUsersCollectionMigration) Description() string {
	return "Create users collection with schema validation and indexes"
}

// Up executes the migration
func (m *CreateUsersCollectionMigration) Up(
	ctx context.Context, db *mongo.Database,
) error {
	// If the collection already exists (e.g., created implicitly by earlier migrations), skip creation.
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": "users"})
	if err != nil {
		return err
	}
	if len(collections) == 0 {
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
	}
	collection := db.Collection("users")

	// Build a set of existing index names to avoid conflicts.
	existing := map[string]struct{}{}
	cursor, err := collection.Indexes().List(ctx)
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err == nil {
				if name, ok := doc["name"].(string); ok {
					existing[name] = struct{}{}
				}
			}
		}
	}
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("idx_users_email_unique").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetName("idx_users_username_unique").SetUnique(true),
		},
	}

	var toCreate []mongo.IndexModel
	for _, idx := range indexes {
		name, ok := indexName(idx.Options)
		if !ok {
			toCreate = append(toCreate, idx)
			continue
		}
		if _, exists := existing[name]; !exists {
			toCreate = append(toCreate, idx)
		}
	}

	if len(toCreate) == 0 {
		return nil
	}

	_, err = collection.Indexes().CreateMany(ctx, toCreate)
	return err
}

// Down rolls back the migration
func (m *CreateUsersCollectionMigration) Down(
	ctx context.Context, db *mongo.Database,
) error {
	return db.Collection("users").Drop(ctx)
}
