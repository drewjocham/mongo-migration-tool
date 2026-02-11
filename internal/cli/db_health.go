package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type HealthReport struct {
	Database    string            `json:"database"`
	Role        string            `json:"role"`
	OplogWindow string            `json:"oplog_window"`
	OplogSize   string            `json:"oplog_size"`
	Connections string            `json:"connections"`
	Lag         map[string]string `json:"lag,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
}

func NewDBCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "db", Short: "Database utilities"}
	cmd.AddCommand(newDBHealthCmd())
	return cmd
}

func newDBHealthCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Show database health and metrics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := getServices(cmd.Context())
			if err != nil {
				return err
			}

			report, err := buildReport(cmd.Context(), s.MongoClient, s.Config.Database)
			if err != nil {
				return err
			}

			if strings.ToLower(output) == "json" {
				data, err := bson.MarshalExtJSONIndent(report, true, false, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal json: %w", err)
				}
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}

			RenderHealthTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")
	return cmd
}

func buildReport(ctx context.Context, client *mongo.Client, dbName string) (HealthReport, error) {
	var raw bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&raw); err != nil {
		return HealthReport{}, err
	}

	data, _ := bson.MarshalExtJSON(raw, true, true)
	json := gjson.ParseBytes(data)

	report := HealthReport{
		Database: dbName,
		Role:     json.Get("repl.me").String(),
		Lag:      make(map[string]string),
	}

	curr := json.Get("connections.current").Int()
	avail := json.Get("connections.available").Int()
	report.Connections = fmt.Sprintf("%d / %d", curr, avail)

	windowSecs := json.Get("oplog.windowSeconds").Float()
	report.OplogWindow = (time.Duration(windowSecs) * time.Second).String()

	sizeMB := json.Get("oplog.logSizeMB").Uint()
	report.OplogSize = humanize.Bytes(sizeMB * 1024 * 1024)

	members := json.Get("repl.members").Array()
	if len(members) > 0 {
		primaryTS := json.Get("repl.members.#(stateStr==\"PRIMARY\").optime.ts.t").Uint()
		for _, m := range members {
			name := m.Get("name").String()
			state := m.Get("stateStr").String()
			if m.Get("self").Bool() {
				report.Role = state
			}

			ts := m.Get("optime.ts.t").Uint()
			if lag := primaryTS - ts; lag > 0 {
				report.Lag[name] = fmt.Sprintf("%ds", lag)
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s is %ds behind", name, lag))
			}
		}
	}

	if windowSecs < 21600 {
		report.Warnings = append(report.Warnings, "Oplog window is under 6 hours")
	}

	return report, nil
}


func RenderHealthTable(w io.Writer, r HealthReport) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.StripEscape)

	fmt.Fprintf(w, "\n\033[1m--- MONGO HEALTH: %s ---\033[0m\n", strings.ToUpper(r.Database))
	fmt.Fprintln(tw, "\033[1mMETRIC\tVALUE\033[0m")

	roleColor := "\033[34m"
	if strings.EqualFold(r.Role, "PRIMARY") {
		roleColor = "\033[32m"
	}

	fmt.Fprintf(tw, "Role\t%s%s\033[0m\n", roleColor, r.Role)
	fmt.Fprintf(tw, "Connections\t%s\n", r.Connections)
	fmt.Fprintf(tw, "Oplog Window\t%s\n", r.OplogWindow)
	fmt.Fprintf(tw, "Oplog Size\t%s\n", r.OplogSize)

	for node, lag := range r.Lag {
		fmt.Fprintf(tw, "Lag (%s)\t%s\n", node, lag)
	}
	tw.Flush()

	if len(r.Warnings) > 0 {
		fmt.Fprintln(w, "\n\033[33m\033[1m⚠️  WARNINGS\033[0m")
		for _, warn := range r.Warnings {
			fmt.Fprintf(w, "  \033[33m!\033[0m %s\n", warn)
		}
	}
}

func RenderMigrationTable(w io.Writer, status []migration.MigrationStatus) {
	if len(status) == 0 {
		fmt.Fprintln(w, "No migrations found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.StripEscape)

	const (
		iconPending = " [ ] PENDING"
		iconApplied = " \033[32m[✓] APPLIED\033[0m"
	)

	fmt.Fprintln(tw, "\033[1mSTATE\tVERSION\tAPPLIED AT\tDESCRIPTION\033[0m")

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
