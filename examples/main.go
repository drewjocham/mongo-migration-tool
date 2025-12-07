package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-migration/config"
	_ "github.com/jocham/mongo-migration/examples/examplemigrations"
	"github.com/jocham/mongo-migration/migration"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [up|down|status]")
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(cfg.Database)

	// Create migration engine
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	switch command {
	case "up":
		err = runMigrationsUp(ctx, engine)
	case "down":
		err = runMigrationsDown(ctx, engine)
	case "status":
		err = showMigrationStatus(ctx, engine)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, down, status")
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
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

	// Find the last applied migration
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
	fmt.Println(strings.Repeat("-", 80))

	status, err := engine.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("%-20s %-10s %-20s %s\n", "Version", "Applied", "Applied At", "Description")
	fmt.Println(strings.Repeat("-", 80))

	for _, s := range status {
		appliedStr := "❌ No"
		appliedAt := "Never"

		if s.Applied {
			appliedStr = "✅ Yes"
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Printf("%-20s %-10s %-20s %s\n", s.Version, appliedStr, appliedAt, s.Description)
	}

	return nil
}
