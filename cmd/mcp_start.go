package cmd

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var mcpStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the Node.js MCP server",
	Long:  `Starts the Node.js MCP server as a child process.`,
	Run:   runMCPStart,
}

func init() { //nolint:gochecknoinits // init functions are used for migration registration
	mcpCmd.AddCommand(mcpStartCmd)
}

func runMCPStart(_ *cobra.Command, _ []string) {
	// The Node.js server is expected to be in /app/mcp-server in the Docker image
	serverDir := "/app/mcp-server"
	serverScript := "index.js"
	serverPath := filepath.Join(serverDir, serverScript)

	// Check if the server script exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		log.Fatalf("MCP server script not found at %s. Make sure you are running inside the Docker container.",
			serverPath)
	}

	cmd := exec.CommandContext(context.Background(), "node", serverScript)
	cmd.Dir = serverDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Starting Node.js MCP server...")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Node.js MCP server failed: %v", err)
	}
}
