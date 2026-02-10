package migrations

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Migration_20260208_032830_drew struct{}

func (m *Migration_20260208_032830_drew) Version() string {
	return "20260208_032830_drew"
}

func (m *Migration_20260208_032830_drew) Description() string {
	return "Drew"
}

func (m *Migration_20260208_032830_drew) Up(ctx context.Context, db *mongo.Database) error {
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": "drews"})
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
		if err := db.CreateCollection(ctx, "drews", opts); err != nil {
			return err
		}
	}
	collection := db.Collection("drews")

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
			Options: options.Index().SetName("idx_drews_email_unique").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetName("idx_drews_username_unique").SetUnique(true),
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

func (m *Migration_20260208_032830_drew) Down(ctx context.Context, db *mongo.Database) error {
	slog.Info("Running migration DOWN", "version", m.Version())
	collection := db.Collection("drews")

	for _, idx := range []string{"idx_drews_email_unique", "idx_drews_status_created_at"} {
		if err := collection.Indexes().DropOne(ctx, idx); err != nil {
			return fmt.Errorf("drop index %s failed: %w", idx, err)
		}
	}

	fmt.Printf("Migration %s: %s - DOWN\\n", m.Version(), m.Description())
	return nil
}
