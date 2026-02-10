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

func newOpslogCmd() *cobra.Command {
	var (
		output string
		search string
		limit  int
	)

	cmd := &cobra.Command{
		Use:     "opslog",
		Short:   "Show applied migration operations",
		Aliases: []string{"ops-log", "history"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			engine, err := getEngine(cmd.Context())
			if err != nil {
				return err
			}

			records, err := engine.ListApplied(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to read opslog: %w", err)
			}

			records = filterOpslog(records, search)
			if limit > 0 && len(records) > limit {
				records = records[:limit]
			}

			out := cmd.OutOrStdout()
			switch strings.ToLower(output) {
			case "json":
				return renderOpslogJSON(out, records)
			case "table", "":
				renderOpslogTable(out, records)
				return nil
			default:
				return fmt.Errorf("unsupported output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by version or description substring")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of results")
	return cmd
}

func filterOpslog(records []migration.MigrationRecord, search string) []migration.MigrationRecord {
	if search == "" {
		return records
	}
	needle := strings.ToLower(search)
	filtered := make([]migration.MigrationRecord, 0, len(records))
	for _, rec := range records {
		if strings.Contains(strings.ToLower(rec.Version), needle) ||
			strings.Contains(strings.ToLower(rec.Description), needle) {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

func renderOpslogJSON(w io.Writer, records []migration.MigrationRecord) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func renderOpslogTable(w io.Writer, records []migration.MigrationRecord) {
	if len(records) == 0 {
		fmt.Fprintln(w, "No applied migrations found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "APPLIED AT\tVERSION\tDESCRIPTION\tCHECKSUM")
	fmt.Fprintln(tw, "----------\t-------\t-----------\t--------")
	for _, rec := range records {
		appliedAt := rec.AppliedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", appliedAt, rec.Version, rec.Description, rec.Checksum)
	}
	tw.Flush()
}
