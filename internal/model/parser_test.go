package model

import (
	"testing"
)

type samplePayload struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func TestParse(t *testing.T) {
	raw := []byte(`{"type":"alpha","name":"ok"}`)
	out, err := Parse[samplePayload](raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if out.Name != "ok" || out.Type != "alpha" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestParseByType(t *testing.T) {
	raw := []byte(`{"meta":{"kind":"alpha"},"name":"ok"}`)
	reg := NewRegistry()
	reg.Register("alpha", func() any { return &samplePayload{} })

	res, err := ParseByType(raw, "meta.kind", reg)
	if err != nil {
		t.Fatalf("parse by type failed: %v", err)
	}
	out, ok := res.(*samplePayload)
	if !ok {
		t.Fatalf("unexpected type: %T", res)
	}
	if out.Name != "ok" {
		t.Fatalf("unexpected output: %+v", out)
	}
}
