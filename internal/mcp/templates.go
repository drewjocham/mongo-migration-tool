package mcp

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed migration.go.tmpl
var migrationTemplateRaw string

var migrationTemplate = template.Must(template.New("migration").Parse(migrationTemplateRaw))

type migrationData struct {
	StructName  string
	Version     string
	Description string
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
