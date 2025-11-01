package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-essential/config"
	"github.com/jocham/mongo-essential/migration"
)

var (
	configFile string
	cfg        *config.Config
	db         *mongo.Database
	engine     *migration.Engine
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mongo-essential",
	Short: "Essential MongoDB toolkit with migrations and AI-powered analysis",
	Long: `A MongoDB migration tool that provides version control for your database schema.
	
Features:
- Version-controlled migrations with up/down support
- Migration status tracking
- Rollback capabilities
- Force migration marking
- Integration with existing Go projects`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Skip configuration loading for version command
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
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
		defer cancel()

		clientOpts := options.Client().
			ApplyURI(cfg.GetConnectionString()).
			SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
			SetMinPoolSize(uint64(cfg.MinPoolSize)).
			SetMaxConnIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second).
			SetServerSelectionTimeout(time.Duration(cfg.Timeout) * time.Second).
			SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second)

		if cfg.SSLEnabled {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: cfg.SSLInsecure, // #nosec G402 -- user-configurable for dev environments
			}
			clientOpts.SetTLSConfig(tlsConfig)
		}

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			return fmt.Errorf("failed to connect to MongoDB: %w", err)
		}

		if err = client.Ping(ctx, nil); err != nil {
			return fmt.Errorf("failed to ping MongoDB: %w", err)
		}

		db = client.Database(cfg.Database)
		engine = migration.NewEngine(db, cfg.MigrationsCollection)

		loadMigrations()

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// SetupRootCommand initializes all command flags and subcommands.
func SetupRootCommand() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .env)")

	// Setup subcommands
	setupDownCommand()
	setupUpCommand()
	setupMCPCommand()

	// Add subcommands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(forceCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(mcpCmd)
}

// loadMigrations loads migration files from the migrations directory
func loadMigrations() {
	fmt.Printf("Looking for migrations in: %s\n", cfg.MigrationsPath)

	// Check if migrations directory exists
	if _, err := os.Stat(cfg.MigrationsPath); os.IsNotExist(err) {
		fmt.Printf("Migrations directory does not exist: %s\n", cfg.MigrationsPath)
	}
}
