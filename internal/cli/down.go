package cli

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

var (
	downTargetVersion string
	downConfirm       bool
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back migrations",
	Long:  `Roll back applied migrations in reverse chronological order. Use --target to specify a version to stop *before*.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getEngine(cmd.Context())
		if err != nil {
			return err
		}

		if !downConfirm {
			prompt := "⚠️  WARNING: This will roll back all applied migrations. Continue? [y/N]: "
			if downTargetVersion != "" {
				prompt = fmt.Sprintf("⚠️  WARNING: You are about to roll back migrations to just before version %s. Continue? [y/N]: ", downTargetVersion)
			}
			fmt.Fprint(cmd.OutOrStdout(), prompt)

			reader := bufio.NewReader(cmd.InOrStdin())
			response, err := reader.ReadString('\n')
			if err != nil {
				slog.Error("Error reading confirmation", "error", err)
				return nil // Abort gracefully
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Rollback aborted by user.")
				return nil
			}
		}

		slog.Info("Starting migration rollback", "target", downTargetVersion)

		if err := engine.Down(cmd.Context(), downTargetVersion); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		slog.Info("Rollback completed successfully")
		return nil
	},
}

// init() function removed, flags moved to root.go
// func init() {
// 	downCmd.Flags().StringVarP(&downTargetVersion, "target", "t", "", "Version to roll back to (exclusive)")
// 	downCmd.Flags().BoolVarP(&downConfirm, "yes", "y", false, "Confirm the action without prompting")
// }
