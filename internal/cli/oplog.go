package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Centralized Operation Metadata for DRY mapping
var operations = struct {
	codes map[string]string // "i" -> "insert"
	names map[string]string // "insert" -> "i"
}{
	codes: map[string]string{"i": "insert", "u": "update", "d": "delete", "c": "command", "n": "noop"},
	names: map[string]string{"insert": "i", "update": "u", "delete": "d", "command": "c", "noop": "n"},
}

type oplogConfig struct {
	output    string
	namespace string
	regex     string
	ops       string
	objectID  string
	from      string
	to        string
	limit     int64
	follow    bool
	fullDoc   bool
}

type oplogEntry struct {
	TS   primitive.Timestamp `bson:"ts"`
	Op   string              `bson:"op"`
	NS   string              `bson:"ns"`
	Wall *time.Time          `bson:"wall,omitempty"`
	O    bson.M              `bson:"o"`
	O2   bson.M              `bson:"o2,omitempty"`
}

type oplogOutput struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	Namespace string    `json:"namespace"`
	ObjectID  string    `json:"object_id,omitempty"`
	Data      bson.M    `json:"data,omitempty"`
}

// Transform raw BSON entry to formatted output
func (e *oplogEntry) ToOutput() oplogOutput {
	ts := time.Unix(int64(e.TS.T), 0)
	if e.Wall != nil {
		ts = *e.Wall
	}

	id := "N/A"
	for _, m := range []bson.M{e.O, e.O2} {
		if v, ok := m["_id"]; ok {
			id = fmt.Sprintf("%v", v)
			break
		}
	}

	opName, ok := operations.codes[e.Op]
	if !ok {
		opName = e.Op
	}

	return oplogOutput{
		Timestamp: ts,
		Operation: opName,
		Namespace: e.NS,
		ObjectID:  id,
		Data:      e.O,
	}
}

