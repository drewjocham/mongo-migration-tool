package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/drewjocham/mongo-migration-tool/migration"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long: `Display the current status of all migrations, showing which have been applied
and which are pending.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		slog.Debug("Fetching migration status from database")

		status, err := engine.GetStatus(ctx)
		if err != nil {
			slog.Error("Failed to retrieve migration status", "error", err)
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		if len(status) == 0 {
			fmt.Println("No migrations found in the registry.")
			return nil
		}

		printStatusTable(status)

		return nil
	},
}

// printStatusTable formats the output into a clean, aligned table
func printStatusTable(status []migration.MigrationStatus) {
	fmt.Println("\nMigration Status Report")
	fmt.Println(strings.Repeat("=", 30))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "STATE\tVERSION\tAPPLIED AT\tDESCRIPTION")
	fmt.Fprintln(w, "-----\t-------\t----------\t-----------")

	for _, s := range status {
		state := "  [ ]" // Pending
		appliedAt := "n/a"

		if s.Applied {
			state = "  [✓]" // Applied
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			state,
			s.Version,
			appliedAt,
			s.Description,
		)
	}
	w.Flush()
	fmt.Println()
}
