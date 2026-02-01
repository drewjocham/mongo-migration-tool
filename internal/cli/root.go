package cli

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

type contextKey string

const (
	ctxEngineKey contextKey = "engine"
	ctxConfigKey contextKey = "config"
	ctxCancelKey contextKey = "rootCancel"

	annotationOffline = "offline"

	maxPingRetries = 5
	pingRetryDelay = 2 * time.Second
	pingTimeout    = 2 * time.Second
)

var (
	configFile string
	debugMode  bool
	logLevel   = new(slog.LevelVar)
)

var rootCmd = &cobra.Command{
	Use:               "mongo-essential",
	Short:             "Essential MongoDB toolkit",
	PersistentPreRunE: setupDependencies,
	PersistentPostRun: teardown,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file (optional)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")

	rootCmd.AddCommand(
		upCmd,
		downCmd,
		forceCmd,
		mcpCmd,
		mcpConfigCmd,
		createCmd,
		statusCmd,
		versionCmd,
	)

	mcpStartCmd.Flags().StringVar(&configFile, "config", "", "Path to config file (optional)")
	mcpCmd.Flags().StringVar(&configFile, "config", "", "The recommended config to apply to your AI client.")
	upCmd.Flags().StringVar(&upTarget, "target", "", "Target version to migrate up to")
	downCmd.Flags().StringVarP(&downTargetVersion, "target", "t", "", "Version to roll back to (exclusive)")
	downCmd.Flags().BoolVarP(&downConfirm, "yes", "y", false, "Confirm the action without prompting")
	forceCmd.Flags().BoolVarP(&forceYes, "yes", "y", false, "Confirm without prompting")
	statusCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
}

func setupDependencies(cmd *cobra.Command, _ []string) error {
	initLogging()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), ctxConfigKey, cfg)

	if isOffline(cmd) {
		cmd.SetContext(ctx)
		return nil
	}

	engine, cancel, err := initEngine(ctx, cfg)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, ctxEngineKey, engine)
	ctx = context.WithValue(ctx, ctxCancelKey, cancel)

	cmd.SetContext(ctx)
	return nil
}

func teardown(cmd *cobra.Command, _ []string) {
	if cancel, ok := cmd.Context().Value(ctxCancelKey).(context.CancelFunc); ok {
		cancel()
	}
}

func isOffline(cmd *cobra.Command) bool {
	if cmd.Annotations[annotationOffline] == "true" {
		return true
	}
	return cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "create"
}

func initLogging() {
	if debugMode {
		logLevel.Set(slog.LevelDebug)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))
}

func loadConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if configFile != "" {
		cfg, err = config.Load(configFile)
	} else {
		cfg, err = config.Load(".env", ".env.local")
	}

	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}
	return cfg, nil
}

func initEngine(ctx context.Context, cfg *config.Config) (*migration.Engine, context.CancelFunc, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)

	opts := options.Client().
		ApplyURI(cfg.GetConnectionString()).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize))

	if cfg.SSLEnabled {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.SSLInsecure})
	}

	client, err := mongo.Connect(timeoutCtx, opts)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("mongo connect failed: %w", err)
	}

	if err := retryPing(timeoutCtx, client); err != nil {
		cancel()
		return nil, nil, err
	}

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	slog.Debug("Engine initialized", "db", cfg.Database)
	return engine, cancel, nil
}

func retryPing(ctx context.Context, client *mongo.Client) error {
	for i := 1; i <= maxPingRetries; i++ {
		pCtx, pCancel := context.WithTimeout(ctx, pingTimeout)
		err := client.Ping(pCtx, nil)
		pCancel()

		if err == nil {
			return nil
		}

		slog.Warn("MongoDB not ready, retrying...", "attempt", i, "err", err)
		if i < maxPingRetries {
			time.Sleep(pingRetryDelay)
		}
	}
	return fmt.Errorf("mongodb unreachable after %d attempts", maxPingRetries)
}

func Execute() error {
	return rootCmd.Execute()
}
