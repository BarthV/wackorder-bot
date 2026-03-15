package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
)

// Repository is the backend-agnostic interface for order persistence.
// Any storage backend (SQLite, PostgreSQL, in-memory, …) must implement this.
type Repository interface {
	Create(ctx context.Context, creatorID, creatorName, component, minQuality string, quantity int) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	ListByCreator(ctx context.Context, creatorID string) ([]model.Order, error)
	ListPending(ctx context.Context) ([]model.Order, error)
	ListAll(ctx context.Context) ([]model.Order, error)
	SearchByComponent(ctx context.Context, name string) ([]model.Order, error)
	ListSince(ctx context.Context, since time.Time) ([]model.Order, error)
	UpdateStatus(ctx context.Context, id int64, newStatus model.Status, updatedBy string) error
	ListReadyByUpdater(ctx context.Context, updatedBy string) ([]model.Order, error)
}

// OrderStore is the SQLite implementation of Repository.
type OrderStore struct {
	db *sql.DB
}

// New creates an OrderStore backed by the given database connection.
func New(db *sql.DB) *OrderStore {
	return &OrderStore{db: db}
}

// Ensure OrderStore implements Repository at compile time.
var _ Repository = (*OrderStore)(nil)

// Create inserts a new order and returns its assigned ID.
func (s *OrderStore) Create(ctx context.Context, creatorID, creatorName, component, minQuality string, quantity int) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO orders (creator_id, creator_name, component, min_quality, quantity)
		 VALUES (?, ?, ?, ?, ?)`,
		creatorID, creatorName, component, minQuality, quantity,
	)
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	return id, nil
}

// GetByID retrieves a single order by its ID. Returns sql.ErrNoRows if not found.
func (s *OrderStore) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE id = ?`, id)
	return scanOrder(row)
}

// ListByCreator returns all orders placed by the given Discord user ID.
func (s *OrderStore) ListByCreator(ctx context.Context, creatorID string) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE creator_id = ? ORDER BY created_at DESC`, creatorID)
	if err != nil {
		return nil, fmt.Errorf("list by creator: %w", err)
	}
	return scanOrders(rows)
}

// ListPending returns all unfinished orders (ordered, ready).
func (s *OrderStore) ListPending(ctx context.Context) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE status IN ('ordered','ready') ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	return scanOrders(rows)
}

// ListAll returns every order regardless of status.
func (s *OrderStore) ListAll(ctx context.Context) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all: %w", err)
	}
	return scanOrders(rows)
}

// escapeLike escapes LIKE special characters so user input is treated as a literal substring.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SearchByComponent returns orders whose component matches the given name (case-insensitive).
func (s *OrderStore) SearchByComponent(ctx context.Context, name string) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE component LIKE ? ESCAPE '\' COLLATE NOCASE ORDER BY created_at DESC`,
		"%"+escapeLike(name)+"%")
	if err != nil {
		return nil, fmt.Errorf("search by component: %w", err)
	}
	return scanOrders(rows)
}

// ListSince returns all orders created at or after the given time.
func (s *OrderStore) ListSince(ctx context.Context, since time.Time) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE created_at >= ? ORDER BY created_at ASC`,
		since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("list since: %w", err)
	}
	return scanOrders(rows)
}

// UpdateStatus changes the status of an order and records who made the change.
func (s *OrderStore) UpdateStatus(ctx context.Context, id int64, newStatus model.Status, updatedBy string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		string(newStatus), updatedBy, now, id,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update status rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("order %d not found", id)
	}
	return nil
}

// ListReadyByUpdater returns all orders with status "ready" last updated by the given Discord user ID.
func (s *OrderStore) ListReadyByUpdater(ctx context.Context, updatedBy string) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, creator_id, creator_name, component, min_quality, quantity, status,
		        updated_by, created_at, updated_at
		 FROM orders WHERE status = 'ready' AND updated_by = ? ORDER BY created_at ASC`, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("list ready by updater: %w", err)
	}
	return scanOrders(rows)
}

// --- helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanOrder(row scanner) (*model.Order, error) {
	var o model.Order
	var createdAt, updatedAt string

	err := row.Scan(
		&o.ID, &o.CreatorID, &o.CreatorName, &o.Component, &o.MinQuality,
		&o.Quantity, &o.Status, &o.UpdatedBy, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		o.CreatedAt = t
	} else {
		slog.Warn("failed to parse order created_at", "order_id", o.ID, "value", createdAt, "err", err)
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		o.UpdatedAt = t
	} else {
		slog.Warn("failed to parse order updated_at", "order_id", o.ID, "value", updatedAt, "err", err)
	}

	return &o, nil
}

func scanOrders(rows *sql.Rows) ([]model.Order, error) {
	defer rows.Close()
	var orders []model.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, *o)
	}
	return orders, rows.Err()
}
