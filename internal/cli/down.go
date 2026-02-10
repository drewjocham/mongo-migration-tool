package cli

import (
	"fmt"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newDownCmd() *cobra.Command {
	var (
		target  string
		confirm bool
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back migrations",
		Long:  "Roll back applied migrations in reverse order. Use --target to stop before a specific version.",
		Example: `  mt down --target 20240101_001
  mt down --yes  # Rollback ALL migrations without prompting`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}
			plan, err := engine.Plan(cmd.Context(), migration.DirectionDown, target)
			if err != nil {
				return err
			}

			if dryRun {
				renderPlan(cmd.OutOrStdout(), "down", plan)
				return nil
			}
			if len(plan) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No migrations to roll back.")
				return nil
			}

			msg := "WARNING: You are about to roll back ALL migrations. Continue? [y/N]: "
			if target != "" {
				msg = fmt.Sprintf("WARNING: Rolling back migrations down to version %s. Continue? [y/N]: ", target)
			}

			if !confirm && !promptConfirmation(cmd, msg) {
				fmt.Fprintln(cmd.OutOrStdout(), "Operation cancelled.")
				return nil
			}

			zap.S().Infow("Starting migration rollback", "target", target)
			if err := engine.Down(cmd.Context(), target); err != nil {
				return fmt.Errorf("%s: %w", ErrFailedToDown, err)
			}

			zap.S().Info("Rollback completed successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Version to roll back to (exclusive)")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print planned rollbacks without executing")

	return cmd
}
