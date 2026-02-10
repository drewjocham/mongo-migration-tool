package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:         "version",
	Short:       "Print the version number",
	Long:        `Print the version, commit hash, and build date of mongo-tool.`,
	Annotations: map[string]string{annotationOffline: "true"},
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), cmd.Root().Version)
	},
}
