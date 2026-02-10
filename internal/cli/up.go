package cli

import (
	"fmt"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newUpCmd() *cobra.Command {
	var (
		target string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Run pending migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			plan, err := engine.Plan(cmd.Context(), migration.DirectionUp, target)
			if err != nil {
				return err
			}
			if dryRun {
				renderPlan(cmd.OutOrStdout(), "up", plan)
				return nil
			}
			if len(plan) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Database is already up to date.")
				return nil
			}

			logIntent(target)

			if err := engine.Up(cmd.Context(), target); err != nil {
				return fmt.Errorf("%s: %w", ErrFailedToRun, err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✨ Database is up to date!")
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target version to migrate up to")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print planned migrations without executing")
	return cmd
}

func logIntent(target string) {
	if target != "" {
		zap.S().Infow("Running migrations up to target", "target", target)
		return
	}
	zap.S().Info("Running all pending migrations")
}
