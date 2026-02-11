package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			status, err := engine.GetStatus(cmd.Context())
			if err != nil {
				return fmt.Errorf("%s: %w", ErrFailedToGetStatus, err)
			}

			out := cmd.OutOrStdout()

			switch strings.ToLower(format) {
			case "json":
				return renderJSON(out, status)
			case "table":
				renderTable(out, status)
				return nil
			default:
				return fmt.Errorf("unsupported output format: %s", format)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "output", "o", "table", "Output format (table, json)")
	return cmd
}

func renderJSON(w io.Writer, status []migration.MigrationStatus) error {
	encoder := jsonutil.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

func renderTable(w io.Writer, status []migration.MigrationStatus) {
	if len(status) == 0 {
		fmt.Fprintln(w, "No migrations found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	const (
		iconPending = "  [ ]"
		iconApplied = "  \033[32m[✓]\033[0m"
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
