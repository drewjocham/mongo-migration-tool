package main

// This package provides an example of how to use the mongo-migration tool.

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-migration-tool/config"
	_ "github.com/jocham/mongo-migration-tool/examples/examplemigrations"
	"github.com/jocham/mongo-migration-tool/migration"
)

const (
	minArgs           = 2
	connectionTimeout = 10 * time.Second
	statusLineLength  = 80
)

func main() {
	if len(os.Args) < minArgs {
		fmt.Println("Usage: go run main.go [up|down|status]")
		os.Exit(1) //nolint:gocritic // exit is intended here
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Print("Failed to load configuration: ", err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}

	client, db, err := connectToMongoDB(context.Background(), cfg)
	if err != nil {
		log.Print("Failed to connect to MongoDB: ", err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}
	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			log.Printf("Error disconnecting from MongoDB: %v", disconnectErr)
		}
	}()

	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	if err := executeCommand(context.Background(), command, engine); err != nil {
		log.Print("Command failed: ", err)
		os.Exit(1) //nolint:gocritic // exit is intended here
	}
}

func connectToMongoDB(ctx context.Context, cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			log.Printf("Warning: failed to disconnect client after ping failure: %v", disconnectErr)
		}
		return nil, nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	return client, client.Database(cfg.Database), nil
}

func executeCommand(ctx context.Context, command string, engine *migration.Engine) error {
	switch command {
	case "up":
		return runMigrationsUp(ctx, engine)
	case "down":
		return runMigrationsDown(ctx, engine)
	case "status":
		return showMigrationStatus(ctx, engine)
	default:
		return fmt.Errorf("unknown command: %s\nAvailable commands: up, down, status", command)
	}
}

func runMigrationsUp(ctx context.Context, engine *migration.Engine) error {
	fmt.Println("Running migrations up...")

	status, err := engine.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	for _, s := range status {
		if !s.Applied {
			fmt.Printf("Running migration: %s - %s\n", s.Version, s.Description)
			if err := engine.Up(ctx, s.Version); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", s.Version, err)
			}
			fmt.Printf("✅ Completed migration: %s\n", s.Version)
		}
	}

	fmt.Println("All migrations completed!")
	return nil
}

func runMigrationsDown(ctx context.Context, engine *migration.Engine) error {
	fmt.Println("Rolling back last migration...")

	status, err := engine.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	var lastApplied *migration.MigrationStatus
	for i := len(status) - 1; i >= 0; i-- {
		if status[i].Applied {
			lastApplied = &status[i]
			break
		}
	}

	if lastApplied == nil {
		fmt.Println("No migrations to roll back")
		return nil
	}

	fmt.Printf("Rolling back migration: %s - %s\n", lastApplied.Version, lastApplied.Description)
	if err := engine.Down(ctx, lastApplied.Version); err != nil {
		return fmt.Errorf("failed to roll back migration %s: %w", lastApplied.Version, err)
	}

	fmt.Printf("✅ Rolled back migration: %s\n", lastApplied.Version)
	return nil
}

func showMigrationStatus(ctx context.Context, engine *migration.Engine) error {
	fmt.Println("Migration Status:")
	fmt.Println(strings.Repeat("-", statusLineLength))

	status, err := engine.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("%-20s %-10s %-20s %s\n", "Version", "Applied", "Applied At", "Description")
	fmt.Println(strings.Repeat("-", statusLineLength))

	for _, s := range status {
		appliedStr := "❌ No"
		appliedAtStr := "Never"

		if s.Applied {
			appliedStr = "✅ Yes"
			if s.AppliedAt != nil {
				appliedAtStr = s.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Printf("%-20s %-10s %-20s %s\n", s.Version, appliedStr, appliedAtStr, s.Description)
	}

	return nil
}
