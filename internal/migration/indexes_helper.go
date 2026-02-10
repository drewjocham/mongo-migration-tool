package migration

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type IndexKey struct {
	Field string
	Order interface{}
}

func Asc(field string) IndexKey  { return IndexKey{Field: field, Order: 1} }
func Desc(field string) IndexKey { return IndexKey{Field: field, Order: -1} }
func Text(field string) IndexKey { return IndexKey{Field: field, Order: "text"} }

type IndexBuilder struct {
	model mongo.IndexModel
}

type IndexCreateOption func(*indexCreateConfig)

type indexCreateConfig struct {
	nameOverride func(string) string
	namePrefix   string
	nameSuffix   string
}

func WithIndexNamePrefix(prefix string) IndexCreateOption {
	return func(cfg *indexCreateConfig) {
		cfg.namePrefix = prefix
	}
}

func WithIndexNameSuffix(suffix string) IndexCreateOption {
	return func(cfg *indexCreateConfig) {
		cfg.nameSuffix = suffix
	}
}

func WithIndexNameOverride(fn func(string) string) IndexCreateOption {
	return func(cfg *indexCreateConfig) {
		cfg.nameOverride = fn
	}
}

func Index(keys ...IndexKey) *IndexBuilder {
	b := &IndexBuilder{model: mongo.IndexModel{Keys: bson.D{}}}
	for _, key := range keys {
		b.Key(key.Field, key.Order)
	}
	return b
}

func (b *IndexBuilder) Key(field string, order interface{}) *IndexBuilder {
	keys, ok := b.model.Keys.(bson.D)
	if !ok {
		return b
	}
	keys = append(keys, bson.E{Key: field, Value: order})
	b.model.Keys = keys
	return b
}

func (b *IndexBuilder) Name(name string) *IndexBuilder {
	opts := b.ensureOptions()
	opts.SetName(name)
	return b
}

func (b *IndexBuilder) Unique() *IndexBuilder {
	opts := b.ensureOptions()
	opts.SetUnique(true)
	return b
}

func (b *IndexBuilder) Sparse() *IndexBuilder {
	opts := b.ensureOptions()
	opts.SetSparse(true)
	return b
}

func (b *IndexBuilder) TTL(seconds int32) *IndexBuilder {
	opts := b.ensureOptions()
	opts.SetExpireAfterSeconds(seconds)
	return b
}

func (b *IndexBuilder) Partial(expr interface{}) *IndexBuilder {
	opts := b.ensureOptions()
	opts.SetPartialFilterExpression(expr)
	return b
}

func (b *IndexBuilder) Model() mongo.IndexModel {
	return b.model
}

func (b *IndexBuilder) ensureOptions() *options.IndexOptionsBuilder {
	if b.model.Options == nil {
		b.model.Options = options.Index()
	}
	return b.model.Options
}

func CreateIndexes(ctx context.Context, coll *mongo.Collection, indexes ...*IndexBuilder) error {
	return CreateIndexesWithOptions(ctx, coll, nil, indexes...)
}

func CreateIndexesWithOptions(
	ctx context.Context, coll *mongo.Collection, opts []IndexCreateOption, indexes ...*IndexBuilder,
) error {
	if len(indexes) == 0 {
		return nil
	}

	cfg := &indexCreateConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	models := make([]mongo.IndexModel, 0, len(indexes))
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		model := idx.Model()
		if err := ensureIndexName(&model, cfg); err != nil {
			return err
		}
		models = append(models, model)
	}

	if len(models) == 0 {
		return nil
	}

	_, err := coll.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("create indexes failed: %w", err)
	}
	return nil
}

func DropIndexes(ctx context.Context, coll *mongo.Collection, names ...string) error {
	for _, name := range names {
		if name == "" {
			continue
		}
		if err := coll.Indexes().DropOne(ctx, name); err != nil {
			if isIndexNotFound(err) {
				continue
			}
			return fmt.Errorf("drop index %s failed: %w", name, err)
		}
	}
	return nil
}

func ensureIndexName(model *mongo.IndexModel, cfg *indexCreateConfig) error {
	keys, ok := model.Keys.(bson.D)
	if !ok || len(keys) == 0 {
		return fmt.Errorf("index must define at least one key")
	}

	opts := ensureIndexOptionsBuilder(model)
	values, err := buildIndexOptions(opts)
	if err != nil {
		return err
	}

	if values.Name != nil && *values.Name != "" {
		name := *values.Name
		if cfg != nil && cfg.nameOverride != nil {
			name = cfg.nameOverride(name)
		}
		if cfg != nil {
			name = cfg.namePrefix + name + cfg.nameSuffix
		}
		opts.SetName(name)
		return nil
	}

	base := buildIndexBaseName(keys)
	optsSuffix := buildIndexOptionsSuffix(values)
	name := base
	if optsSuffix != "" {
		name = base + "_" + optsSuffix
	}
	if cfg != nil {
		if cfg.nameOverride != nil {
			name = cfg.nameOverride(name)
		}
		name = cfg.namePrefix + name + cfg.nameSuffix
	}
	opts.SetName(name)
	return nil
}

func buildIndexBaseName(keys bson.D) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s_%v", key.Key, key.Value))
	}
	return strings.Join(parts, "_")
}

func buildIndexOptionsSuffix(opts *options.IndexOptions) string {
	if opts == nil {
		return ""
	}

	var parts []string
	if opts.Unique != nil && *opts.Unique {
		parts = append(parts, "unique")
	}
	if opts.Sparse != nil && *opts.Sparse {
		parts = append(parts, "sparse")
	}
	if opts.ExpireAfterSeconds != nil && *opts.ExpireAfterSeconds > 0 {
		parts = append(parts, fmt.Sprintf("ttl_%d", *opts.ExpireAfterSeconds))
	}
	if opts.PartialFilterExpression != nil {
		parts = append(parts, "partial")
	}
	sort.Strings(parts)
	return strings.Join(parts, "_")
}

func ensureIndexOptionsBuilder(model *mongo.IndexModel) *options.IndexOptionsBuilder {
	if model.Options == nil {
		model.Options = options.Index()
	}
	return model.Options
}

func buildIndexOptions(builder *options.IndexOptionsBuilder) (*options.IndexOptions, error) {
	if builder == nil {
		return &options.IndexOptions{}, nil
	}
	opts := &options.IndexOptions{}
	for _, setter := range builder.List() {
		if setter == nil {
			continue
		}
		if err := setter(opts); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

func isIndexNotFound(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Name == "IndexNotFound" {
			return true
		}
	}
	if strings.Contains(err.Error(), "IndexNotFound") {
		return true
	}
	return false
}
