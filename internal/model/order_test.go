package model

import (
	"testing"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input   string
		want    Status
		wantErr bool
	}{
		{"ordered", StatusOrdered, false},
		{"ready", StatusReady, false},
		{"done", StatusDone, false},
		{"canceled", StatusCanceled, false},
		{"", "", true},
		{"READY", "", true},    // case-sensitive
		{"pending", "", true},  // not a valid status
		{"cancel", "", true},   // close but not valid
		{" ready ", "", true},  // no trimming
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseStatus(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidNextStatuses(t *testing.T) {
	tests := []struct {
		current Status
		want    []Status
	}{
		{StatusOrdered, []Status{StatusReady, StatusDone}},
		{StatusReady, []Status{StatusDone, StatusOrdered}},
		{StatusDone, nil},     // terminal
		{StatusCanceled, nil}, // terminal
	}

	for _, tt := range tests {
		t.Run(string(tt.current), func(t *testing.T) {
			got := ValidNextStatuses(tt.current)
			if len(got) != len(tt.want) {
				t.Fatalf("ValidNextStatuses(%q) returned %d statuses, want %d", tt.current, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidNextStatuses(%q)[%d] = %q, want %q", tt.current, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateTransition_Exhaustive(t *testing.T) {
	tests := []struct {
		name      string
		current   Status
		next      Status
		isCreator bool
		wantErr   bool
	}{
		// From ordered
		{"ordered→ready (anyone)", StatusOrdered, StatusReady, false, false},
		{"ordered→done (anyone)", StatusOrdered, StatusDone, false, false},
		{"ordered→canceled (creator)", StatusOrdered, StatusCanceled, true, false},
		{"ordered→canceled (non-creator)", StatusOrdered, StatusCanceled, false, true},
		{"ordered→ordered (invalid)", StatusOrdered, StatusOrdered, true, true},

		// From ready
		{"ready→done (anyone)", StatusReady, StatusDone, false, false},
		{"ready→ordered (anyone)", StatusReady, StatusOrdered, false, false},
		{"ready→canceled (creator)", StatusReady, StatusCanceled, true, false},
		{"ready→canceled (non-creator)", StatusReady, StatusCanceled, false, true},
		{"ready→ready (invalid)", StatusReady, StatusReady, true, true},

		// Terminal: done
		{"done→ordered", StatusDone, StatusOrdered, true, true},
		{"done→ready", StatusDone, StatusReady, true, true},
		{"done→canceled", StatusDone, StatusCanceled, true, true},
		{"done→done", StatusDone, StatusDone, true, true},

		// Terminal: canceled
		{"canceled→ordered", StatusCanceled, StatusOrdered, true, true},
		{"canceled→ready", StatusCanceled, StatusReady, true, true},
		{"canceled→done", StatusCanceled, StatusDone, true, true},
		{"canceled→canceled", StatusCanceled, StatusCanceled, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.current, tt.next, tt.isCreator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition(%q, %q, creator=%v) error = %v, wantErr %v",
					tt.current, tt.next, tt.isCreator, err, tt.wantErr)
			}
		})
	}
}
