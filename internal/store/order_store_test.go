package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/barthv/wackorder-bot/internal/db"
	"github.com/barthv/wackorder-bot/internal/model"
	"github.com/barthv/wackorder-bot/internal/store"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.InitSchema(database); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return database
}

func newTestStore(t *testing.T) store.Repository {
	return store.New(newTestDB(t))
}

func createOrder(t *testing.T, s store.Repository) int64 {
	t.Helper()
	id, err := s.Create(context.Background(), "user1", "Alice", "Shield Generator", 0, 5)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return id
}

func TestCreate_And_GetByID(t *testing.T) {
	s := newTestStore(t)
	id := createOrder(t, s)

	o, err := s.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if o.ID != id {
		t.Errorf("expected ID %d, got %d", id, o.ID)
	}
	if o.Component != "Shield Generator" {
		t.Errorf("unexpected component: %q", o.Component)
	}
	if o.Status != model.StatusOrdered {
		t.Errorf("expected status ordered, got %q", o.Status)
	}
	if o.Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", o.Quantity)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListByCreator(t *testing.T) {
	s := newTestStore(t)

	s.Create(context.Background(), "user1", "Alice", "Shield", 0, 1)
	s.Create(context.Background(), "user2", "Bob", "Gun", 0, 2)
	s.Create(context.Background(), "user1", "Alice", "Armor", 0, 3)

	orders, err := s.ListByCreator(context.Background(), "user1")
	if err != nil {
		t.Fatalf("ListByCreator: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders for user1, got %d", len(orders))
	}
}

func TestListPending(t *testing.T) {
	s := newTestStore(t)

	id1, _ := s.Create(context.Background(), "u1", "A", "Part1", 0, 1)
	id2, _ := s.Create(context.Background(), "u2", "B", "Part2", 0, 2)
	s.Create(context.Background(), "u3", "C", "Part3", 0, 3)

	// Mark one as done, one as ready.
	s.UpdateStatus(context.Background(), id1, model.StatusDone, "tester")
	s.UpdateStatus(context.Background(), id2, model.StatusReady, "tester")

	pending, err := s.ListPending(context.Background())
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	// id1 is done (excluded), id2 is ready (included), third is ordered (included)
	if len(pending) != 2 {
		t.Errorf("expected 2 pending, got %d", len(pending))
	}
}

func TestListAll(t *testing.T) {
	s := newTestStore(t)
	s.Create(context.Background(), "u1", "A", "X", 0, 1)
	s.Create(context.Background(), "u2", "B", "Y", 0, 2)

	all, err := s.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 orders, got %d", len(all))
	}
}

func TestSearchByComponent(t *testing.T) {
	s := newTestStore(t)
	s.Create(context.Background(), "u1", "A", "Shield Generator", 0, 1)
	s.Create(context.Background(), "u1", "A", "shield gen mk2", 0, 1)
	s.Create(context.Background(), "u2", "B", "Mining Laser", 0, 1)

	results, err := s.SearchByComponent(context.Background(), "shield")
	if err != nil {
		t.Fatalf("SearchByComponent: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'shield', got %d", len(results))
	}
}

func TestListSince(t *testing.T) {
	s := newTestStore(t)
	s.Create(context.Background(), "u1", "A", "Part", 0, 1)

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	results, err := s.ListSince(context.Background(), past)
	if err != nil {
		t.Fatalf("ListSince(past): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result since past, got %d", len(results))
	}

	results, err = s.ListSince(context.Background(), future)
	if err != nil {
		t.Fatalf("ListSince(future): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results since future, got %d", len(results))
	}
}

func TestUpdateStatus_Done(t *testing.T) {
	s := newTestStore(t)
	id := createOrder(t, s)

	if err := s.UpdateStatus(context.Background(), id, model.StatusDone, "tester"); err != nil {
		t.Fatalf("UpdateStatus done: %v", err)
	}
	o, _ := s.GetByID(context.Background(), id)
	if o.Status != model.StatusDone {
		t.Errorf("expected done, got %q", o.Status)
	}
}

func TestValidateTransition_CreatorCanCancel(t *testing.T) {
	err := model.ValidateTransition(model.StatusOrdered, model.StatusCanceled, true)
	if err != nil {
		t.Errorf("creator should be able to cancel: %v", err)
	}
}

func TestValidateTransition_NonCreatorCannotCancel(t *testing.T) {
	err := model.ValidateTransition(model.StatusOrdered, model.StatusCanceled, false)
	if err == nil {
		t.Error("non-creator should not be able to cancel")
	}
}

func TestValidateTransition_DoneIsTerminal(t *testing.T) {
	err := model.ValidateTransition(model.StatusDone, model.StatusReady, true)
	if err == nil {
		t.Error("done should be terminal — no transitions allowed")
	}
}

func TestValidateTransition_CanceledIsTerminal(t *testing.T) {
	err := model.ValidateTransition(model.StatusCanceled, model.StatusDone, true)
	if err == nil {
		t.Error("canceled should be terminal — no transitions allowed")
	}
}

