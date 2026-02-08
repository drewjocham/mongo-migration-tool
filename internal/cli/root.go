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

const (
	annotationOffline = "offline"
	maxPingRetries    = 5
	pingRetryDelay    = 1 * time.Second
	pingTimeout       = 2 * time.Second
)

var (
	configFile string
	debugMode  bool
	logFile    string
	showConfig bool

	appVersion, commit, date = "dev", "none", "unknown"
	ErrShowConfigDisplayed   = errors.New("configuration displayed")
)

type Dependencies struct {
	Config      *config.Config
	Engine      *migration.Engine
	MongoClient *mongo.Client
}

func Execute() error {
	deps := &Dependencies{}
	return newRootCmd(deps).Execute()
}

func newRootCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mmt",
		Short:   "MongoDB migration toolkit",
		Version: fmt.Sprintf("%s (commit: %s, build date: %s)", appVersion, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupDependencies(cmd, deps)
		},
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			teardown(deps)
		},
		SilenceUsage: true,
	}

	pFlags := cmd.PersistentFlags()
	pFlags.StringVarP(&configFile, "config", "c", "", "Path to config file")
	pFlags.BoolVar(&debugMode, "debug", false, "Enable debug logging")
	pFlags.StringVar(&logFile, "log-file", "", "Path to write logs to a file")
	pFlags.BoolVar(&showConfig, "show-config", false, "Print effective configuration and exit")

	cmd.AddCommand(
		newUpCmd(), newDownCmd(), newForceCmd(), newUnlockCmd(),
		newStatusCmd(), newCreateCmd(), newSchemaCmd(), NewMCPCmd(),
		versionCmd,
	)

	return cmd
}

func setupDependencies(cmd *cobra.Command, deps *Dependencies) error {
	if _, err := logging.New(debugMode, logFile); err != nil {
		return fmt.Errorf("logger init: %w", err)
	}

	cfg, err := loadConfigFromFlags(configFile)
	if err != nil {
		return err
	}
	deps.Config = cfg

	if showConfig {
		if err := renderConfig(cmd.OutOrStdout(), cfg); err != nil {
			return err
		}
		return ErrShowConfigDisplayed
	}

	if isOffline(cmd) {
		return nil
	}

	if err := ensureMigrationsRegistered(); err != nil {
		return err
	}

	engine, client, err := initEngine(cmd.Context(), cfg)
	if err != nil {
		return err
	}
	deps.Engine = engine
	deps.MongoClient = client

	return nil
}

func initEngine(ctx context.Context, cfg *config.Config) (*migration.Engine, *mongo.Client, error) {
	connCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(cfg.GetConnectionString()).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize))

	if cfg.SSLEnabled {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.SSLInsecure})
	}

	client, err := mongo.Connect(connCtx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := runPingWithRetry(connCtx, client); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection, migration.RegisteredMigrations())
	return engine, client, nil
}

func runPingWithRetry(ctx context.Context, client *mongo.Client) error {
	for i := 1; i <= maxPingRetries; i++ {
		pCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err := client.Ping(pCtx, nil)
		cancel()

		if err == nil {
			return nil
		}

		zap.S().Warnf("MongoDB attempt %d/%d failed: %v", i, maxPingRetries, err)

		if i == maxPingRetries {
			return fmt.Errorf("mongodb unreachable after %d attempts: %w", maxPingRetries, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pingRetryDelay):
			continue
		}
	}
	return nil
}

func isOffline(cmd *cobra.Command) bool {
	if cmd.Annotations[annotationOffline] == "true" {
		return true
	}
	offlineNames := map[string]bool{"help": true, "version": true, "create": true, "config": true}
	return offlineNames[cmd.Name()]
}

func teardown(deps *Dependencies) {
	if deps.MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := deps.MongoClient.Disconnect(ctx); err != nil {
			zap.S().Warnf("failed to disconnect mongo client: %v", err)
		}
	}
	_ = zap.L().Sync()
}

func loadConfigFromFlags(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.Load(".env", ".env.local")
}

func ensureMigrationsRegistered() error {
	if len(migration.RegisteredMigrations()) == 0 {
		return errors.New("no migrations registered: import your migrations package")
	}
	return nil
}
