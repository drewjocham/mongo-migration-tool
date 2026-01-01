package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/jocham/mongo-migration-tool/mcp"
	_ "github.com/jocham/mongo-migration-tool/migrations"
)

var (
	mcpWithExamples bool

	mcpCmd = func() *cobra.Command {
		cmd := &cobra.Command{
			Use:   "mcp",
			Short: "Start MCP server for AI assistant integration",
			Long: `Start the Model Context Protocol (MCP) server for AI assistants.
IMPORTANT: This command uses stdin/stdout for communication. 
Logs are automatically redirected to stderr.`,
			Run: runMCP,
		}

		cmd.Flags().BoolVar(&mcpWithExamples, "with-examples", false, "Register example migrations with the MCP server")

		return cmd
	}()
)

func runMCP(cmd *cobra.Command, _ []string) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if mcpWithExamples {
		slog.Info("Registering example migrations")
		if err := registerExampleMigrations(); err != nil {
			slog.Error("Failed to register example migrations", "error", err)
			os.Exit(1)
		}
	}

	server, err := mcp.NewMCPServer()
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := server.Close(); err != nil {
			slog.Error("Error closing MCP server", "error", err)
		}
	}()

	slog.Info("Starting MCP server", "pid", os.Getpid())

	if err := server.Start(); err != nil {
		slog.Error("MCP server execution failed", "error", err)
		os.Exit(1)
	}
}
