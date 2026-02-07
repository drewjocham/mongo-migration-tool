package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newForceCmd() *cobra.Command {
	var assumeYes bool

	cmd := &cobra.Command{
		Use:   "force [version]",
		Short: "Force mark a migration as applied without running it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]

			if !assumeYes && !confirmForce(cmd, version) {
				zap.S().Info("Operation cancelled")
				return nil
			}

			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			if err := engine.Force(cmd.Context(), version); err != nil {
				return fmt.Errorf("%s: %w", ErrFailedToForce, err)
			}

			zap.S().Infow("Migration force marked successfully", "version", version)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Confirm without prompting")
	return cmd
}

func confirmForce(cmd *cobra.Command, version string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "WARNING: Force marking %s will NOT execute migration logic.\n", version)
	fmt.Fprint(cmd.OutOrStdout(), "Confirm action? (y/N): ")

	var response string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &response)
	if err != nil {
		zap.S().Errorw("Error reading confirmation", "error", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
