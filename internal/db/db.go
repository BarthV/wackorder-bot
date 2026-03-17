package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at the given path and applies WAL/FK pragmas.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	// SQLite is single-writer. Limit to one connection so per-connection
	// pragmas (foreign_keys, busy_timeout) apply to every query.
	db.SetMaxOpenConns(1)

	return db, nil
}

// InitSchema creates the orders table and indexes if they don't exist yet.
func InitSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS orders (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	creator_id   TEXT    NOT NULL,
	creator_name TEXT    NOT NULL,
	component    TEXT    NOT NULL,
	min_quality  INTEGER NOT NULL DEFAULT 0,
	quantity     INTEGER NOT NULL CHECK (quantity > 0),
	status       TEXT    NOT NULL DEFAULT 'ordered'
	             CHECK (status IN ('ordered','ready','done','canceled')),
	created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_by   TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_orders_creator_id ON orders(creator_id);
CREATE INDEX IF NOT EXISTS idx_orders_status     ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_component  ON orders(component COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
`)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}
