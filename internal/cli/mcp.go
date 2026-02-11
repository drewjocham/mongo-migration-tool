package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	logging "github.com/drewjocham/mongo-migration-tool/internal/log"
	"github.com/drewjocham/mongo-migration-tool/mcp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

func NewMCPCmd() *cobra.Command {
	var withExamples bool

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for AI assistant integration",
		Long:  "Starts the MCP server using stdin/stdout. Logs are redirected to stderr to avoid protocol corruption.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd, withExamples)
		},
	}
	mcpCmd.Flags().BoolVar(&withExamples, "with-examples", false, "Register example migrations on startup")

	mcpCmd.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Generate MCP configuration JSON",
		Annotations: map[string]string{
			annotationOffline: "true",
		},
		RunE: runMCPConfig,
	})

	return mcpCmd
}

func runMCP(cmd *cobra.Command, withExamples bool) error {
	logger, err := logging.New(false, "")
	if err != nil {
		return fmt.Errorf("failed to re-initialize logger for mcp: %w", err)
	}
	defer func() {
		if syncErr := zap.S().Sync(); syncErr != nil {
			zap.L().Warn("failed to flush sugared logger", zap.Error(syncErr))
		}
	}()

	if withExamples {
		zap.S().Info("Registering example migrations...")
		if err := registerExampleMigrations(); err != nil {
			return fmt.Errorf("failed to register examples: %w", err)
		}
	}
	cfg, err := getConfig(cmd.Context())
	if err != nil {
		return err
	}

	server, err := mcp.NewMCPServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("mcp init failed: %w", err)
	}
	defer server.Close(cmd.Context())

	zap.S().Infow("Starting MCP server", "pid", os.Getpid())

	if err := server.Start(); err != nil && !isClosingError(err) {
		return fmt.Errorf("mcp server failure: %w", err)
	}

	zap.S().Info("MCP server session ended")
	return nil
}

func runMCPConfig(cmd *cobra.Command, _ []string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine path: %w", err)
	}

	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	config := map[string]any{
		"mcpServers": map[string]any{
			"mt": map[string]any{
				"command": exePath,
				"args":    []string{"mcp"},
				"env": map[string]string{
					"MONGO_URI":      getEnv("MONGO_URI", "mongodb://localhost:27017"),
					"MONGO_DATABASE": getEnv("MONGO_DATABASE", "your_database"),
				},
			},
		},
	}

	enc := jsonutil.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(config)
}

func isClosingError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "EOF")
}
