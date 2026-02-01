package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newDownCmd() *cobra.Command {
	var (
		target  string
		confirm bool
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back migrations",
		Long:  "Roll back applied migrations in reverse order. Use --target to stop before a specific version.",
		Example: `  mongo-essential down --target 20240101_001
  mongo-essential down --yes  # Rollback ALL migrations without prompting`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			if !confirm && !askForConfirmation(cmd, target) {
				fmt.Fprintln(cmd.OutOrStdout(), "Operation cancelled.")
				return nil
			}

			zap.S().Infow("Starting migration rollback", "target", target)
			if err := engine.Down(cmd.Context(), target); err != nil {
				return fmt.Errorf("rollback failed: %w", err)
			}

			zap.S().Info("Rollback completed successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Version to roll back to (exclusive)")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func askForConfirmation(cmd *cobra.Command, target string) bool {
	msg := "⚠️  WARNING: You are about to roll back ALL migrations."
	if target != "" {
		msg = fmt.Sprintf("⚠️  WARNING: Rolling back migrations to version %s.", target)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s Continue? [y/N]: ", msg)

	var response string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &response)
	if err != nil {
		zap.S().Errorw("Failed to read confirmation", "error", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
