package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	downTargetVersion string

	// downCmd represents the down command
	downCmd = func() *cobra.Command {
		cmd := &cobra.Command{
			Use:   "down",
			Short: "Roll back migrations (down to target version)",
			Long: `Roll back applied migrations in reverse order.
The target version itself will remain applied (the rollback stops once it is reached).`,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()

				slog.Info("Starting rollback", "target", downTargetVersion)

				if err := engine.Down(ctx, downTargetVersion); err != nil {
					slog.Error("Migration rollback failed",
						"target", downTargetVersion,
						"error", err,
					)
					return fmt.Errorf("migration down failed: %w", err)
				}

				slog.Info("Rollback completed successfully", "target", downTargetVersion)
				return nil
			},
		}

		cmd.Flags().StringVar(&downTargetVersion, "target", "", "Target version to rollback to (required)")
		_ = cmd.MarkFlagRequired("target")

		return cmd
	}()
)
