//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/drewjocham/mongo-migration-tool/examples/examplemigrations"
	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const connectionTimeout = 10 * time.Second

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [up|down|status]")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	command := os.Args[1]
	if err := execute(ctx, command, engine); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func execute(ctx context.Context, cmd string, e *migration.Engine) error {
	switch cmd {
	case "up":
		fmt.Println("Running pending migrations...")
		return e.Up(ctx, "")
	case "down":
		fmt.Println("Rolling back last migration...")
		return e.Down(ctx, "")
	case "status":
		return showStatus(ctx, e)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func showStatus(ctx context.Context, e *migration.Engine) error {
	stats, err := e.GetStatus(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tDESCRIPTION")
	fmt.Fprintln(w, "-------\t-----\t----------\t-----------")

	for _, s := range stats {
		state := "Pending"
		appliedAt := "-"

		if s.Applied {
			state = "Applied"
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04")
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Version, state, appliedAt, s.Description)
	}

	return w.Flush()
}
