package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/drewjocham/mongo-migration-tool/mcp"
	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

var mcpWithExamples bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI assistant integration",
	Long: `Start the Model Context Protocol (MCP) server for AI assistants.
IMPORTANT: This command uses stdin/stdout for communication. 
Logs are automatically redirected to stderr.`,
	RunE: runMCP,
}

func init() {
	mcpCmd.Flags().BoolVar(&mcpWithExamples, "with-examples", false, "Register example migrations")
}

func runMCP(cmd *cobra.Command, _ []string) error {

	if mcpWithExamples {
		slog.Info("Registering example migrations")
		if err := registerExampleMigrations(); err != nil {
			return fmt.Errorf("failed to register example migrations: %w", err)
		}
	}

	server, err := mcp.NewMCPServer()
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	defer func() {
		if err := server.Close(); err != nil {
			slog.Error("Error closing MCP server", "error", err)
		}
	}()

	slog.Info("Starting MCP server", "pid", os.Getpid())

	if err := server.Start(); err != nil {
		if isClosingError(err) {
			slog.Info("MCP server session ended", "reason", "client disconnected")
			return nil
		}
		return fmt.Errorf("mcp server failure: %w", err)
	}

	return nil
}

func isClosingError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		strings.Contains(err.Error(), "EOF")
}
