package cli

import (
	"bufio"
	"fmt"
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
	Long:  `Roll back applied migrations in reverse chronological order.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		engine, err := getEngine(cmd.Context())
		if err != nil {
			return err
		}

		if !downConfirm {
			if confirmed := askForConfirmation(cmd); !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "⏪ Rolling back migrations...\n")

		if err := engine.Down(cmd.Context(), downTargetVersion); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "✅ Rollback completed successfully.")
		return nil
	},
}

func askForConfirmation(cmd *cobra.Command) bool {
	prompt := "WARNING: This will roll back migrations. Continue? [y/N]: "
	if downTargetVersion != "" {
		prompt = fmt.Sprintf("WARNING: Rolling back to version %s. Continue? [y/N]: ", downTargetVersion)
	}

	fmt.Fprint(cmd.OutOrStdout(), prompt)

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
