package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/drewjocham/mongo-migration-tool/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	configFile string
	debugMode  bool
	cfg        *config.Config
	db         *mongo.Database
	engine     *migration.Engine

	logLevel   = new(slog.LevelVar)
	rootCancel context.CancelFunc
)

var rootCmd = &cobra.Command{
	Use:   "mongo-essential",
	Short: "Essential MongoDB toolkit with migrations and AI-powered analysis",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if debugMode {
			logLevel.Set(slog.LevelDebug)
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
		slog.SetDefault(logger)

		if cmd.Name() == "version" {
			return nil
		}

		var err error
		if configFile != "" {
			cfg, err = config.Load(configFile)
		} else {
			cfg, err = config.Load(".env", ".env.local")
		}
		if err != nil {
			return fmt.Errorf("config load failed: %w", err)
		}

		slog.Debug("Config loaded", "db", cfg.Database, "user", cfg.Username)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
		rootCancel = cancel

		clientOpts := options.Client().
			ApplyURI(cfg.GetConnectionString()).
			SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
			SetMinPoolSize(uint64(cfg.MinPoolSize))

		if cfg.SSLEnabled {
			clientOpts.SetTLSConfig(&tls.Config{
				InsecureSkipVerify: cfg.SSLInsecure,
			})
		}

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			return fmt.Errorf("mongo connect failed: %w", err)
		}

		if err := retryPing(ctx, client); err != nil {
			return err
		}

		db = client.Database(cfg.Database)

		migrations := migration.RegisteredMigrations()
		engine = migration.NewEngine(db, cfg.MigrationsCollection, migrations)

		slog.Info("Engine initialized", "registered_migrations", len(migrations))

		// Pass the database/engine context down to subcommands
		cmd.SetContext(ctx)
		return nil
	},
}

func retryPing(ctx context.Context, client *mongo.Client) error {
	const maxRetries = 5
	const delay = 2 * time.Second

	for i := 1; i <= maxRetries; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := client.Ping(pingCtx, nil)
		cancel()

		if err == nil {
			slog.Debug("MongoDB connection verified")
			return nil
		}

		slog.Warn("MongoDB not ready, retrying...", "attempt", i, "max", maxRetries, "error", err)
		if i < maxRetries {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("could not reach MongoDB after %d attempts", maxRetries)
}

func SetupRootCommand() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug (verbose) logging")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (default .env)")
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		if rootCancel != nil {
			rootCancel()
			rootCancel = nil
		}
	}

	// Subcommands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(forceCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
