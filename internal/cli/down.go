package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var downTarget string

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback migrations",
	RunE:  runDown,
}

func init() {
	downCmd.Flags().StringVar(&downTarget, "target", "", "Target version to roll back to")
}

func runDown(cmd *cobra.Command, _ []string) error {
	engine, err := getEngine(cmd.Context())
	if err != nil {
		return err
	}

	if downTarget != "" {
		slog.Info("Rolling back migrations to version", "target", downTarget)
	} else {
		slog.Info("Rolling back the last migration")
	}

	if err := engine.Down(cmd.Context(), downTarget); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "✅ Rollback completed successfully")
	return nil
}
