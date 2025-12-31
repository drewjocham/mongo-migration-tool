package migration

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Migration represents a single database migration.
type Migration interface {
	// Version returns the unique version identifier for this migration.
	// Recommended format: YYYYMMDD_NNN (e.g., "20240101_001")
	Version() string

	// Description returns a human-readable description of what this migration does.
	Description() string

	// Up executes the migration, applying changes to the database.
	Up(ctx context.Context, db *mongo.Database) error

	// Down rolls back the migration, undoing changes made by Up.
	Down(ctx context.Context, db *mongo.Database) error
}

// MigrationRecord represents a migration record stored in the database.
type MigrationRecord struct { //nolint:revive // MigrationRecord is clearer than Record in this context
	Version     string    `bson:"version"`
	Description string    `bson:"description"`
	AppliedAt   time.Time `bson:"applied_at"`
	Checksum    string    `bson:"checksum,omitempty"`
}

// Direction represents the migration direction (up or down).
type Direction int

const (
	// DirectionUp indicates applying migrations forward
	DirectionUp Direction = iota
	// DirectionDown indicates rolling back migrations
	DirectionDown
)

// String returns a string representation of the direction.
func (d Direction) String() string {
	switch d {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	default:
		return "unknown"
	}
}

// MigrationStatus represents the status of a migration.
//
// This shows whether a migration has been applied and when.
type MigrationStatus struct { //nolint:revive // MigrationStatus is clearer than Status in this context
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Applied     bool       `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
}

// ErrNotSupported is returned when a migration doesn't support an operation.
type ErrNotSupported struct {
	Operation string
}

func (e ErrNotSupported) Error() string {
	return "operation not supported: " + e.Operation
}
