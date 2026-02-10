package parser

import "strings"

type Registry map[string]func() any

var DefaultRegistry = Registry{}

func NewRegistry() Registry {
	return Registry{}
}

func (r Registry) Register(name string, ctor func() any) {
	if name == "" || ctor == nil {
		return
	}
	r[strings.ToLower(name)] = ctor
}

func Register(name string, ctor func() any) {
	DefaultRegistry.Register(name, ctor)
}
