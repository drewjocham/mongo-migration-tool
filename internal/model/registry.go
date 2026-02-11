package model

import "strings"

type Registry map[string]func() any

func NewRegistry() Registry {
	return Registry{}
}

func (r Registry) Register(name string, ctor func() any) {
	if name == "" || ctor == nil {
		return
	}
	r[strings.ToLower(name)] = ctor
}
