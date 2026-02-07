package cli

import (
	"context"
	"crypto/tls"
	"errors"
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
	ctxEngineKey      contextKey = "engine"
	ctxConfigKey      contextKey = "config"
	ctxCancelKey      contextKey = "rootCancel"
	ctxMongoClientKey contextKey = "mongoClient"

	annotationOffline = "offline"

	maxPingRetries = 5
	pingRetryDelay = 1 * time.Second
	pingTimeout    = 2 * time.Second
)

var (
	configFile string
	debugMode  bool
	logFile    string
	showConfig bool

	appVersion, commit, date = "dev", "none", "unknown"
)

var ErrShowConfigDisplayed = errors.New("configuration displayed")

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mmt",
		Short:             "MongoDB migration toolkit",
		Version:           fmt.Sprintf("%s (commit: %s, build date: %s)", appVersion, commit, date),
		PersistentPreRunE: setupDependencies,
		PersistentPostRun: teardown,
		SilenceUsage:      true, // Prevents printing help on execution errors
	}

	pFlags := cmd.PersistentFlags()
	pFlags.StringVarP(&configFile, "config", "c", "", "Path to config file")
	pFlags.BoolVar(&debugMode, "debug", false, "Enable debug logging")
	pFlags.StringVar(&logFile, "log-file", "", "Path to write logs to a file")
	pFlags.BoolVar(&showConfig, "show-config", false, "Print the effective configuration (with secrets masked) and exit")

	cmd.AddCommand(
		newUpCmd(), newDownCmd(), newForceCmd(), newUnlockCmd(),
		newStatusCmd(), newCreateCmd(), newSchemaCmd(), NewMCPCmd(),
		versionCmd,
	)

	return cmd
}

func setupDependencies(cmd *cobra.Command, _ []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	debug, _ := cmd.Flags().GetBool("debug")
	logPath, _ := cmd.Flags().GetString("log-file")

	if _, err := logging.New(debug, logPath); err != nil {
		return fmt.Errorf("logger init: %w", err)
	}
	cfg, err := loadConfigFromFlags(cfgPath)
	if err != nil {
		return err
	}
	if showConfig {
		if err := renderConfig(cmd.OutOrStdout(), cfg); err != nil {
			return err
		}
		return ErrShowConfigDisplayed
	}

	if isOffline(cmd) {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxConfigKey, cfg))
		return nil
	}
	if err := ensureMigrationsRegistered(); err != nil {
		return err
	}

	engine, client, cancel, err := initEngine(cmd.Context(), cfg)
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), ctxConfigKey, cfg)
	ctx = context.WithValue(ctx, ctxEngineKey, engine)
	ctx = context.WithValue(ctx, ctxCancelKey, cancel)
	ctx = context.WithValue(ctx, ctxMongoClientKey, client)

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

func initEngine(ctx context.Context, cfg *config.Config) (*migration.Engine, *mongo.Client, context.CancelFunc, error) {
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
		return nil, nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := retryPing(connCtx, client, 5); err != nil {
		_ = client.Disconnect(context.Background())
		cancel()
		return nil, nil, nil, err
	}

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())
	return engine, client, cancel, nil
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
	if client, ok := cmd.Context().Value(ctxMongoClientKey).(*mongo.Client); ok {
		if err := client.Disconnect(context.Background()); err != nil {
			zap.S().Warnf("failed to disconnect mongo client: %v", err)
		}
	}
	if err := zap.L().Sync(); err != nil {
		zap.S().Warnf("failed to sync logger: %v", err)
	}
}

func Execute() error {
	return newRootCmd().Execute()
}

func ensureMigrationsRegistered() error {
	if len(migration.RegisteredMigrations()) == 0 {
		return fmt.Errorf("no migrations registered: import your migrations package")
	}
	return nil
}

func loadConfigFromFlags(path string) (*config.Config, error) {
	if path != "" {
		cfg, err := config.Load(path)
		if err != nil {
			return nil, fmt.Errorf("config load failed: %w", err)
		}
		return cfg, nil
	}

	cfg, err := config.Load(".env", ".env.local")
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}
	return cfg, nil
}
