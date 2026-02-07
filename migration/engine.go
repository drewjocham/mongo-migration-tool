package migration

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultLockID  = "migration_engine_lock"
	collLock       = "migrations_lock"
	collMigrations = "schema_migrations"

	// Log messages
	logExecutingMigration = "Executing migration"
)

type Migration interface {
	Version() string
	Description() string
	Up(ctx context.Context, db *mongo.Database) error
	Down(ctx context.Context, db *mongo.Database) error
}

type MigrationRecord struct {
	Version     string    `bson:"version"`
	Description string    `bson:"description"`
	AppliedAt   time.Time `bson:"applied_at"`
	Checksum    string    `bson:"checksum"`
}

type MigrationStatus struct {
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Applied     bool       `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
}

type Engine struct {
	db         *mongo.Database
	migrations map[string]Migration
	coll       string
}

func NewEngine(db *mongo.Database, migrationsCollection string, migrations map[string]Migration) *Engine {
	if migrationsCollection == "" {
		migrationsCollection = collMigrations
	}
	return &Engine{
		db:         db,
		migrations: migrations,
		coll:       migrationsCollection,
	}
}

func (e *Engine) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrFailedToReadMigrations, err)
	}

	versions := e.getSortedVersions()
	status := make([]MigrationStatus, len(versions))

	for i, v := range versions {
		rec, isApplied := applied[v]
		status[i] = MigrationStatus{
			Version:     v,
			Description: e.migrations[v].Description(),
			Applied:     isApplied,
		}
		if isApplied {
			status[i].AppliedAt = &rec.AppliedAt
		}
	}

	return status, nil
}

func (e *Engine) Up(ctx context.Context, target string) error {
	return e.run(ctx, DirectionUp, target)
}

func (e *Engine) Down(ctx context.Context, target string) error {
	return e.run(ctx, DirectionDown, target)
}

func (e *Engine) Force(ctx context.Context, version string) error {
	m, ok := e.migrations[version]
	if !ok {
		return fmt.Errorf("%s: %s", ErrMigrationNotFound, version)
	}

	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrFailedToReadMigrations, err)
	}

	if _, isApplied := applied[version]; isApplied {
		// Migration is already applied, nothing to do.
		return nil
	}

	coll := e.db.Collection(e.coll)
	_, err = coll.InsertOne(ctx, e.newRecord(m))
	if err != nil {
		return fmt.Errorf("%s: %w", ErrFailedToSetVersion, err)
	}

	return nil
}

func (e *Engine) run(ctx context.Context, dir Direction, target string) error {
	if err := e.acquireLock(ctx); err != nil {
		return err
	}
	defer e.releaseLock(context.Background())

	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return err
	}

	plan, err := e.planExecution(dir, target, applied)
	if err != nil {
		return err
	}

	for _, version := range plan {
		m := e.migrations[version]

		if dir == DirectionUp {
			if rec, ok := applied[version]; ok {
				if err := e.validateChecksum(m, rec); err != nil {
					return err
				}
			}
		}

		slog.Info(logExecutingMigration, "version", version, "direction", dir)
		if err := e.executeWithRetry(ctx, m, dir); err != nil {
			return fmt.Errorf("%s: %w", ErrFailedToRunMigration, err)
		}
	}

	return nil
}

func (e *Engine) Plan(ctx context.Context, dir Direction, target string) ([]string, error) {
	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("plan migrations: %w", err)
	}
	return e.planExecution(dir, target, applied)
}
func (e *Engine) executeWithRetry(ctx context.Context, m Migration, dir Direction) error {
	session, err := e.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sCtx mongo.SessionContext) (interface{}, error) {
		return nil, e.perform(sCtx, m, dir)
	})

	if err != nil && isTransactionNotSupported(err) {
		return e.perform(ctx, m, dir)
	}

	return err
}

func (e *Engine) planExecution(dir Direction, target string, applied map[string]MigrationRecord) ([]string, error) {
	versions := e.getSortedVersions()

	if dir == DirectionDown {
		slices.Reverse(versions)
	}

	var plan []string
	for _, v := range versions {
		_, isApplied := applied[v]

		if dir == DirectionUp && !isApplied {
			plan = append(plan, v)
		} else if dir == DirectionDown && isApplied {
			plan = append(plan, v)
		}

		if target != "" && v == target {
			break
		}
	}
	return plan, nil
}

func (e *Engine) getSortedVersions() []string {
	versions := make([]string, 0, len(e.migrations))
	for v := range e.migrations {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

func (e *Engine) perform(ctx context.Context, m Migration, dir Direction) error {
	coll := e.db.Collection(e.coll)
	version := m.Version()

	if dir == DirectionUp {
		if err := m.Up(ctx, e.db); err != nil {
			return err
		}
		_, err := coll.InsertOne(ctx, e.newRecord(m))
		return err
	}

	if err := m.Down(ctx, e.db); err != nil {
		return err
	}
	_, err := coll.DeleteOne(ctx, bson.M{"version": version})
	return err
}

func (e *Engine) getAppliedMap(ctx context.Context) (map[string]MigrationRecord, error) {
	coll := e.db.Collection(e.coll)
	cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"version": 1}))
	if err != nil {
		return nil, err
	}

	var records []MigrationRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	applied := make(map[string]MigrationRecord, len(records))
	for _, r := range records {
		applied[r.Version] = r
	}
	return applied, nil
}

func (e *Engine) validateChecksum(m Migration, record MigrationRecord) error {
	if record.Checksum != e.calculateChecksum(m) {
		return fmt.Errorf(
			"checksum mismatch for %s: expected %s, got %s",
			m.Version(), record.Checksum, e.calculateChecksum(m),
		)
	}
	return nil
}

func (e *Engine) calculateChecksum(m Migration) string {
	data := fmt.Sprintf("%s:%s", m.Version(), m.Description())
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}

func (e *Engine) newRecord(m Migration) MigrationRecord {
	return MigrationRecord{
		Version:     m.Version(),
		Description: m.Description(),
		AppliedAt:   time.Now().UTC(),
		Checksum:    e.calculateChecksum(m),
	}
}

func isTransactionNotSupported(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case 20, 251, 303: // IllegalOperation, NoSuchTransaction, TransactionNotSupportedInShardedCluster
			return true
		}
		if strings.Contains(strings.ToLower(cmdErr.Message), "transactions are not supported") {
			return true
		}
	}

	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		if writeErr.WriteConcernError != nil {
			switch writeErr.WriteConcernError.Code {
			case 20, 251, 303:
				return true
			}
			if strings.Contains(strings.ToLower(writeErr.WriteConcernError.Message), "transactions are not supported") {
				return true
			}
		}
	}

	return strings.Contains(strings.ToLower(err.Error()), "transactions are not supported")
}

func (e *Engine) acquireLock(ctx context.Context) error {
	coll := e.db.Collection(collLock)

	_, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "acquired_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(600),
	})
	_, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "lock_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	_, err := coll.InsertOne(ctx, bson.M{
		"lock_id":     defaultLockID,
		"acquired_at": time.Now().UTC(),
	})

	if mongo.IsDuplicateKeyError(err) {
		return ErrFailedToLock
	}
	return err
}

func (e *Engine) releaseLock(ctx context.Context) {
	coll := e.db.Collection(collLock)
	_, _ = coll.DeleteOne(ctx, bson.M{"lock_id": defaultLockID})
}

func (e *Engine) ForceUnlock(ctx context.Context) error {
	coll := e.db.Collection(collLock)
	_, err := coll.DeleteMany(ctx, bson.M{"lock_id": defaultLockID})
	return err
}
