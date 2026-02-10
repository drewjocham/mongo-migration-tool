package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
	output     string
	namespace  string
	regex      string
	ops        string
	objectID   string
	from       string
	to         string
	limit      int64
	follow     bool
	fullDoc    bool
	resumeFile string
}

type oplogEntry struct {
	TS   bson.Timestamp `bson:"ts"`
	Op   string         `bson:"op"`
	NS   string         `bson:"ns"`
	Wall *time.Time     `bson:"wall,omitempty"`
	O    bson.M         `bson:"o"`
	O2   bson.M         `bson:"o2,omitempty"`
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
	f.StringVar(&cfg.resumeFile, "resume-file", "", "File to store/read the resume token for persistent tailing")
	return cmd
}

func runOplog(ctx context.Context, w io.Writer, client *mongo.Client, cfg oplogConfig) error {
	if cfg.namespace != "" && cfg.regex != "" {
		return fmt.Errorf("use --namespace or --regex, not both")
	}
	if cfg.follow && cfg.to != "" {
		return fmt.Errorf("--to is not supported with --follow")
	}

	render := func(entries []oplogEntry) error {
		if strings.ToLower(cfg.output) == "json" {
			out := make([]oplogOutput, len(entries))
			for i, e := range entries {
				out[i] = e.ToOutput()
			}
			enc := jsonutil.NewEncoder(w)
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
		add("ns", bson.Regex{Pattern: cfg.regex})
	}

	if cfg.ops != "" {
		codes, err := parseOps(cfg.ops)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 {
			add("op", bson.M{"$in": codes})
		}
	}

	if cfg.objectID != "" {
		var id interface{} = cfg.objectID
		if oid, err := bson.ObjectIDFromHex(cfg.objectID); err == nil {
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
	coll, err := oplogCollection(client)
	if err != nil {
		return nil, err
	}

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
		if oid, err := bson.ObjectIDFromHex(cfg.objectID); err == nil {
			id = oid
		}
		match["documentKey._id"] = id
	}
	if cfg.ops != "" {
		codes, err := parseOps(cfg.ops)
		if err != nil {
			return err
		}
		names := mapOpsToNames(codes)
		if len(names) > 0 {
			match["operationType"] = bson.M{"$in": names}
		}
	}
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})
	}

	opts := options.ChangeStream()
	if cfg.fullDoc {
		opts.SetFullDocument(options.UpdateLookup)
	}
	if cfg.resumeFile != "" {
		if token, err := os.ReadFile(cfg.resumeFile); err == nil && len(token) > 0 {
			opts.SetResumeAfter(bson.Raw(token))
		}
	}

	// watch the whole cluster or specific DB based on namespace
	var stream *mongo.ChangeStream
	var err error
	if cfg.namespace != "" {
		parts := strings.SplitN(cfg.namespace, ".", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid namespace: %s (expected db.collection)", cfg.namespace)
		}
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

		entry := oplogEntry{}
		if opType, ok := event["operationType"].(string); ok {
			entry.Op = opFromType(opType)
		}
		if ns := formattedNamespace(event["ns"]); ns != "" {
			entry.NS = ns
		}
		if doc, ok := toBsonM(event["fullDocument"]); ok {
			entry.O = doc
		}
		if key, ok := toBsonM(event["documentKey"]); ok {
			entry.O2 = key
		}
		if clusterTime, ok := event["clusterTime"].(bson.Timestamp); ok {
			entry.TS = clusterTime
		}
		if wall, ok := event["wallTime"].(bson.DateTime); ok {
			t := wall.Time()
			entry.Wall = &t
			if entry.TS.T == 0 && entry.TS.I == 0 {
				entry.TS = bson.Timestamp{T: uint32(t.Unix())}
			}
		}

		if cfg.resumeFile != "" {
			if token := stream.ResumeToken(); len(token) > 0 {
				_ = os.WriteFile(cfg.resumeFile, token, 0o644)
			}
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

func formattedNamespace(raw interface{}) string {
	if ns, ok := toBsonM(raw); ok {
		db, _ := ns["db"].(string)
		coll, _ := ns["coll"].(string)
		if db != "" && coll != "" {
			return fmt.Sprintf("%s.%s", db, coll)
		}
	}
	return ""
}

func toBsonM(val interface{}) (bson.M, bool) {
	if val == nil {
		return nil, false
	}
	switch v := val.(type) {
	case bson.M:
		return v, true
	case bson.D:
		return bsonDToMap(v), true
	case map[string]interface{}:
		return bson.M(v), true
	default:
		return nil, false
	}
}

func bsonDToMap(d bson.D) bson.M {
	if len(d) == 0 {
		return bson.M{}
	}
	out := make(bson.M, len(d))
	for _, elem := range d {
		out[elem.Key] = elem.Value
	}
	return out
}

func parseOps(raw string) ([]string, error) {
	clean := strings.Split(strings.ReplaceAll(raw, " ", ""), ",")
	out := make([]string, 0, len(clean))
	for _, item := range clean {
		if item == "" {
			continue
		}
		item = strings.ToLower(item)
		if _, ok := operations.codes[item]; ok {
			out = append(out, item)
			continue
		}
		if code, ok := operations.names[item]; ok {
			out = append(out, code)
			continue
		}
		return nil, fmt.Errorf("unsupported op: %s", item)
	}
	return out, nil
}

func mapOpsToNames(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		if name, ok := operations.codes[code]; ok {
			out = append(out, name)
		}
	}
	return out
}

func parseTime(v string) (bson.Timestamp, error) {
	for _, f := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(f, v); err == nil {
			return bson.Timestamp{T: uint32(t.Unix())}, nil
		}
	}
	return bson.Timestamp{}, fmt.Errorf("invalid time: %s", v)
}

func oplogCollection(client *mongo.Client) (*mongo.Collection, error) {
	localDB := client.Database("local")
	names, err := localDB.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list local collections: %w", err)
	}
	for _, name := range names {
		if name == "oplog.rs" {
			return localDB.Collection("oplog.rs"), nil
		}
		if name == "oplog.$main" {
			return localDB.Collection("oplog.$main"), nil
		}
	}
	return nil, fmt.Errorf("oplog collection not found (requires replica set)")
}
