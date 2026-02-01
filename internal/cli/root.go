package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/logging"
	"github.com/drewjocham/mongo-migration-tool/migration"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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
	logFile    string // Variable for the log file path

	appVersion = "dev"
	commit     = "none"
	date       = "unknown"
)

var rootCmd = &cobra.Command{
	Use:               "mongo-essential",
	Short:             "Essential MongoDB toolkit",
	Version:           fmt.Sprintf("%s (commit: %s, build date: %s)", appVersion, commit, date),
	PersistentPreRunE: setupDependencies,
	PersistentPostRun: teardown,
}

func init() { //nolint:gochecknoinits // cobra init function
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file (optional)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "Path to write logs to a file instead of stderr")

	rootCmd.AddCommand(
		newUpCmd(),
		newDownCmd(),
		newForceCmd(),
		newStatusCmd(),
		newCreateCmd(),
		NewMCPCmd(),
		versionCmd,
	)
}

func setupDependencies(cmd *cobra.Command, _ []string) error {
	if _, err := logging.New(debugMode, logFile); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer zap.S().Sync()

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
	switch cmd.Name() {
	case "help", "version", "create", "config":
		return true
	default:
		return false
	}
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

	zap.S().Debugw("Engine initialized", "db", cfg.Database)
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

		zap.S().Warnw("MongoDB not ready, retrying...", "attempt", i, "error", err)
		if i < maxPingRetries {
			time.Sleep(pingRetryDelay)
		}
	}
	return fmt.Errorf("mongodb unreachable after %d attempts", maxPingRetries)
}

func Execute() error {
	return rootCmd.Execute()
}
