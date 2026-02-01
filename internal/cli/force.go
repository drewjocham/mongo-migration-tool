package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

var forceYes bool

var forceCmd = &cobra.Command{
	Use:   "force [version]",
	Short: "Force mark a migration as applied without running it",
	Args:  cobra.ExactArgs(1),
	RunE:  runForce,
}

func runForce(cmd *cobra.Command, args []string) error {
	version := args[0]

	if !forceYes && !confirmForce(cmd, version) {
		slog.Info("Operation cancelled")
		return nil
	}

	engine, err := getEngine(cmd.Context())
	if err != nil {
		return err
	}

	// Assuming migration.Engine has a Force method
	if err := engine.Force(cmd.Context(), version); err != nil {
		return fmt.Errorf("failed to force mark %s: %w", version, err)
	}

	slog.Info("Migration force marked successfully", "version", version)
	return nil
}

func confirmForce(cmd *cobra.Command, version string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "⚠️  WARNING: Force marking %s will NOT execute migration logic.\n", version)
	fmt.Fprint(cmd.OutOrStdout(), "Confirm action? (y/N): ")

	var response string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &response)
	if err != nil {
		slog.Error("Error reading confirmation", "error", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
