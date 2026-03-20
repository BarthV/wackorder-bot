package bot

import (
	"testing"
)

func TestParseDurationDays(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		// Plain integers (days)
		{"7", 7, false},
		{"1", 1, false},
		{"30", 30, false},

		// Days suffix
		{"7d", 7, false},
		{"1d", 1, false},
		{"14d", 14, false},

		// Weeks suffix
		{"2w", 14, false},
		{"1w", 7, false},
		{"4w", 28, false},

		// Months suffix (30 days each)
		{"1mo", 30, false},
		{"2mo", 60, false},
		{"3mo", 90, false},

		// Errors
		{"", 0, true},
		{"0", 0, true},
		{"-1", 0, true},
		{"0d", 0, true},
		{"-1w", 0, true},
		{"0mo", 0, true},
		{"abc", 0, true},
		{"d", 0, true},
		{"w", 0, true},
		{"mo", 0, true},

		// Whitespace handling
		{"  7d  ", 7, false},
		{" 2w ", 14, false},

		// Case insensitivity
		{"7D", 7, false},
		{"2W", 14, false},
		{"1MO", 30, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDurationDays(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDurationDays(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseDurationDays(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
