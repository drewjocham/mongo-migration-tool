package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	upTargetVersion string

	// upCmd represents the up command
	upCmd = func() *cobra.Command {
		cmd := &cobra.Command{
			Use:   "up",
			Short: "Run all pending migrations (or up to target version)",
			Long: `Run pending migrations in forward direction.
		By default, all pending migrations are executed in version order.`,
			RunE: func(cmd *cobra.Command, _ []string) error {
				ctx := cmd.Context()

				if upTargetVersion != "" {
					slog.Info("Running migrations up to specific version", "target", upTargetVersion)
				} else {
					slog.Info("Running all pending migrations")
				}

				if err := engine.Up(ctx, upTargetVersion); err != nil {
					slog.Error("Migration 'Up' failed",
						"target", upTargetVersion,
						"error", err,
					)
					return fmt.Errorf("migration up failed: %w", err)
				}

				slog.Info("Migrations completed successfully")
				fmt.Println("✓ Database is up to date!")
				return nil
			},
		}

		cmd.Flags().StringVar(&upTargetVersion, "target", "", "Target version to migrate up to")
		return cmd
	}()
)
