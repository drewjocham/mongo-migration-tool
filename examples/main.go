package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"text/tabwriter"

	_ "github.com/drewjocham/mongo-migration-tool/examples/examplemigrations"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: go run %s [up|down|status]\n", os.Args[0])
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("failed to disconnect MongoDB client: %v", err)
		}
	}()

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	switch cmd := os.Args[1]; cmd {
	case "up":
		fmt.Println("Applying pending migrations...")
		err = engine.Up(ctx, "")
	case "down":
		fmt.Println("Rolling back last migration...")
		err = engine.Down(ctx, "")
	case "status":
		err = printStatus(ctx, engine)
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func printStatus(ctx context.Context, e *migration.Engine) error {
	stats, err := e.GetStatus(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tDESCRIPTION")
	fmt.Fprintln(w, "-------\t-----\t----------\t-----------")

	for _, s := range stats {
		statusIcon := "⏳ Pending"
		appliedAt := "-"

		if s.Applied {
			statusIcon = "✅ Applied"
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04")
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Version, statusIcon, appliedAt, s.Description)
	}

	return nil
}
