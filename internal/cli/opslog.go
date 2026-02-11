package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
)

func newOpslogCmd() *cobra.Command {
	var (
		output  string
		search  string
		version string
		regex   string
		from    string
		to      string
		limit   int
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

			options, err := buildOpslogFilter(search, version, regex, from, to)
			if err != nil {
				return err
			}
			records = filterOpslog(records, options)
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
	cmd.Flags().StringVar(&version, "version", "", "Filter by exact migration version")
	cmd.Flags().StringVar(&regex, "regex", "", "Filter by regex against version or description")
	cmd.Flags().StringVar(&from, "from", "", "Filter applied at or after time (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "Filter applied at or before time (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of results")
	return cmd
}

type opslogFilter struct {
	search  string
	version string
	regex   *regexp.Regexp
	from    *time.Time
	to      *time.Time
}

func buildOpslogFilter(search, version, regex, from, to string) (opslogFilter, error) {
	filter := opslogFilter{
		search:  strings.TrimSpace(search),
		version: strings.TrimSpace(version),
	}
	if regex != "" {
		compiled, err := regexp.Compile(regex)
		if err != nil {
			return opslogFilter{}, fmt.Errorf("invalid regex: %w", err)
		}
		filter.regex = compiled
	}
	if from != "" {
		ts, err := parseOpslogTime(from)
		if err != nil {
			return opslogFilter{}, err
		}
		filter.from = &ts
	}
	if to != "" {
		ts, err := parseOpslogTime(to)
		if err != nil {
			return opslogFilter{}, err
		}
		filter.to = &ts
	}
	return filter, nil
}

func filterOpslog(records []migration.MigrationRecord, filter opslogFilter) []migration.MigrationRecord {
	filtered := make([]migration.MigrationRecord, 0, len(records))
	for _, rec := range records {
		if filter.version != "" && rec.Version != filter.version {
			continue
		}
		if filter.from != nil && rec.AppliedAt.Before(*filter.from) {
			continue
		}
		if filter.to != nil && rec.AppliedAt.After(*filter.to) {
			continue
		}
		if filter.regex != nil && !filter.regex.MatchString(rec.Version+" "+rec.Description) {
			continue
		}
		if filter.search != "" {
			needle := strings.ToLower(filter.search)
			if !strings.Contains(strings.ToLower(rec.Version), needle) &&
				!strings.Contains(strings.ToLower(rec.Description), needle) {
				continue
			}
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

func parseOpslogTime(value string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02", value); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("invalid time: %s (use RFC3339 or YYYY-MM-DD)", value)
}

func renderOpslogJSON(w io.Writer, records []migration.MigrationRecord) error {
	encoder := jsonutil.NewEncoder(w)
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
