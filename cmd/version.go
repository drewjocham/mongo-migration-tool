package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersion(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version, commit hash, and build date of mongo-essential.`,
	Run: func(cmd *cobra.Command, _ []string) {
		slog.Debug("Version command executed",
			"version", appVersion, "commit", appCommit)

		// Human-readable output
		fmt.Printf("mongo-essential version: %s\n", appVersion)
		fmt.Printf("  Commit ID:  %s\n", appCommit)
		fmt.Printf("  Build Date: %s\n", appDate)
	},
}
