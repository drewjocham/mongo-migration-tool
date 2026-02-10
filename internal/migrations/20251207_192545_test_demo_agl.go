package migrations //nolint:dupl

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Migration20251207_192545TestDemoAgl test-demo-agl
type Migration20251207_192545TestDemoAgl struct{}

// Version returns the unique version identifier for this migration
func (m *Migration20251207_192545TestDemoAgl) Version() string {
	return "20251207_192545_test_demo_agl"
}

// Description returns a human-readable description of what this migration does
func (m *Migration20251207_192545TestDemoAgl) Description() string {
	return "test-demo-agl"
}

// Up executes the migration
func (m *Migration20251207_192545TestDemoAgl) Up(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("demo")
	//
	// Create indexes, insert data, etc.
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "field_name", Value: 1}},
		Options: options.Index().SetName("ian_nat_idx"),
	}
	_, err := collection.Indexes().CreateOne(ctx, index)

	fmt.Printf("Migration %s: %s - UP\\n", m.Version(), m.Description())
	return err
}

// Down rolls back the migration
func (m *Migration20251207_192545TestDemoAgl) Down(
	ctx context.Context, db *mongo.Database,
) error {
	// Example:
	collection := db.Collection("demo")
	if err := collection.Indexes().DropOne(ctx, "ian_nat_idx"); err != nil {
		// Log the error but don't fail the migration if the index doesn't exist
		fmt.Printf("Could not drop index ian_nat_idx (it may not exist): %v\n", err)
	}

	fmt.Printf("Migration %s: %s - DOWN\\n", m.Version(), m.Description())
	return nil
}
