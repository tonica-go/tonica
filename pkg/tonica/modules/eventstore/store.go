package eventstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"

	// native drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

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
	switch dialectName {
	case "pg":
		dialect = "postgres"
	case "mysql":
		dialect = "mysql"
	case "sqlite", "sqlite3":
		dialect = "sqlite"
	}

	store := &sqlStore{db: sqlDB, dialect: dialect}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func newSQLiteStore(ctx context.Context, dsn string) (Store, error) {
	db, err := sql.Open("sqlite", dsn)
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
	db, err := sql.Open("sqlite", dsn)
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

func newMySQLStore(ctx context.Context, dsn string) (Store, error) {
	db, err := sql.Open("postgres", dsn)
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
		return errors.New("concurrency conflict")
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
			return err
		}
	}

	if err := tx.Commit(); err != nil {
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
