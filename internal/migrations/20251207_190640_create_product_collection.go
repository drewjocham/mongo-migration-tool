package migrations //nolint:dupl

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Migration20251207_190640CreateProductCollection create-product-collection
type Migration20251207_190640CreateProductCollection struct{}

// Version returns the unique version identifier for this migration
func (m *Migration20251207_190640CreateProductCollection) Version() string {
	return "20251207_190640_create_product_collection"
}

// Description returns a human-readable description of what this migration does
func (m *Migration20251207_190640CreateProductCollection) Description() string {
	return "create-product-collection"
}

// Up executes the migration
func (m *Migration20251207_190640CreateProductCollection) Up(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("product")

	// // Create indexes, insert data, etc.
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "nat_ian", Value: 1}},
		Options: options.Index().SetName("_id_nat_idx"),
	}
	_, err := collection.Indexes().CreateOne(ctx, index)

	fmt.Printf("Migration %s: %s - UP\\n", m.Version(), m.Description())

	return err
}

// Down rolls back the migration
func (m *Migration20251207_190640CreateProductCollection) Down(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("product")
	if err := collection.Indexes().DropOne(ctx, "_id_nat_idx"); err != nil {
		// Log the error but don't fail the migration if the index doesn't exist
		fmt.Printf("Could not drop index _id_nat_idx (it may not exist): %v\n", err)
	}

	fmt.Printf("Migration %s: %s - DOWN\\n", m.Version(), m.Description())

	return nil
}
