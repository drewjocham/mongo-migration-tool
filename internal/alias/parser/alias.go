package parser

import (
	ip "github.com/drewjocham/mongo-migration-tool/internal/parser"
	"github.com/go-playground/validator/v10"
)

type (
	Format   = ip.Format
	Cleaner  = ip.Cleaner
	Option   = ip.Option
	Registry = ip.Registry
)

const (
	FormatJSON = ip.FormatJSON
	FormatBSON = ip.FormatBSON
)

var (
	DefaultRegistry = ip.DefaultRegistry
	NewRegistry     = ip.NewRegistry
	Register        = ip.Register

	WithFormat     = ip.WithFormat
	WithCleaner    = ip.WithCleaner
	WithValidation = ip.WithValidation
	WithValidator  = ip.WithValidator
)

func Parse[T any](raw []byte, opts ...Option) (*T, error) { return ip.Parse[T](raw, opts...) }
func ParseInto(raw []byte, out any, opts ...Option) error { return ip.ParseInto(raw, out, opts...) }
func ParseMap(raw []byte, opts ...Option) (map[string]any, error) {
	return ip.ParseMap(raw, opts...)
}
func ParseByType(raw []byte, fieldPath string, reg Registry, opts ...Option) (any, error) {
	return ip.ParseByType(raw, fieldPath, reg, opts...)
}
func DecodePayload(raw string, format Format) ([]byte, error) { return ip.DecodePayload(raw, format) }
func ValidateStruct(v any, val *validator.Validate) error     { return ip.ValidateStruct(v, val) }
