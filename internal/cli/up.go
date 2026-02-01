package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	RunE:  runUp,
}

func runUp(cmd *cobra.Command, _ []string) error {
	engine, err := getEngine(cmd.Context())
	if err != nil {
		return err
	}

	logIntent(upTarget)

	if err := engine.Up(cmd.Context(), upTarget); err != nil {
		return fmt.Errorf("migration up failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✨ Database is up to date!")
	return nil
}

func logIntent(target string) {
	if target != "" {
		slog.Info("Running migrations up to target", "target", target)
		return
	}
	slog.Info("Running all pending migrations")
}