func TestValidateTransition_InvalidForward(t *testing.T) {
	// done is terminal — cannot transition to any other state
	err := model.ValidateTransition(model.StatusDone, model.StatusOrdered, false)
	if err == nil {
		t.Error("done → ordered should be invalid")
	}
}

func TestListBefore(t *testing.T) {
	s := newTestStore(t)
	s.Create(context.Background(), "u1", "A", "Part", 0, 1)

	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)

	results, err := s.ListBefore(context.Background(), future)
	if err != nil {
		t.Fatalf("ListBefore(future): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result before future, got %d", len(results))
	}

	results, err = s.ListBefore(context.Background(), past)
	if err != nil {
		t.Fatalf("ListBefore(past): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results before past, got %d", len(results))
	}
}

func TestListReadyByUpdater(t *testing.T) {
	s := newTestStore(t)

	id1, _ := s.Create(context.Background(), "u1", "A", "Part1", 0, 1)
	id2, _ := s.Create(context.Background(), "u2", "B", "Part2", 0, 2)
	id3, _ := s.Create(context.Background(), "u3", "C", "Part3", 0, 3)

	// Mark orders ready by different updaters.
	s.UpdateStatus(context.Background(), id1, model.StatusReady, "handler1")
	s.UpdateStatus(context.Background(), id2, model.StatusReady, "handler1")
	s.UpdateStatus(context.Background(), id3, model.StatusReady, "handler2")

	results, err := s.ListReadyByUpdater(context.Background(), "handler1")
	if err != nil {
		t.Fatalf("ListReadyByUpdater: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 ready orders for handler1, got %d", len(results))
	}

	results, err = s.ListReadyByUpdater(context.Background(), "handler2")
	if err != nil {
		t.Fatalf("ListReadyByUpdater: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 ready order for handler2, got %d", len(results))
	}

	results, err = s.ListReadyByUpdater(context.Background(), "nobody")
	if err != nil {
		t.Fatalf("ListReadyByUpdater: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 ready orders for nobody, got %d", len(results))
	}
}

func TestPrune(t *testing.T) {
	s := newTestStore(t)

	id1, _ := s.Create(context.Background(), "u1", "A", "Part1", 0, 1)
	id2, _ := s.Create(context.Background(), "u2", "B", "Part2", 0, 2)
	id3, _ := s.Create(context.Background(), "u3", "C", "Part3", 0, 3)

	// Mark id1 and id2 as done, leave id3 as ordered.
	s.UpdateStatus(context.Background(), id1, model.StatusDone, "tester")
	s.UpdateStatus(context.Background(), id2, model.StatusDone, "tester")

	// Prune with a future cutoff — should delete both done orders.
	future := time.Now().Add(1 * time.Hour)
	count, err := s.Prune(context.Background(), future)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 pruned, got %d", count)
	}

	// Ordered order should still exist.
	all, _ := s.ListAll(context.Background())
	if len(all) != 1 {
		t.Errorf("expected 1 remaining order, got %d", len(all))
	}
	if all[0].ID != id3 {
		t.Errorf("expected remaining order to be id3=%d, got %d", id3, all[0].ID)
	}
}

func TestPrune_DoesNotDeleteRecentDone(t *testing.T) {
	s := newTestStore(t)

	id, _ := s.Create(context.Background(), "u1", "A", "Part", 0, 1)
	s.UpdateStatus(context.Background(), id, model.StatusDone, "tester")

	// Prune with a past cutoff — should not delete the recent done order.
	past := time.Now().Add(-1 * time.Hour)
	count, err := s.Prune(context.Background(), past)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pruned with past cutoff, got %d", count)
	}
}

func TestUpdateStatus_SetsUpdatedBy(t *testing.T) {
	s := newTestStore(t)
	id := createOrder(t, s)

	if err := s.UpdateStatus(context.Background(), id, model.StatusReady, "handler42"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	o, _ := s.GetByID(context.Background(), id)
	if o.UpdatedBy != "handler42" {
		t.Errorf("expected UpdatedBy=handler42, got %q", o.UpdatedBy)
	}
	if o.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero after update")
	}
}

func TestSearchByComponent_SpecialChars(t *testing.T) {
	s := newTestStore(t)
	s.Create(context.Background(), "u1", "A", "100% Pure Gold", 0, 1)
	s.Create(context.Background(), "u1", "A", "Shield_Gen", 0, 1)
	s.Create(context.Background(), "u1", "A", "Normal Item", 0, 1)

	// Search for "%" — should match only the order containing "%"
	results, err := s.SearchByComponent(context.Background(), "%")
	if err != nil {
		t.Fatalf("SearchByComponent(%%): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for '%%', got %d", len(results))
	}

	// Search for "_" — should match only the order containing "_"
	results, err = s.SearchByComponent(context.Background(), "_")
	if err != nil {
		t.Fatalf("SearchByComponent(_): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for '_', got %d", len(results))
	}
}
