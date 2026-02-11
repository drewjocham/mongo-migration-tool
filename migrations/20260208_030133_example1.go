package migrations

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Migration_20260208_030133_example1 struct{}

func (m *Migration_20260208_030133_example1) Version() string {
	return "20260208_030133_example1"
}

func (m *Migration_20260208_030133_example1) Description() string {
	return "create index on address field in users-again collection"
}

func (m *Migration_20260208_030133_example1) Up(ctx context.Context, db *mongo.Database) error {
	slog.Info("Running migration UP", "version", m.Version())

	coll := db.Collection("users")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "address", Value: 1}},
		Options: options.Index().SetName("idx_address"),
	}

	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (m *Migration_20260208_030133_example1) Down(ctx context.Context, db *mongo.Database) error {
	slog.Info("Running migration DOWN", "version", m.Version())

	coll := db.Collection("users")

	return coll.Indexes().DropOne(ctx, "idx_address")
}
