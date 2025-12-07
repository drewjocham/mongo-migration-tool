package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/jocham/mongo-migration/examples/examplemigrations"
	"github.com/jocham/mongo-migration/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI assistant integration",
	Long: `Start the Model Context Protocol (MCP) server that allows AI assistants
like Ollama, Goose, and others to interact with your MongoDB migrations.

The MCP server exposes migration operations as tools that AI assistants can call:
- migration_status: Get migration status
- migration_up: Apply migrations 
- migration_down: Roll back migrations
- migration_create: Create new migration files
- migration_list: List all registered migrations

The server reads from stdin and writes to stdout using JSON-RPC protocol.`,
	Run: runMCP,
}

var mcpWithExamples bool

func setupMCPCommand() {
	mcpCmd.Flags().BoolVar(&mcpWithExamples, "with-examples", false, "Register example migrations with the MCP server")
}

func runMCP(_ *cobra.Command, _ []string) {
	server, err := mcp.NewMCPServer()
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	defer func() {
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Error closing server: %v", closeErr)
		}
	}()

	// Register example migrations if requested
	if mcpWithExamples {
		server.RegisterMigrations(
			&examplemigrations.AddUserIndexesMigration{},
			&examplemigrations.TransformUserDataMigration{},
			&examplemigrations.CreateAuditCollectionMigration{},
		)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("MCP server failed: %v", err) //nolint:gocritic // exit is intended here
	}
}
