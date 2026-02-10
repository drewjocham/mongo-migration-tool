package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type healthReport struct {
	Database        string             `json:"database"`
	Role            string             `json:"role"`
	IsReplicaSet    bool               `json:"is_replica_set"`
	OplogWindow     string             `json:"oplog_window,omitempty"`
	OplogFirst      *time.Time         `json:"oplog_first,omitempty"`
	OplogLast       *time.Time         `json:"oplog_last,omitempty"`
	OplogSizeBytes  int64              `json:"oplog_size_bytes,omitempty"`
	Connections     map[string]float64 `json:"connections,omitempty"`
	OpCounters      map[string]float64 `json:"op_counters,omitempty"`
	MemberLagSecs   map[string]float64 `json:"member_lag_seconds,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
	CollectionStats map[string]any     `json:"collection_stats,omitempty"`
}

type replMember struct {
	Name     string `bson:"name"`
	StateStr string `bson:"stateStr"`
	Self     bool   `bson:"self"`
	Health   int    `bson:"health"`
	Optime   struct {
		TS bson.Timestamp `bson:"ts"`
	} `bson:"optime"`
}

type replStatus struct {
	Set     string       `bson:"set"`
	Members []replMember `bson:"members"`
}

func NewDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database utilities",
	}
	cmd.AddCommand(newDBHealthCmd())
	return cmd
}

func newDBHealthCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Show database health and oplog window",
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := getServices(cmd.Context())
			if err != nil || s.MongoClient == nil {
				return fmt.Errorf("mongo services unavailable")
			}

			report, err := buildHealthReport(cmd.Context(), s.MongoClient, s.Config.Database)
			if err != nil {
				return err
			}

			if strings.ToLower(output) == "json" {
				enc := jsonutil.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}

			renderHealthTable(cmd.OutOrStdout(), report)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")
	return cmd
}

func buildHealthReport(ctx context.Context, client *mongo.Client, dbName string) (healthReport, error) {
	report := healthReport{
		Database:      dbName,
		Connections:   map[string]float64{},
		OpCounters:    map[string]float64{},
		MemberLagSecs: map[string]float64{},
	}

	admin := client.Database("admin")

	var status replStatus
	if err := admin.RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&status); err == nil {
		report.IsReplicaSet = true
		report.Role = detectSelfRole(status.Members)
		applyReplWarnings(&report, status.Members)
	} else {
		report.Role = "standalone/unknown"
		report.Warnings = append(report.Warnings, "replSet status unavailable (standalone or permission denied)")
	}

	var serverStatus bson.M
	if err := admin.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&serverStatus); err == nil {
		report.Connections = flattenToFloatMap(serverStatus["connections"])
		report.OpCounters = flattenToFloatMap(serverStatus["opcounters"])
	}

	if err := fillOplogStats(ctx, client, &report); err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("oplog: %v", err))
	}

	return report, nil
}

func fillOplogStats(ctx context.Context, client *mongo.Client, report *healthReport) error {
	db := client.Database("local")
	coll := db.Collection("oplog.rs")

	var first, last oplogEntry
	if err := coll.FindOne(ctx, bson.D{}, options.FindOne().SetSort(
		bson.D{{Key: "$natural", Value: 1}})).Decode(&first); err != nil { //nolint:lll
		return fmt.Errorf("failed to read oplog start: %w", err)
	}
	if err := coll.FindOne(ctx, bson.D{}, options.FindOne().SetSort(
		bson.D{{Key: "$natural", Value: -1}})).Decode(&last); err != nil { //nolint:lll
		return fmt.Errorf("failed to read oplog end: %w", err)
	}

	firstTime := time.Unix(int64(first.TS.T), 0)
	lastTime := time.Unix(int64(last.TS.T), 0)
	window := lastTime.Sub(firstTime)

	report.OplogFirst, report.OplogLast = &firstTime, &lastTime
	report.OplogWindow = window.String()

	var stats bson.M
	if err := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: "oplog.rs"}}).Decode(&stats); err == nil {
		if size, ok := stats["size"].(float64); ok {
			report.OplogSizeBytes = int64(size)
		}
		report.CollectionStats = stats
	}

	if window < (6 * time.Hour) {
		report.Warnings = append(report.Warnings, fmt.Sprintf("short oplog window: %s", window))
	}
	return nil
}

// flattens nested BSON metrics into a float map for reporting
func flattenToFloatMap(input any) map[string]float64 {
	out := make(map[string]float64)
	m, ok := input.(bson.M)
	if !ok {
		return out
	}
	for k, v := range m {
		switch n := v.(type) {
		case int32:
			out[k] = float64(n)
		case int64:
			out[k] = float64(n)
		case float64:
			out[k] = n
		}
	}
	return out
}

func detectSelfRole(members []replMember) string {
	for _, m := range members {
		if m.Self {
			return strings.ToLower(m.StateStr)
		}
	}
	return "unknown"
}

func applyReplWarnings(report *healthReport, members []replMember) {
	var primaryTS uint32
	for _, m := range members {
		if strings.EqualFold(m.StateStr, "PRIMARY") {
			primaryTS = m.Optime.TS.T
			break
		}
	}
	for _, m := range members {
		if m.Health != 1 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("member %s is unhealthy", m.Name))
		}
		if primaryTS > 0 {
			lag := int64(primaryTS) - int64(m.Optime.TS.T)
			if lag < 0 {
				lag = 0
			}
			report.MemberLagSecs[m.Name] = float64(lag)
		}
	}
}

func renderHealthTable(w io.Writer, report healthReport) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer tw.Flush()

	fmt.Fprintf(tw, "--- DATABASE HEALTH: %s ---\n", report.Database)
	fmt.Fprintf(tw, "METRIC\tVALUE\n")
	fmt.Fprintf(tw, "Role\t%s\n", report.Role)
	fmt.Fprintf(tw, "Oplog Window\t%s\n", report.OplogWindow)

	if len(report.MemberLagSecs) > 0 {
		for name, lag := range report.MemberLagSecs {
			fmt.Fprintf(tw, "Member Lag (%s)\t%.0fs\n", name, lag)
		}
	}

	if cur, ok := report.Connections["current"]; ok {
		fmt.Fprintf(tw, "Connections (Active/Avail)\t%.0f / %.0f\n", cur, report.Connections["available"])
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(tw, "\n--- WARNINGS ---")
		for _, w := range report.Warnings {
			fmt.Fprintf(tw, "!\t%s\n", w)
		}
	}
}
