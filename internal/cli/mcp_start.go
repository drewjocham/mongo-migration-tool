package cli

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

func runMCPStart(cmd *cobra.Command, _ []string) error {
	const (
		serverDir    = "/app/mcp-server"
		serverScript = "index.js"
	)

	serverPath := filepath.Join(serverDir, serverScript)

	if _, err := os.Stat(serverPath); err != nil {
		return fmt.Errorf("MCP server script missing: %w", err)
	}

	nodeCmd := exec.CommandContext(cmd.Context(), "node", serverScript)
	nodeCmd.Dir = serverDir

	nodeCmd.Stdout = cmd.OutOrStdout()
	nodeCmd.Stderr = cmd.ErrOrStderr()
	nodeCmd.Stdin = cmd.InOrStdin()

	slog.Info("Launching MCP server", "dir", serverDir, "script", serverScript)

	if err := nodeCmd.Run(); err != nil {
		return fmt.Errorf("node process exited: %w", err)
	}

	return nil
}
