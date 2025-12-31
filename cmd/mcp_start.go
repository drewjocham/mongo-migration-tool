package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var mcpStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the Node.js MCP server",
	Long: `Starts the Node.js MCP server as a child process. 
			This is intended for use within the Docker environment where the 
			Node.js implementation is located at /app/mcp-server.`,
	RunE: runMCPStart,
}

func init() {
	mcpCmd.AddCommand(mcpStartCmd)
}

func runMCPStart(cmd *cobra.Command, _ []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	serverDir := "/app/mcp-server"
	serverScript := "index.js"
	serverPath := filepath.Join(serverDir, serverScript)

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		slog.Error("MCP server script not found", "path", serverPath)
		return fmt.Errorf("node script not found at %s: ensure you are in the correct Docker container", serverPath)
	}

	nodeCmd := exec.CommandContext(cmd.Context(), "node", serverScript)
	nodeCmd.Dir = serverDir

	nodeCmd.Stdout = os.Stdout
	nodeCmd.Stderr = os.Stderr
	nodeCmd.Stdin = os.Stdin

	slog.Info("Starting Node.js MCP child process", "script", serverPath)

	if err := nodeCmd.Run(); err != nil {
		slog.Error("Node.js MCP server exited with error", "error", err)
		return err
	}

	return nil
}
