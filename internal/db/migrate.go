package db

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// migration holds a versioned schema change.
type migration struct {
	version int
	name    string
	sql     string
}

// migrations is the ordered list of all schema versions.
// To add a new migration: append a new entry with the next version number.
// Never edit or delete existing entries — only append.
var migrations = []migration{
	{
		version: 1,
		name:    "create_orders_table",
		sql: `
CREATE TABLE IF NOT EXISTS orders (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	creator_id   TEXT    NOT NULL,
	creator_name TEXT    NOT NULL,
	component    TEXT    NOT NULL,
	min_quality  TEXT    NOT NULL DEFAULT '',
	quantity     INTEGER NOT NULL CHECK (quantity > 0),
	status       TEXT    NOT NULL DEFAULT 'ordered'
	             CHECK (status IN ('ordered','ready','in-transit','done','canceled')),
	meeting_date TEXT,
	created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_orders_creator_id ON orders(creator_id);
CREATE INDEX IF NOT EXISTS idx_orders_status     ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_component  ON orders(component COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
`,
	},
	// Add future migrations here, e.g.:
	// {
	//   version: 2,
	//   name:    "add_components_table",
	//   sql:     `CREATE TABLE components (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE);`,
	// },
}

// Migrate creates the schema_migrations tracking table and applies any
// unapplied migrations in version order. Each migration runs in its own
// transaction and is recorded atomically.
func Migrate(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}

		slog.Info("applying migration", "version", m.version, "name", m.name)
		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("migration v%d %q: %w", m.version, m.name, err)
		}
		slog.Info("migration applied", "version", m.version, "name", m.name)
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT    NOT NULL,
			applied_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`,
		m.version, m.name,
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}
