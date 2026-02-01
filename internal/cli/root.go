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
	pingRetryDelay = 1 * time.Second
	pingTimeout    = 2 * time.Second
)

var (
	configFile string
	debugMode  bool
	logFile    string

	appVersion, commit, date = "dev", "none", "unknown"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mongo-essential",
		Short:             "Essential MongoDB toolkit",
		Version:           fmt.Sprintf("%s (commit: %s, build date: %s)", appVersion, commit, date),
		PersistentPreRunE: setupDependencies,
		PersistentPostRun: teardown,
		SilenceUsage:      true, // Prevents printing help on actual execution errors
	}

	pFlags := cmd.PersistentFlags()
	pFlags.StringVarP(&configFile, "config", "c", "", "Path to config file")
	pFlags.BoolVar(&debugMode, "debug", false, "Enable debug logging")
	pFlags.StringVar(&logFile, "log-file", "", "Path to write logs to a file")

	cmd.AddCommand(
		newUpCmd(), newDownCmd(), newForceCmd(),
		newStatusCmd(), newCreateCmd(), NewMCPCmd(),
		versionCmd,
	)

	return cmd
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

func setupDependencies(cmd *cobra.Command, _ []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	debug, _ := cmd.Flags().GetBool("debug")
	logPath, _ := cmd.Flags().GetString("log-file")

	if _, err := logging.New(debug, logPath); err != nil {
		return fmt.Errorf("logger init: %w", err)
	}
	var cfg *config.Config
	var err error
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.Load(".env", ".env.local")
	}
	cfg, err = config.Load()
	if err != nil {
		return err
	}

	if isOffline(cmd) {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxConfigKey, cfg))
		return nil
	}

	engine, cancel, err := initEngine(cmd.Context(), cfg)
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), ctxConfigKey, cfg)
	ctx = context.WithValue(ctx, ctxEngineKey, engine)
	ctx = context.WithValue(ctx, ctxCancelKey, cancel)

	cmd.SetContext(ctx)
	return nil
}

func isOffline(cmd *cobra.Command) bool {
	if cmd.Annotations[annotationOffline] == "true" {
		return true
	}
	offlineNames := map[string]bool{"help": true, "version": true, "create": true, "config": true}
	return offlineNames[cmd.Name()]
}

func initEngine(ctx context.Context, cfg *config.Config) (*migration.Engine, context.CancelFunc, error) {
	connCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)

	opts := options.Client().
		ApplyURI(cfg.GetConnectionString()).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize))

	if cfg.SSLEnabled {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.SSLInsecure})
	}

	client, err := mongo.Connect(connCtx, opts)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := retryPing(connCtx, client, 5); err != nil {
		_ = client.Disconnect(context.Background())
		cancel()
		return nil, nil, err
	}

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())

	return engine, cancel, nil
}

func retryPing(ctx context.Context, client *mongo.Client, attempt int) error {
	pCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	err := client.Ping(pCtx, nil)
	cancel()

	// Success
	if err == nil {
		return nil
	}

	if attempt >= maxPingRetries {
		return fmt.Errorf("mongodb unreachable after %d attempts: %w", maxPingRetries, err)
	}

	zap.S().Warnf("MongoDB attempt %d/%d failed: %v", attempt, maxPingRetries, err)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pingRetryDelay):
		return retryPing(ctx, client, attempt+1)
	}
}

func teardown(cmd *cobra.Command, _ []string) {
	if cancel, ok := cmd.Context().Value(ctxCancelKey).(context.CancelFunc); ok {
		cancel()
	}
	_ = zap.L().Sync()
}

func Execute() error {
	return newRootCmd().Execute()
}
