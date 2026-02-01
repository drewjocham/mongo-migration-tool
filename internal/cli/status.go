package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/drewjocham/mongo-migration-tool/migration"
	"github.com/spf13/cobra"
)

var outputFormat string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		engine, err := getEngine(cmd.Context())
		if err != nil {
			return err
		}

		status, err := engine.GetStatus(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}

		out := cmd.OutOrStdout()

		switch strings.ToLower(outputFormat) {
		case "json":
			return renderJSON(out, status)
		default:
			renderTable(out, status)
			return nil
		}
	},
}

func renderJSON(w io.Writer, status []migration.MigrationStatus) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

func renderTable(w io.Writer, status []migration.MigrationStatus) {
	if len(status) == 0 {
		fmt.Fprintln(w, "∅ No migrations found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	const (
		iconPending = "  [ ]"
		iconApplied = "  \033[32m[✓]\033[0m" // ANSI Green checkmark
	)

	fmt.Fprintln(tw, "STATE\tVERSION\tAPPLIED AT\tDESCRIPTION")
	fmt.Fprintln(tw, "-----\t-------\t----------\t-----------")

	for _, s := range status {
		state := iconPending
		appliedAt := "-"

		if s.Applied {
			state = iconApplied
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04")
			}
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", state, s.Version, appliedAt, s.Description)
	}

	tw.Flush()
}
