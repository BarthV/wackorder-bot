package model

import (
	"fmt"
	"time"
)

// Status represents the lifecycle state of an order.
type Status string

const (
	StatusOrdered  Status = "ordered"
	StatusReady    Status = "ready"
	StatusDone     Status = "done"
	StatusCanceled Status = "canceled"
)

// AllStatuses lists every valid status value.
var AllStatuses = []Status{
	StatusOrdered,
	StatusReady,
	StatusDone,
	StatusCanceled,
}

// ActiveStatuses lists statuses that represent unfinished orders.
var ActiveStatuses = []Status{
	StatusOrdered,
	StatusReady,
}

// ParseStatus converts a string to a Status, returning an error if invalid.
func ParseStatus(s string) (Status, error) {
	st := Status(s)
	for _, v := range AllStatuses {
		if st == v {
			return st, nil
		}
	}
	return "", fmt.Errorf("Statut invalide : %q.\nValeurs acceptées : `ready`, `done`.\nUtilise `/order-help` pour voir le workflow complet.", s)
}

// ValidNextStatuses returns the valid non-cancel target statuses from the given state.
func ValidNextStatuses(current Status) []Status {
	allowed := map[Status][]Status{
		StatusOrdered: {StatusReady, StatusDone},
		StatusReady:   {StatusDone, StatusOrdered},
	}
	return allowed[current]
}

// ValidateTransition checks whether transitioning from current to next is allowed.
// isCreator must be true when the invoking user is the order's creator.
func ValidateTransition(current, next Status, isCreator bool) error {
	// Terminal states: no outgoing transitions.
	if current == StatusDone {
		return fmt.Errorf("Cette commande est déjà terminée et ne peut plus être modifiée.")
	}
	if current == StatusCanceled {
		return fmt.Errorf("Cette commande est déjà annulée et ne peut plus être modifiée.")
	}

	// Cancel requires the caller to be the creator.
	if next == StatusCanceled {
		if !isCreator {
			return fmt.Errorf("Seul le créateur de la commande peut l'annuler.")
		}
		return nil // all non-terminal states can be canceled by the creator
	}

	// Valid forward transitions.
	allowed := map[Status][]Status{
		StatusOrdered: {StatusReady, StatusDone},
		StatusReady:   {StatusDone, StatusOrdered},
	}

	for _, a := range allowed[current] {
		if a == next {
			return nil
		}
	}

	return fmt.Errorf("Impossible de passer du statut %q à %q.", current, next)
}

// Order is the core domain entity.
type Order struct {
	ID          int64
	CreatorID   string
	CreatorName string
	Component   string
	MinQuality  string
	Quantity    int
	Status      Status
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
