package migration

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

//go:embed template.tmpl
var migrationTemplate string

type Generator struct {
	OutputPath string
}

func (g *Generator) Create(name string) (string, string, error) {
	timestamp := time.Now().Format("20060102_150405")

	cleanName := strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(name))
	version := fmt.Sprintf("%s_%s", timestamp, cleanName)
	targetPath := filepath.Join(g.OutputPath, version+".go")

	if err := os.MkdirAll(g.OutputPath, 0750); err != nil {
		return "", "", fmt.Errorf("%s: %w", ErrFailedToCreateFile, err)
	}

	data := struct {
		PackageName string
		Version     string
		Description string
		StructName  string
	}{
		PackageName: filepath.Base(g.OutputPath),
		Version:     version,
		Description: name,
		StructName:  "Migration_" + version,
	}

	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", ErrFailedToParseTemplate, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("%s: %w", ErrFailedToExecuteTemplate, err)
	}

	return targetPath, version, os.WriteFile(targetPath, buf.Bytes(), 0600)
}
