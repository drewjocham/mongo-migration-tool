package cli

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/drewjocham/mongo-migration-tool/internal/config"
	logging "github.com/drewjocham/mongo-migration-tool/internal/log"
	"io"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

type Services struct {
	Config      *config.Config
	Engine      *migration.Engine
	MongoClient *mongo.Client
}

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "mongo-tool",
		Aliases: []string{"mt", "mmt"},
		Short:   "MongoDB migration toolkit",
		Version: fmt.Sprintf("%s (commit: %s, build date: %s)", appVersion, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := logging.New(debugMode, logFile); err != nil {
				return err
			}

			s, err := bootstrap(cmd.Context(), configFile, showConfig, cmd.OutOrStdout(), isOffline(cmd))
			if err != nil {
				return err
			}
			if s != nil {
				ctx := context.WithValue(cmd.Context(), ctxServicesKey, s)
				if s.Config != nil {
					ctx = context.WithValue(ctx, ctxConfigKey, s.Config)
				}
				if s.Engine != nil {
					ctx = context.WithValue(ctx, ctxEngineKey, s.Engine)
				}
				cmd.SetContext(ctx)
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			if s, ok := cmd.Context().Value(ctxServicesKey).(*Services); ok {
				teardown(s)
			}
		},
		SilenceUsage: true,
	}

	p := cmd.PersistentFlags()
	p.StringVarP(&configFile, "config", "c", "", "Path to config file")
	p.BoolVar(&debugMode, "debug", false, "Enable debug logging")
	p.StringVar(&logFile, "log-file", "", "Path to write logs to a file")
	p.BoolVar(&showConfig, "show-config", false, "Print effective configuration and exit")

	cmd.AddCommand(
		newUpCmd(), newDownCmd(), newForceCmd(), newUnlockCmd(),
		newStatusCmd(), newOpslogCmd(),
		NewOplogCmd(),
		NewDBCmd(),
		newParseCmd(), newValidateCmd(),
		newCreateCmd(), newSchemaCmd(), NewMCPCmd(),
		versionCmd,
	)

	return cmd
}

func bootstrap(ctx context.Context, path string, show bool, out io.Writer, offline bool) (*Services, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}

	if show {
		if err := renderConfig(out, cfg); err != nil {
			return nil, err
		}
		return nil, ErrShowConfigDisplayed
	}

	if offline {
		return &Services{Config: cfg}, nil
	}

	if err := validateRegistry(); err != nil {
		return nil, err
	}

	client, err := dial(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Services{
		Config:      cfg,
		MongoClient: client,
		Engine: migration.NewEngine(client.Database(cfg.Database), cfg.MigrationsCollection,
			migration.RegisteredMigrations()),
	}, nil
}

func dial(ctx context.Context, cfg *config.Config) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(cfg.GetConnectionString()).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize))

	if cfg.SSLEnabled {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.SSLInsecure})
	}

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	if err := ping(ctx, client, maxPingRetries); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}

func ping(ctx context.Context, client *mongo.Client, retries int) error {
	for i := 1; i <= retries; i++ {
		pCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err := client.Ping(pCtx, nil)
		cancel()

		if err == nil {
			return nil
		}

		zap.S().Warnf("Attempt %d/%d failed: %v", i, retries, err)

		if i == retries {
			return fmt.Errorf("unreachable: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pingRetryDelay):
		}
	}
	return nil
}

func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.Load(".env", ".env.local")
}

func validateRegistry() error {
	if len(migration.RegisteredMigrations()) == 0 {
		return errors.New("no migrations registered")
	}
	return nil
}

func isOffline(cmd *cobra.Command) bool {
	if cmd.Annotations[annotationOffline] == "true" {
		return true
	}
	offlineCommands := map[string]bool{
		"help":    true,
		"version": true,
		"create":  true,
	}
	return offlineCommands[cmd.Name()]
}

func teardown(s *Services) {
	if s.MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.MongoClient.Disconnect(ctx)
	}
	_ = zap.L().Sync()
}
