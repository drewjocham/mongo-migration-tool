package migration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type TestMigration struct {
	version      string
	description  string
	upExecuted   bool
	downExecuted bool
}

func (m *TestMigration) Version() string {
	return m.version
}

func (m *TestMigration) Description() string {
	return m.description
}

func (m *TestMigration) Up(_ context.Context, _ *mongo.Database) error {
	m.upExecuted = true
	return nil
}

func (m *TestMigration) Down(_ context.Context, _ *mongo.Database) error {
	m.downExecuted = true
	return nil
}

func TestNewEngine(t *testing.T) {
	migrations := make(map[string]Migration)
	db := &mongo.Database{}

	engine := NewEngine(db, "test_migrations", migrations)

	if engine.db != db {
		t.Error("Engine database not set correctly")
	}

	if engine.coll != "test_migrations" {
		t.Error("Engine migrations collection not set correctly")
	}

	if engine.migrations == nil {
		t.Error("Engine migrations map not initialized")
	}
}

func TestDirection(t *testing.T) {
	tests := []struct {
		direction Direction
		expected  string
	}{
		{DirectionUp, "up"},
		{DirectionDown, "down"},
		{Direction(999), "unknown"},
	}

	for _, test := range tests {
		if test.direction.String() != test.expected {
			t.Errorf("Direction %d should return %s, got %s",
				test.direction, test.expected, test.direction.String())
		}
	}
}

func TestMigrationStatus(t *testing.T) {
	status := MigrationStatus{
		Version:     "20240101_001",
		Description: "Test migration",
		Applied:     true,
		AppliedAt:   &time.Time{},
	}

	if status.Version != "20240101_001" {
		t.Error("Version not set correctly")
	}

	if status.Description != "Test migration" {
		t.Error("Description not set correctly")
	}

	if !status.Applied {
		t.Error("Applied status not set correctly")
	}

	if status.AppliedAt == nil {
		t.Error("AppliedAt should not be nil")
	}
}

func TestErrNotSupported(t *testing.T) {
	err := ErrNotSupported{Operation: "test operation"}
	expected := "operation not supported: test operation"

	if err.Error() != expected {
		t.Errorf("Expected error message %s, got %s", expected, err.Error())
	}
}
