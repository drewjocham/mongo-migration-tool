package model

import im "github.com/drewjocham/mongo-migration-tool/internal/model"

type (
	Cleaner  = im.Cleaner
	Option   = im.Option
	Registry = im.Registry
)

var (
	NewRegistry = im.NewRegistry
)

func Parse[T any](raw []byte, opts ...Option) (*T, error) { return im.Parse[T](raw, opts...) }
func ParseInto(raw []byte, out any, opts ...Option) error { return im.ParseInto(raw, out, opts...) }
func ParseByType(raw []byte, fieldPath string, reg Registry, opts ...Option) (any, error) {
	return im.ParseByType(raw, fieldPath, reg, opts...)
}
