package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
)

const connectionTimeout = 10 * time.Second

type ExampleMigration struct{}

func (m *ExampleMigration) Version() string { return "20240109_001" }

func (m *ExampleMigration) Description() string {
	return "Example migration - creates sample_collection with index"
}

func (m *ExampleMigration) Up(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("sample_collection")

	_, err := collection.InsertOne(ctx, bson.M{
		"message":    "Hello from mongo-migration!",
		"created_at": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: -1}},
		Options: options.Index().SetName("idx_sample_created_at"),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	fmt.Println("Created sample_collection with index")
	return nil
}

func (m *ExampleMigration) Down(ctx context.Context, db *mongo.Database) error {
	err := db.Collection("sample_collection").Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	fmt.Println("Dropped sample_collection")
	return nil
}

func main() {
	fmt.Println("mongo-migration Library Example")
	fmt.Println("=====================================")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, db, err := connectToMongoDB(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("failed to disconnect MongoDB client: %v", err)
		}
	}()

	migration.MustRegister(&ExampleMigration{})

	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	if err := runExampleFlow(ctx, engine); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nExample completed successfully!")
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{
			MongoURL:             "mongodb://localhost:27017",
			Database:             "standalone_example",
			MigrationsCollection: "schema_migrations",
		}
		fmt.Println("Using default configuration")
	} else {
		fmt.Println("Loaded configuration from .env file")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return cfg, nil
}

func connectToMongoDB(ctx context.Context, cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	connCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	fmt.Printf("Connecting to: %s/%s\n", cfg.MongoURL, cfg.Database)
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %w", err)
	}

	if err = client.Ping(connCtx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping failed: %w", err)
	}

	fmt.Println("Connected successfully")
	return client, client.Database(cfg.Database), nil
}

func runExampleFlow(ctx context.Context, engine *migration.Engine) error {
	fmt.Println("\nInitial Status:")
	if err := showStatus(ctx, engine); err != nil {
		return err
	}

	fmt.Println("\nMigrating Up...")
	if err := engine.Up(ctx, ""); err != nil {
		return err
	}

	fmt.Println("\nUpdated Status:")
	if err := showStatus(ctx, engine); err != nil {
		return err
	}

	fmt.Println("\nRolling Back...")
	status, err := engine.GetStatus(ctx)
	if err != nil {
		return err
	}

	for i := len(status) - 1; i >= 0; i-- {
		if status[i].Applied {
			if err := engine.Down(ctx, status[i].Version); err != nil {
				return err
			}
			fmt.Printf("Rolled back: %s\n", status[i].Version)
			break
		}
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
		applied := "No"
		if s.Applied {
			applied = "Yes"
		}
		fmt.Printf("   %-15s %-8s %s\n", s.Version, applied, s.Description)
	}
	return nil
}
