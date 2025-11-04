package eventstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	// native drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// ErrConcurrencyConflict indicates an optimistic concurrency check failed.
// This error should be retried by the caller.
var ErrConcurrencyConflict = errors.New("concurrency conflict: aggregate was modified concurrently")

// Event represents a domain event persisted in the store.
type Event struct {
	ID            int64
	AggregateID   string
	AggregateType string
	Version       int64
	Type          string
	Payload       []byte
	Metadata      []byte
}

// Store defines the event store contract.
type Store interface {
	Append(ctx context.Context, streamID string, expectedVersion int64, events []Event) error
	Load(ctx context.Context, streamID string, fromVersion int64) ([]Event, error)
	Close(ctx context.Context) error
}

type sqlStore struct {
	db      *sql.DB
	dialect string
}

// New creates a Store based on configuration.
func New(ctx context.Context, driver, dsn string) (Store, error) {
	switch driver {
	case "postgres":
		return newPostgresStore(ctx, dsn)
	case "sqlite":
		return newSQLiteStore(ctx, dsn)
	case "mysql":
		return newMySQLStore(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported event store driver: %s", driver)
	}
}

// NewFromBun creates a Store from a bun.DB instance.
// This is useful for integration with tonica framework.
func NewFromBun(ctx context.Context, bunDB *bun.DB) (Store, error) {
	if bunDB == nil {
		return nil, fmt.Errorf("bunDB cannot be nil")
	}

	// Extract the underlying sql.DB from bun
	sqlDB := bunDB.DB

	// Detect dialect from bun driver name
	dialect := "postgres"
	dialectName := bunDB.Dialect().Name().String()
	var dbSystem string
	switch dialectName {
	case "pg":
		dialect = "postgres"
		dbSystem = "postgresql"
	case "mysql":
		dialect = "mysql"
		dbSystem = "mysql"
	case "sqlite", "sqlite3":
		dialect = "sqlite"
		dbSystem = "sqlite"
	}

	// Register OpenTelemetry metrics for sql.DB stats
	// This will expose connection pool metrics
	otelsql.ReportDBStatsMetrics(sqlDB,
		otelsql.WithAttributes(semconv.DBSystemKey.String(dbSystem)),
		otelsql.WithDBName(dialect),
	)

	store := &sqlStore{db: sqlDB, dialect: dialect}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func newSQLiteStore(ctx context.Context, dsn string) (Store, error) {
	// Use otelsql.Open to get a traced sql.DB
	db, err := otelsql.Open("sqlite", dsn,
		otelsql.WithAttributes(semconv.DBSystemKey.String("sqlite")),
		otelsql.WithDBName("sqlite"),
	)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	store := &sqlStore{db: db, dialect: "sqlite"}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func newPostgresStore(ctx context.Context, dsn string) (Store, error) {
	// Use otelsql.Open to get a traced sql.DB
	db, err := otelsql.Open("postgres", dsn,
		otelsql.WithAttributes(semconv.DBSystemKey.String("postgresql")),
		otelsql.WithDBName("postgres"),
	)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	store := &sqlStore{db: db, dialect: "postgres"}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func newMySQLStore(ctx context.Context, dsn string) (Store, error) {
	// Use otelsql.Open to get a traced sql.DB
	db, err := otelsql.Open("mysql", dsn,
		otelsql.WithAttributes(semconv.DBSystemKey.String("mysql")),
		otelsql.WithDBName("mysql"),
	)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	store := &sqlStore{db: db, dialect: "mysql"}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *sqlStore) ensureSchema(ctx context.Context) error {
	var schema string
	switch s.dialect {
	case "mysql":
		schema = `
CREATE TABLE IF NOT EXISTS events (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	aggregate_id CHAR(36) NOT NULL,
	aggregate_type VARCHAR(255) NOT NULL,
	version BIGINT NOT NULL,
	type VARCHAR(255) NOT NULL,
	payload LONGBLOB NOT NULL,
	metadata LONGBLOB,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE KEY idx_events_stream_version (aggregate_id, version),
	KEY idx_events_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	case "postgres":
		schema = `
CREATE TABLE IF NOT EXISTS events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id text NOT NULL,
    aggregate_type VARCHAR(255) NOT NULL,
    version BIGINT NOT NULL,
    "type" VARCHAR(255) NOT NULL,
    payload BYTEA NOT NULL,
    metadata BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_created_at ON events (created_at);
`
	case "sqlite":
		schema = `
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	aggregate_id TEXT NOT NULL,
	aggregate_type TEXT NOT NULL,
	version INTEGER NOT NULL,
	type TEXT NOT NULL,
	payload BLOB NOT NULL,
	metadata BLOB,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_stream_version ON events(aggregate_id, version);
`
	default:
		return fmt.Errorf("unsupported dialect: %s", s.dialect)
	}

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *sqlStore) Append(ctx context.Context, streamID string, expectedVersion int64, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // best effort

	var currentVersion sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT MAX(version) FROM events WHERE aggregate_id = $1", streamID).Scan(&currentVersion)
	if err != nil {
		return err
	}

	if expectedVersion >= 0 && currentVersion.Valid && currentVersion.Int64 != expectedVersion {
		return ErrConcurrencyConflict
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO events (aggregate_id, aggregate_type, version, type, payload, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	version := currentVersion.Int64
	for _, evt := range events {
		version++
		if _, err := stmt.ExecContext(ctx, streamID, evt.AggregateType, version, evt.Type, evt.Payload, evt.Metadata); err != nil {
			// Check if error is a unique constraint violation on (aggregate_id, version)
			// This can happen in race conditions even after the version check
			if isUniqueConstraintViolation(err) {
				return ErrConcurrencyConflict
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		// Check if commit fails due to unique constraint violation
		if isUniqueConstraintViolation(err) {
			return ErrConcurrencyConflict
		}
		return err
	}

	//logging.FromContext(ctx).Debug("appended events", zap.String("stream_id", streamID), zap.Int("count", len(events)))
	return nil
}

func (s *sqlStore) Load(ctx context.Context, streamID string, fromVersion int64) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, aggregate_id, aggregate_type, version, type, payload, metadata
FROM events
WHERE aggregate_id = $1 AND version >= $2
ORDER BY version
`, streamID, fromVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var evt Event
		if err := rows.Scan(&evt.ID, &evt.AggregateID, &evt.AggregateType, &evt.Version, &evt.Type, &evt.Payload, &evt.Metadata); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, rows.Err()
}

func (s *sqlStore) Close(ctx context.Context) error {
	return s.db.Close()
}

// isUniqueConstraintViolation checks if an error is a unique constraint violation
// across different database drivers
func isUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: error code 23505
	// Error message contains: "duplicate key value violates unique constraint"
	if containsAny(errMsg, "duplicate key", "23505", "unique constraint") {
		return true
	}

	// MySQL: error code 1062
	// Error message contains: "Duplicate entry"
	if containsAny(errMsg, "Duplicate entry", "1062") {
		return true
	}

	// SQLite: UNIQUE constraint failed
	if containsAny(errMsg, "UNIQUE constraint failed") {
		return true
	}

	return false
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
