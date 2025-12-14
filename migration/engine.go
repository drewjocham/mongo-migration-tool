package migration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Engine is the main migration engine for executing and tracking MongoDB migrations.
type Engine struct {
	db                   *mongo.Database
	migrationsCollection string
	migrations           map[string]Migration
}

// NewEngine creates a new migration engine.
func NewEngine(db *mongo.Database, migrationsCollection string, migrations map[string]Migration) *Engine {
	return &Engine{
		db:                   db,
		migrationsCollection: migrationsCollection,
		migrations:           migrations,
	}
}

// GetStatus returns the status of all migrations.
func (e *Engine) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	appliedMigrations, err := e.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]MigrationRecord)
	for _, record := range appliedMigrations {
		appliedMap[record.Version] = record
	}

	var allVersions []string
	for version := range e.migrations {
		allVersions = append(allVersions, version)
	}

	for version := range appliedMap {
		if _, exists := e.migrations[version]; !exists {
			allVersions = append(allVersions, version)
		}
	}

	sort.Strings(allVersions)

	var status []MigrationStatus
	for _, version := range allVersions {
		migration := e.migrations[version]
		applied, exists := appliedMap[version]

		description := ""
		if migration != nil {
			description = migration.Description()
		} else if exists {
			description = applied.Description
		}

		migrationStatus := MigrationStatus{
			Version:     version,
			Description: description,
			Applied:     exists,
		}

		if exists {
			migrationStatus.AppliedAt = &applied.AppliedAt
		}

		status = append(status, migrationStatus)
	}

	return status, nil
}

// Up runs migrations forward to the specified target version.
func (e *Engine) Up(ctx context.Context, target string) error {
	return e.migrate(ctx, DirectionUp, target)
}

// Down rolls back migrations to the specified target version.
func (e *Engine) Down(ctx context.Context, target string) error {
	return e.migrate(ctx, DirectionDown, target)
}

// Force marks a migration as applied without actually running it.
func (e *Engine) Force(ctx context.Context, version string) error {
	migration, exists := e.migrations[version]
	if !exists {
		return fmt.Errorf("migration %s not found", version)
	}

	record := MigrationRecord{
		Version:     version,
		Description: migration.Description(),
		AppliedAt:   time.Now().UTC(),
		Checksum:    e.calculateChecksum(migration),
	}

	collection := e.db.Collection(e.migrationsCollection)
	_, err := collection.ReplaceOne(
		ctx,
		bson.M{"version": version},
		record,
		options.Replace().SetUpsert(true),
	)

	if err != nil {
		return fmt.Errorf("failed to force migration %s: %w", version, err)
	}

	return nil
}

// migrate executes migrations in the specified direction
func (e *Engine) migrate(ctx context.Context, direction Direction, target string) error {
	appliedMigrations, err := e.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, record := range appliedMigrations {
		appliedMap[record.Version] = true
	}

	migrationsToExecute, err := e.getMigrationsToExecute(direction, target, appliedMap)
	if err != nil {
		return fmt.Errorf("failed to determine migrations to execute: %w", err)
	}

	if len(migrationsToExecute) == 0 {
		return nil
	}

	for _, version := range migrationsToExecute {
		migration := e.migrations[version]
		if migration == nil {
			return fmt.Errorf("migration %s not found", version)
		}

		err := e.executeMigration(ctx, migration, direction)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s %s: %w", version, direction.String(), err)
		}
	}

	return nil
}

// executeMigration executes a single migration
func (e *Engine) executeMigration(ctx context.Context, migration Migration, direction Direction) error {
	version := migration.Version()

	switch direction {
	case DirectionUp:
		if err := migration.Up(ctx, e.db); err != nil {
			return err
		}

		record := MigrationRecord{
			Version:     version,
			Description: migration.Description(),
			AppliedAt:   time.Now().UTC(),
			Checksum:    e.calculateChecksum(migration),
		}

		collection := e.db.Collection(e.migrationsCollection)
		_, err := collection.InsertOne(ctx, record)
		return err

	case DirectionDown:
		if err := migration.Down(ctx, e.db); err != nil {
			return err
		}

		collection := e.db.Collection(e.migrationsCollection)
		_, err := collection.DeleteOne(ctx, bson.M{"version": version})
		return err

	default:
		return fmt.Errorf("unknown direction: %v", direction)
	}
}

// getAppliedMigrations retrieves all applied migrations from the database
func (e *Engine) getAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	collection := e.db.Collection(e.migrationsCollection)

	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"version": 1}))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			// Log the error but don't override the main error
			_ = closeErr
		}
	}()

	var records []MigrationRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// getMigrationsToExecute determines which migrations need to be executed
func (e *Engine) getMigrationsToExecute(
	direction Direction, target string, appliedMap map[string]bool,
) ([]string, error) {
	versions := e.getSortedVersions()

	switch direction {
	case DirectionUp:
		return e.getMigrationsForUp(versions, target, appliedMap), nil
	case DirectionDown:
		return e.getMigrationsForDown(versions, target, appliedMap), nil
	default:
		return nil, fmt.Errorf("unknown direction: %v", direction)
	}
}

// getSortedVersions returns all migration versions sorted alphabetically
func (e *Engine) getSortedVersions() []string {
	var versions []string
	for version := range e.migrations {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
}

// getMigrationsForUp gets migrations to execute for up direction
func (e *Engine) getMigrationsForUp(versions []string, target string, appliedMap map[string]bool) []string {
	var toExecute []string
	for _, version := range versions {
		if !appliedMap[version] {
			toExecute = append(toExecute, version)
			if e.shouldStopAtTarget(target, version) {
				break
			}
		}
	}
	return toExecute
}

// getMigrationsForDown gets migrations to execute for down direction
func (e *Engine) getMigrationsForDown(versions []string, target string, appliedMap map[string]bool) []string {
	var toExecute []string
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		if appliedMap[version] {
			if e.shouldStopAtTarget(target, version) {
				break
			}
			toExecute = append(toExecute, version)
		}
	}
	return toExecute
}

// shouldStopAtTarget determines if we should stop at the target version
func (e *Engine) shouldStopAtTarget(target, currentVersion string) bool {
	return target != "" && currentVersion == target
}

// calculateChecksum calculates a checksum for the migration
func (e *Engine) calculateChecksum(migration Migration) string {
	data := fmt.Sprintf("%s:%s", migration.Version(), migration.Description())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}
