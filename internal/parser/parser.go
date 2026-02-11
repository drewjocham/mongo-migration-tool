package parser

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatBSON Format = "bson"
)

type Cleaner func(map[string]any) map[string]any

type Option func(*config)

type config struct {
	format    Format
	cleaner   Cleaner
	validate  bool
	validator *validator.Validate
}

func WithFormat(format Format) Option {
	return func(c *config) {
		c.format = format
	}
}

func WithCleaner(cleaner Cleaner) Option {
	return func(c *config) {
		c.cleaner = cleaner
	}
}

func WithValidation(enabled bool) Option {
	return func(c *config) {
		c.validate = enabled
	}
}

func WithValidator(v *validator.Validate) Option {
	return func(c *config) {
		c.validator = v
	}
}

// ToConcreteType unmarshals raw JSON bytes directly into a concrete type T
func ToConcreteType[T any](rawPayload []byte) (*T, error) {
	out := new(T)
	if err := jsonutil.Unmarshal(rawPayload, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Parse[T any](raw []byte, opts ...Option) (*T, error) {
	var out T
	if err := ParseInto(raw, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func ParseInto(raw []byte, out any, opts ...Option) error {
	cfg := defaultConfig(opts...)
	m, err := parseMap(raw, cfg)
	if err != nil {
		return err
	}
	buf, err := jsonutil.Marshal(m)
	if err != nil {
		return err
	}
	if err := jsonutil.Unmarshal(buf, out); err != nil {
		return err
	}
	if cfg.validate {
		return ValidateStruct(out, cfg.validator)
	}
	return nil
}

func ParseMap(raw []byte, opts ...Option) (map[string]any, error) {
	cfg := defaultConfig(opts...)
	return parseMap(raw, cfg)
}

func ParseByType(raw []byte, fieldPath string, reg Registry, opts ...Option) (any, error) {
	if fieldPath == "" {
		return nil, fmt.Errorf("type field path is required")
	}
	cfg := defaultConfig(opts...)

	m, err := parseMap(raw, cfg)
	if err != nil {
		return nil, err
	}

	val, ok := valueAtPath(m, fieldPath)
	if !ok {
		return nil, fmt.Errorf("type field %q not found", fieldPath)
	}
	kind, ok := val.(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("type field %q is not a string", fieldPath)
	}

	ctor := reg[strings.ToLower(kind)]
	if ctor == nil {
		return nil, fmt.Errorf("no registry entry for type %q", kind)
	}

	instance := ctor()
	buf, err := jsonutil.Marshal(m)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.Unmarshal(buf, instance); err != nil {
		return nil, err
	}
	if cfg.validate {
		if err := ValidateStruct(instance, cfg.validator); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

func DecodePayload(raw string, format Format) ([]byte, error) {
	switch format {
	case FormatBSON:
		return base64.StdEncoding.DecodeString(raw)
	case FormatJSON, "":
		return []byte(raw), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func ValidateStruct(v any, val *validator.Validate) error {
	if val == nil {
		val = validator.New()
	}
	return val.Struct(v)
}

func defaultConfig(opts ...Option) *config {
	cfg := &config{
		format:  FormatJSON,
		cleaner: DefaultCleaner,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func parseMap(raw []byte, cfg *config) (map[string]any, error) {
	var m map[string]any
	switch cfg.format {
	case FormatBSON:
		var bm bson.M
		if err := bson.Unmarshal(raw, &bm); err != nil {
			return nil, err
		}
		m = map[string]any(bm)
	case FormatJSON, "":
		if err := jsonutil.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", cfg.format)
	}

	if cfg.cleaner != nil {
		m = cfg.cleaner(m)
	}
	return m, nil
}

func valueAtPath(m map[string]any, path string) (any, bool) {
	current := any(m)
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}
