package main

// This example shows how to use mongo-essential as a library
// in a standalone application outside of the main project.
//
// To use this in your own project:
// 1. go mod init your-project
// 2. go get github.com/jocham/mongo-essential@latest
// 3. Copy this code and adapt it to your needs

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-essential/config"
	"github.com/jocham/mongo-essential/migration"
)

// ExampleMigration is a simple migration that can be used
// as a template for your own migrations
type ExampleMigration struct{}

func (m *ExampleMigration) Version() string {
	return "20240109_001"
}

func (m *ExampleMigration) Description() string {
	return "Example migration - creates sample_collection with index"
}

func (m *ExampleMigration) Up(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("sample_collection")

	// Insert a sample document
	_, err := collection.InsertOne(ctx, bson.M{
		"message":    "Hello from mongo-essential!",
		"created_at": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to insert sample document: %w", err)
	}

	// Create an index
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
		Options: options.Index().
			SetName("idx_sample_created_at").
			SetBackground(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	fmt.Println("✅ Created sample_collection with index")
	return nil
}

func (m *ExampleMigration) Down(ctx context.Context, db *mongo.Database) error {
	// Drop the entire collection
	err := db.Collection("sample_collection").Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop sample_collection: %w", err)
	}

	fmt.Println("✅ Dropped sample_collection")
	return nil
}

func main() {
	fmt.Println("🚀 mongo-essential Standalone Example")
	fmt.Println("=====================================")

	// Method 1: Use environment variables or .env file
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
		log.Fatalf("❌ Configuration validation failed: %v", err)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("🔗 Connecting to MongoDB: %s/%s\n", cfg.MongoURL, cfg.Database)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Test connection
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("❌ Failed to ping MongoDB: %v", err)
	}

	fmt.Println("✅ Connected to MongoDB successfully")

	// Create migration engine
	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	// Show current status
	fmt.Println("\n📊 Migration Status:")
	if err := showStatus(ctx, engine); err != nil {
		log.Fatalf("❌ Failed to get status: %v", err)
	}

	// Run migrations up
	fmt.Println("\n⬆️  Running migrations up...")
	if err := engine.Up(ctx, ""); err != nil {
		log.Fatalf("❌ Migration up failed: %v", err)
	}
	fmt.Println("✅ All migrations applied successfully")

	// Show status again
	fmt.Println("\n📊 Updated Migration Status:")
	if err := showStatus(ctx, engine); err != nil {
		log.Fatalf("❌ Failed to get status: %v", err)
	}

	// Demonstrate rollback
	fmt.Println("\n⬇️  Rolling back last migration...")
	status, err := engine.GetStatus(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get status: %v", err)
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
			log.Fatalf("❌ Migration down failed: %v", err)
		}
		fmt.Printf("✅ Rolled back migration: %s\n", lastApplied.Version)
	} else {
		fmt.Println("ℹ️  No migrations to roll back")
	}

	fmt.Println("\n🎉 Standalone example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Create your own migration structs")
	fmt.Println("- Register them with migration.Register() in an init() function")
	fmt.Println("- Use engine.Up(), engine.Down(), and engine.GetStatus() as needed")
	fmt.Println("- See the documentation for more advanced features")
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
