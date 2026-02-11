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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	defaultLockID         = "migration_engine_lock"
	collLock              = "migrations_lock"
	collMigrations        = "schema_migrations"
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

func NewEngine(db *mongo.Database, coll string, migrations map[string]Migration) *Engine {
	if coll == "" {
		coll = collMigrations
	}
	return &Engine{db: db, migrations: migrations, coll: coll}
}

func (e *Engine) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToReadMigrations, err)
	}

	versions := e.getSortedVersions(DirectionUp)
	status := make([]MigrationStatus, len(versions))

	for i, v := range versions {
		m := e.migrations[v]
		rec, isApplied := applied[v]
		status[i] = MigrationStatus{
			Version:     v,
			Description: m.Description(),
			Applied:     isApplied,
		}
		if isApplied {
			status[i].AppliedAt = &rec.AppliedAt
		}
	}
	return status, nil
}

func (e *Engine) Up(ctx context.Context, target string) error { return e.run(ctx, DirectionUp, target) }
func (e *Engine) Down(ctx context.Context, target string) error {
	return e.run(ctx, DirectionDown, target)
}

func (e *Engine) ListApplied(ctx context.Context) ([]MigrationRecord, error) {
	coll := e.db.Collection(e.coll)
	cur, err := coll.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "applied_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToReadMigrations, err)
	}
	defer cur.Close(ctx)

	var records []MigrationRecord
	if err := cur.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToReadMigrations, err)
	}
	return records, nil
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

	if _, exists := applied[version]; exists {
		return nil
	}

	coll := e.db.Collection(e.coll)
	if _, err := coll.InsertOne(ctx, e.newRecord(m)); err != nil {
		return fmt.Errorf("%s: %w", ErrFailedToSetVersion, err)
	}
	return nil
}

func (e *Engine) run(ctx context.Context, dir Direction, target string) error {
	if err := e.acquireLock(ctx); err != nil {
		return err
	}
	defer e.releaseLock(context.Background()) // to release on cancel

	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return err
	}

	plan, err := e.Plan(ctx, dir, target)
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
			return fmt.Errorf("%w: %s: %w", ErrFailedToRunMigration, version, err)
		}
	}
	return nil
}

func (e *Engine) Plan(ctx context.Context, dir Direction, target string) ([]string, error) {
	applied, err := e.getAppliedMap(ctx)
	if err != nil {
		return nil, err
	}

	versions := e.getSortedVersions(dir)
	var plan []string

	for _, v := range versions {
		_, isApplied := applied[v]
		shouldInclude := (dir == DirectionUp && !isApplied) || (dir == DirectionDown && isApplied)

		if shouldInclude {
			plan = append(plan, v)
		}
		if target != "" && v == target {
			break
		}
	}
	return plan, nil
}

func (e *Engine) ForceUnlock(ctx context.Context) error {
	coll := e.db.Collection(collLock)
	_, err := coll.DeleteMany(ctx, bson.M{"lock_id": defaultLockID})
	if err != nil {
		return fmt.Errorf("failed to force unlock: %w", err)
	}
	return nil
}

func (e *Engine) executeWithRetry(ctx context.Context, m Migration, dir Direction) error {
	work := func(sCtx context.Context) error { return e.perform(sCtx, m, dir) }
	session, err := e.db.Client().StartSession()
	if err != nil {
		return work(ctx)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sCtx context.Context) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}
		if err := work(sCtx); err != nil {
			_ = session.AbortTransaction(sCtx)
			return err
		}
		return session.CommitTransaction(sCtx)
	})

	if err != nil && isTransactionNotSupported(err) {
		return work(ctx)
	}
	return err
}

func (e *Engine) perform(ctx context.Context, m Migration, dir Direction) error {
	coll := e.db.Collection(e.coll)
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
	_, err := coll.DeleteOne(ctx, bson.M{"version": m.Version()})
	return err
}

func (e *Engine) getSortedVersions(dir Direction) []string {
	versions := make([]string, 0, len(e.migrations))
	for v := range e.migrations {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	if dir == DirectionDown {
		slices.Reverse(versions)
	}
	return versions
}

func (e *Engine) getAppliedMap(ctx context.Context) (map[string]MigrationRecord, error) {
	cursor, err := e.db.Collection(e.coll).Find(ctx, bson.M{})
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
	if current := e.calculateChecksum(m); record.Checksum != current {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", m.Version(), record.Checksum, current)
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

func (e *Engine) acquireLock(ctx context.Context) error {
	coll := e.db.Collection(collLock)

	_, _ = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "acquired_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(600)},
		{Keys: bson.D{{Key: "lock_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	})

	_, err := coll.InsertOne(ctx, bson.M{"lock_id": defaultLockID, "acquired_at": time.Now().UTC()})
	if mongo.IsDuplicateKeyError(err) {
		return ErrFailedToLock
	}
	return err
}

func (e *Engine) releaseLock(ctx context.Context) {
	_, _ = e.db.Collection(collLock).DeleteOne(ctx, bson.M{"lock_id": defaultLockID})
}

func isTransactionNotSupported(err error) bool {
	msg := strings.ToLower(err.Error())
	isCodeMatch := false

	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		isCodeMatch = slices.Contains([]int32{20, 251, 303}, cmdErr.Code)
	}

	return isCodeMatch || strings.Contains(msg, "transactions are not supported")
}
