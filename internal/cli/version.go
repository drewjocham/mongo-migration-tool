package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version, commit hash, and build date of mongo-essential.`,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), rootCmd.Version)
	},
}
