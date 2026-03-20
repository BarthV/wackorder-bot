package bot

import (
	"testing"
)

func TestCanonicalResource(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantFound bool
	}{
		{"exact match", "Gold", "Gold", true},
		{"lowercase", "gold", "Gold", true},
		{"uppercase", "GOLD", "Gold", true},
		{"with spaces", "Raw Ice", "Raw Ice", true},
		{"trimmed", "  Gold  ", "Gold", true},
		{"invalid", "Unobtanium", "", false},
		{"empty", "", "", false},
		{"partial", "Gol", "", false},
		{"prefix match only", "Gold Mine", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := canonicalResource(tt.input)
			if ok != tt.wantFound {
				t.Errorf("canonicalResource(%q) found=%v, want %v", tt.input, ok, tt.wantFound)
			}
			if got != tt.wantName {
				t.Errorf("canonicalResource(%q) = %q, want %q", tt.input, got, tt.wantName)
			}
		})
	}
}

func TestFilterResources(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantMin  int // at least this many results
		wantMax  int // at most this many results
		contains string // one expected result (empty = skip check)
	}{
		{"empty query returns all up to 25", "", 25, 25, "Gold"},
		{"exact name", "Gold", 1, 2, "Gold"},         // Gold + possibly others containing "gold"
		{"case insensitive", "gold", 1, 2, "Gold"},
		{"substring", "ium", 5, 25, "Agricium"},        // Agricium, Jaclium, Lindinium, etc.
		{"no match", "zzzzz", 0, 0, ""},
		{"trimmed", "  gold  ", 1, 2, "Gold"},
		{"multi-word", "raw", 1, 5, "Raw Ice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterResources(tt.query)
			if len(got) < tt.wantMin {
				t.Errorf("filterResources(%q) returned %d results, want at least %d", tt.query, len(got), tt.wantMin)
			}
			if len(got) > tt.wantMax {
				t.Errorf("filterResources(%q) returned %d results, want at most %d", tt.query, len(got), tt.wantMax)
			}
			if tt.contains != "" {
				found := false
				for _, r := range got {
					if r == tt.contains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("filterResources(%q) does not contain %q, got %v", tt.query, tt.contains, got)
				}
			}
		})
	}
}

func TestFilterResources_MaxResults(t *testing.T) {
	// With empty query, should return exactly 25 (capped) even though there are more resources
	got := filterResources("")
	if len(got) > 25 {
		t.Errorf("filterResources(\"\") returned %d results, should cap at 25", len(got))
	}
}
