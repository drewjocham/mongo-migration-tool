package migrations

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Migration_20251231_192427_add_user_indexes add_user_indexes
type Migration_20251231_192427_add_user_indexes struct{}

// Version returns the unique version identifier for this migration
func (m *Migration_20251231_192427_add_user_indexes) Version() string {
	return "20251231_192427_add_user_indexes"
}

// Description returns a human-readable description of what this migration does
func (m *Migration_20251231_192427_add_user_indexes) Description() string {
	return "add_user_indexes"
}

// Up executes the migration
func (m *Migration_20251231_192427_add_user_indexes) Up(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().
				SetName("idx_users_email_unique").
				SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_users_status_created_at"),
		},
	}

	if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("create indexes failed: %w", err)
	}

	fmt.Printf("Migration %s: %s - UP\\n", m.Version(), m.Description())
	return nil
}

// Down rolls back the migration
func (m *Migration_20251231_192427_add_user_indexes) Down(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")

	for _, idx := range []string{"idx_users_email_unique", "idx_users_status_created_at"} {
		if err := collection.Indexes().DropOne(ctx, idx); err != nil {
			return fmt.Errorf("drop index %s failed: %w", idx, err)
		}
	}

	fmt.Printf("Migration %s: %s - DOWN\\n", m.Version(), m.Description())
	return nil
}
