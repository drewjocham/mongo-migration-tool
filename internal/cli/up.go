package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newUpCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Run pending migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			logIntent(target)

			if err := engine.Up(cmd.Context(), target); err != nil {
				return fmt.Errorf("migration up failed: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✨ Database is up to date!")
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target version to migrate up to")
	return cmd
}

func logIntent(target string) {
	if target != "" {
		zap.S().Infow("Running migrations up to target", "target", target)
		return
	}
	zap.S().Info("Running all pending migrations")
}