func NewOplogCmd() *cobra.Command {
	cfg := oplogConfig{}
	cmd := &cobra.Command{
		Use:   "oplog",
		Short: "Query MongoDB oplog entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := getServices(cmd.Context())
			if err != nil || s.MongoClient == nil {
				return fmt.Errorf("mongo client unavailable")
			}
			return runOplog(cmd.Context(), cmd.OutOrStdout(), s.MongoClient, cfg)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cfg.output, "output", "o", "table", "Output format (table, json)")
	f.StringVar(&cfg.namespace, "namespace", "", "Filter by exact namespace (db.collection)")
	f.StringVar(&cfg.regex, "regex", "", "Filter by namespace regex")
	f.StringVar(&cfg.ops, "ops", "", "Filter by op codes/names (i,u,d or insert,update)")
	f.StringVar(&cfg.objectID, "object-id", "", "Filter by _id")
	f.StringVar(&cfg.from, "from", "", "Start time (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&cfg.to, "to", "", "End time (RFC3339 or YYYY-MM-DD)")
	f.Int64Var(&cfg.limit, "limit", 50, "Limit results")
	f.BoolVar(&cfg.follow, "follow", false, "Tail entries in real-time")
	f.BoolVar(&cfg.fullDoc, "full-document", false, "Include full document on updates")
	return cmd
}

func runOplog(ctx context.Context, w io.Writer, client *mongo.Client, cfg oplogConfig) error {
	if cfg.namespace != "" && cfg.regex != "" {
		return fmt.Errorf("use --namespace or --regex, not both")
	}

	render := func(entries []oplogEntry) error {
		if strings.ToLower(cfg.output) == "json" {
			out := make([]oplogOutput, len(entries))
			for i, e := range entries {
				out[i] = e.ToOutput()
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
		if len(entries) > 0 {
			fmt.Fprintln(tw, "TIME\tOPERATION\tNS\tOBJECT ID")
		}
		for _, e := range entries {
			o := e.ToOutput()
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				o.Timestamp.Format("2006-01-02 15:04:05"),
				o.Operation,
				o.Namespace,
				o.ObjectID,
			)
		}
		return tw.Flush()
	}

	if cfg.follow {
		return streamOplog(ctx, client, cfg, render)
	}

	filter, err := buildFilter(cfg)
	if err != nil {
		return err
	}

	entries, err := fetchOplog(ctx, client, filter, cfg.limit)
	if err != nil {
		return err
	}
	return render(entries)
}

func buildFilter(cfg oplogConfig) (bson.D, error) {
	filter := bson.D{}
	add := func(k string, v interface{}) { filter = append(filter, bson.E{Key: k, Value: v}) }

	if cfg.namespace != "" {
		add("ns", cfg.namespace)
	}
	if cfg.regex != "" {
		add("ns", primitive.Regex{Pattern: cfg.regex})
	}

	if cfg.ops != "" {
		var codes []string
		for _, op := range strings.Split(cfg.ops, ",") {
			op = strings.TrimSpace(strings.ToLower(op))
			if code, ok := operations.names[op]; ok {
				codes = append(codes, code)
			} else {
				codes = append(codes, op)
			}
		}
		add("op", bson.M{"$in": codes})
	}

	if cfg.objectID != "" {
		var id interface{} = cfg.objectID
		if oid, err := primitive.ObjectIDFromHex(cfg.objectID); err == nil {
			id = oid
		}
		add("$or", bson.A{bson.M{"o._id": id}, bson.M{"o2._id": id}})
	}

	// Time range processing
	tsFilter := bson.M{}
	for _, spec := range []struct {
		val string
		op  string
	}{{cfg.from, "$gte"}, {cfg.to, "$lte"}} {
		if spec.val != "" {
			t, err := parseTime(spec.val)
			if err != nil {
				return nil, err
			}
			tsFilter[spec.op] = t
		}
	}
	if len(tsFilter) > 0 {
		add("ts", tsFilter)
	}

	return filter, nil
}

func fetchOplog(ctx context.Context, client *mongo.Client, filter bson.D, limit int64) ([]oplogEntry, error) {
	db := client.Database("local")
	coll := db.Collection("oplog.rs")

	findOpts := options.Find().SetSort(bson.D{{Key: "ts", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(limit)
	}

	cur, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query oplog: %w", err)
	}
	defer cur.Close(ctx)

	var entries []oplogEntry
	return entries, cur.All(ctx, &entries)
}

func streamOplog(ctx context.Context, client *mongo.Client, cfg oplogConfig, render func([]oplogEntry) error) error {
	pipeline := mongo.Pipeline{}

	match := bson.M{}
	if cfg.regex != "" {
		match["$or"] = bson.A{
			bson.M{"ns.db": bson.M{"$regex": cfg.regex}},
			bson.M{"ns.coll": bson.M{"$regex": cfg.regex}},
		}
	}
	if cfg.objectID != "" {
		var id interface{} = cfg.objectID
		if oid, err := primitive.ObjectIDFromHex(cfg.objectID); err == nil {
			id = oid
		}
		match["documentKey._id"] = id
	}
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})
	}

	opts := options.ChangeStream()
	if cfg.fullDoc {
		opts.SetFullDocument(options.UpdateLookup)
	}

	// watch the whole cluster or specific DB based on namespace
	var stream *mongo.ChangeStream
	var err error
	if cfg.namespace != "" {
		parts := strings.SplitN(cfg.namespace, ".", 2)
		stream, err = client.Database(parts[0]).Collection(parts[1]).Watch(ctx, pipeline, opts)
	} else {
		stream, err = client.Watch(ctx, pipeline, opts)
	}

	if err != nil {
		return fmt.Errorf("stream failed: %w", err)
	}
	defer stream.Close(ctx)

	for stream.Next(ctx) {
		var event bson.M
		if err := stream.Decode(&event); err != nil {
			return err
		}

		// convert ChangeEvent back to pseudo-oplogEntry
		entry := oplogEntry{
			Op: opFromType(event["operationType"].(string)),
			NS: fmt.Sprintf("%v.%v", event["ns"].(bson.M)["db"], event["ns"].(bson.M)["coll"]),
		}
		if doc, ok := event["fullDocument"].(bson.M); ok {
			entry.O = doc
		}
		if key, ok := event["documentKey"].(bson.M); ok {
			entry.O2 = key
		}
		if wall, ok := event["wallTime"].(primitive.DateTime); ok {
			t := wall.Time()
			entry.Wall = &t
			entry.TS = primitive.Timestamp{T: uint32(t.Unix())}
		}

		if err := render([]oplogEntry{entry}); err != nil {
			return err
		}
	}
	return stream.Err()
}

func opFromType(st string) string {
	if code, ok := operations.names[st]; ok {
		return code
	}
	return st
}

func parseTime(v string) (primitive.Timestamp, error) {
	for _, f := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(f, v); err == nil {
			return primitive.Timestamp{T: uint32(t.Unix())}, nil
		}
	}
	return primitive.Timestamp{}, fmt.Errorf("invalid time: %s", v)
}
