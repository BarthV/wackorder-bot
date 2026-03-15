package model

import (
	"fmt"
	"time"
)

// Status represents the lifecycle state of an order.
type Status string

const (
	StatusOrdered   Status = "ordered"
	StatusReady     Status = "ready"
	StatusInTransit Status = "in-transit"
	StatusDone      Status = "done"
	StatusCanceled  Status = "canceled"
)

// AllStatuses lists every valid status value.
var AllStatuses = []Status{
	StatusOrdered,
	StatusReady,
	StatusInTransit,
	StatusDone,
	StatusCanceled,
}

// ActiveStatuses lists statuses that represent unfinished orders.
var ActiveStatuses = []Status{
	StatusOrdered,
	StatusReady,
	StatusInTransit,
}

// ParseStatus converts a string to a Status, returning an error if invalid.
func ParseStatus(s string) (Status, error) {
	st := Status(s)
	for _, v := range AllStatuses {
		if st == v {
			return st, nil
		}
	}
	return "", fmt.Errorf("invalid status %q: must be one of ordered, ready, in-transit, done, canceled", s)
}

// RequiresMeetingDate returns true when the target status requires a meeting date.
func RequiresMeetingDate(next Status) bool {
	return next == StatusInTransit
}

// ValidateTransition checks whether transitioning from current to next is allowed.
// isCreator must be true when the invoking user is the order's creator.
func ValidateTransition(current, next Status, isCreator bool) error {
	// Terminal states: no outgoing transitions.
	if current == StatusDone {
		return fmt.Errorf("order is already done and cannot be changed")
	}
	if current == StatusCanceled {
		return fmt.Errorf("order is already canceled and cannot be changed")
	}

	// Cancel requires the caller to be the creator.
	if next == StatusCanceled {
		if !isCreator {
			return fmt.Errorf("only the order creator can cancel it")
		}
		return nil // all non-terminal states can be canceled by the creator
	}

	// Valid forward transitions.
	allowed := map[Status][]Status{
		StatusOrdered:   {StatusReady, StatusInTransit, StatusDone},
		StatusReady:     {StatusInTransit, StatusDone},
		StatusInTransit: {StatusDone},
	}

	for _, a := range allowed[current] {
		if a == next {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %q to %q", current, next)
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
	MeetingDate *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
