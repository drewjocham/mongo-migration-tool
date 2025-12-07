package main

// This package provides a library example for mongo-migration.

// This example shows how to use mongo-migration as a library
// in a standalone application outside of the main project.
//
// To use this in your own project:
// 1. go mod init your-project
// 2. go get github.com/jocham/mongo-migration@latest
// 3. Copy this code and adapt it to your needs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-migration/config"
	"github.com/jocham/mongo-migration/migration"
)

const (
	connectionTimeout = 10 * time.Second
)

// ExampleMigration is a simple migration that can be used
// as a template for your own migrations
type ExampleMigration struct{}

// Version returns the unique version identifier for this migration
func (m *ExampleMigration) Version() string {
	return "20240109_001"
}

// Description returns a human-readable description of what this migration does
func (m *ExampleMigration) Description() string {
	return "Example migration - creates sample_collection with index"
}

// Up executes the migration
func (m *ExampleMigration) Up(
	ctx context.Context, db *mongo.Database,
) error {
	collection := db.Collection("sample_collection")

	// Insert a sample document
	_, err := collection.InsertOne(ctx, bson.M{
		"message":    "Hello from mongo-migration!",
		"created_at": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to insert sample document: %w", err)
	}

	// Create an index
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
		Options: options.Index().
			SetName("idx_sample_created_at"),
		// SetBackground is deprecated in MongoDB 4.2+
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	fmt.Println("✅ Created sample_collection with index")
	return nil
}

// Down rolls back the migration
func (m *ExampleMigration) Down(
	ctx context.Context, db *mongo.Database,
) error {
	// Drop the entire collection
	err := db.Collection("sample_collection").Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop sample_collection: %w", err)
	}

	fmt.Println("✅ Dropped sample_collection")
	return nil
}

func main() {
	fmt.Println("🚀 mongo-migration Standalone Example")
	fmt.Println("=====================================")

	cfg, err := loadConfig()
	if err != nil {
		log.Print(err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}

	client, db, err := connectToMongoDB(context.Background(), cfg)
	if err != nil {
		log.Print(err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}
	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			log.Printf("Error disconnecting from MongoDB: %v", disconnectErr)
		}
	}()

	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	if err := runExampleFlow(context.Background(), engine); err != nil {
		log.Print(err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}

	fmt.Println("\n🎉 Standalone example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Create your own migration structs")
	fmt.Println("- Register them with migration.Register() in an init() function")
	fmt.Println("- Use engine.Up(), engine.Down(), and engine.GetStatus() as needed")
	fmt.Println("- See the documentation for more advanced features")
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load() // Will look for .env file
	if err != nil {
		// Method 2: Create config programmatically (fallback)
		cfg = &config.Config{
			MongoURL:             "mongodb://localhost:27017",
			Database:             "standalone_example",
			MigrationsCollection: "schema_migrations",
		}
		fmt.Println("ℹ️  Using default configuration (no .env file found)")
	} else {
		fmt.Println("ℹ️  Loaded configuration from .env file")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("❌ Configuration validation failed: %w", err)
	}
	return cfg, nil
}

func connectToMongoDB(ctx context.Context, cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	fmt.Printf("🔗 Connecting to MongoDB: %s/%s\n", cfg.MongoURL, cfg.Database)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		return nil, nil, fmt.Errorf("❌ Failed to connect to MongoDB: %w", err)
	}

	// Test connection
	if err = client.Ping(ctx, nil); err != nil {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			log.Printf("Warning: failed to disconnect client after ping failure: %v", disconnectErr)
		}
		return nil, nil, fmt.Errorf("❌ Failed to ping MongoDB: %w", err)
	}

	fmt.Println("✅ Connected to MongoDB successfully")
	return client, client.Database(cfg.Database), nil
}

func runExampleFlow(ctx context.Context, engine *migration.Engine) error {
	// Show current status
	fmt.Println("\n📊 Migration Status:")
	if err := showStatus(ctx, engine); err != nil {
		return fmt.Errorf("❌ Failed to get status: %w", err)
	}

	// Run migrations up
	fmt.Println("\n⬆️  Running migrations up...")
	if err := engine.Up(ctx, ""); err != nil {
		return fmt.Errorf("❌ Migration up failed: %w", err)
	}
	fmt.Println("✅ All migrations applied successfully")

	// Show status again
	fmt.Println("\n📊 Updated Migration Status:")
	if err := showStatus(ctx, engine); err != nil {
		return fmt.Errorf("❌ Failed to get status: %w", err)
	}

	// Demonstrate rollback
	fmt.Println("\n⬇️  Rolling back last migration...")
	status, err := engine.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("❌ Failed to get status: %w", err)
	}

	// Find last applied migration
	var lastApplied *migration.MigrationStatus
	for i := len(status) - 1; i >= 0; i-- {
		if status[i].Applied {
			lastApplied = &status[i]
			break
		}
	}

	if lastApplied != nil {
		if err := engine.Down(ctx, lastApplied.Version); err != nil {
			return fmt.Errorf("❌ Migration down failed: %w", err)
		}
		fmt.Printf("✅ Rolled back migration: %s\n", lastApplied.Version)
	} else {
		fmt.Println("ℹ️  No migrations to roll back")
	}
	return nil
}

func showStatus(ctx context.Context, engine *migration.Engine) error {
	status, err := engine.GetStatus(ctx)
	if err != nil {
		return err
	}

	if len(status) == 0 {
		fmt.Println("   No migrations registered")
		return nil
	}

	for _, s := range status {
		appliedStr := "❌ No"
		if s.Applied {
			appliedStr = "✅ Yes"
		}
		fmt.Printf("   %-15s %-8s %s\n", s.Version, appliedStr, s.Description)
	}

	return nil
}
