package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
)

func makeOrder(daysAgo int) model.Order {
	return model.Order{
		Component: "Gold",
		Quantity:  1,
		Status:    model.StatusOrdered,
		CreatedAt: time.Now().UTC().Truncate(24*time.Hour).Add(-time.Duration(daysAgo) * 24 * time.Hour),
	}
}

func TestBuildHistogram_Empty(t *testing.T) {
	result := buildHistogram(nil, 10, 6)
	if result != "" {
		t.Errorf("expected empty string for no orders, got %q", result)
	}
}

func TestBuildHistogram_SingleOrder_Today(t *testing.T) {
	orders := []model.Order{makeOrder(0)}
	result := buildHistogram(orders, 10, 6)

	if result == "" {
		t.Fatal("expected non-empty histogram")
	}

	lines := strings.Split(result, "\n")
	// Should have: maxHeight bar rows + x-axis line + x-axis labels = 6+1+1 = 8
	if len(lines) != 8 {
		t.Errorf("expected 8 lines, got %d:\n%s", len(lines), result)
	}

	// Last column (rightmost = today) should have a bar
	barLines := lines[:6]
	lastBarHasBlock := false
	for _, line := range barLines {
		if strings.ContainsRune(line, '█') {
			lastBarHasBlock = true
			// The block should be at the rightmost column position
			runes := []rune(line)
			if runes[len(runes)-1] != '█' {
				t.Errorf("expected bar at rightmost column, got line: %q", line)
			}
		}
	}
	if !lastBarHasBlock {
		t.Error("expected at least one bar row with a block")
	}

	// X-axis should contain arrow
	xAxisLine := lines[6]
	if !strings.HasSuffix(xAxisLine, "→") {
		t.Errorf("x-axis should end with arrow, got %q", xAxisLine)
	}

	// Labels should contain "now"
	labelLine := lines[7]
	if !strings.Contains(labelLine, "now") {
		t.Errorf("labels should contain 'now', got %q", labelLine)
	}
}

func TestBuildHistogram_OldOrders_CatchAllBucket(t *testing.T) {
	// Orders older than maxCols-1 days should land in the leftmost bucket
	orders := []model.Order{
		makeOrder(100), // much older than 10 columns
		makeOrder(50),  // also older
	}
	result := buildHistogram(orders, 10, 6)

	if result == "" {
		t.Fatal("expected non-empty histogram")
	}

	// Both should be in leftmost bucket, so only the first column has bars
	lines := strings.Split(result, "\n")
	for _, line := range lines[:6] {
		if !strings.ContainsRune(line, '█') {
			continue
		}
		// In runes, the block should be right after "│"
		runes := []rune(line)
		pipeIdx := -1
		for j, r := range runes {
			if r == '│' {
				pipeIdx = j
				break
			}
		}
		blockIdx := -1
		for j, r := range runes {
			if r == '█' {
				blockIdx = j
				break
			}
		}
		if blockIdx != pipeIdx+1 {
			t.Errorf("expected bar at leftmost column (rune pos %d), got at %d in: %q", pipeIdx+1, blockIdx, line)
		}
	}
}

func TestBuildHistogram_MultipleOrders_SameDay(t *testing.T) {
	orders := []model.Order{
		makeOrder(0),
		makeOrder(0),
		makeOrder(0),
	}
	result := buildHistogram(orders, 10, 6)
	if result == "" {
		t.Fatal("expected non-empty histogram")
	}

	// Y-axis max value should be 3
	lines := strings.Split(result, "\n")
	if !strings.Contains(lines[0], "3") {
		t.Errorf("expected y-axis max of 3 in top row, got: %q", lines[0])
	}
}

func TestBuildXAxisLabels(t *testing.T) {
	tests := []struct {
		name    string
		maxCols int
	}{
		{"small", 8},
		{"default", 32},
		{"min", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildXAxisLabels(tt.maxCols)

			// Length should match maxCols
			if len([]rune(result)) != tt.maxCols {
				t.Errorf("expected label length %d, got %d: %q", tt.maxCols, len([]rune(result)), result)
			}

			// Should contain the left label with days count
			leftLabel := "+" + strings.TrimLeft(result, " +")
			if !strings.Contains(result, "+") {
				t.Errorf("expected '+Nd' left label, got %q", result)
			}
			_ = leftLabel

			// Should contain "now" at the right if there's enough space
			if tt.maxCols >= 8 && !strings.Contains(result, "now") {
				t.Errorf("expected 'now' label for maxCols=%d, got %q", tt.maxCols, result)
			}
		})
	}
}

func TestBuildXAxisLabels_IntermediateLabels(t *testing.T) {
	// With 32 columns, we should have intermediate labels at every 8 cols
	result := buildXAxisLabels(32)
	runes := []rune(result)

	// Position 0 should start with "+31d"
	if string(runes[:4]) != "+31d" {
		t.Errorf("expected '+31d' at start, got %q", string(runes[:4]))
	}

	// Should end with "now"
	if string(runes[29:32]) != "now" {
		t.Errorf("expected 'now' at end, got %q", string(runes[29:32]))
	}
}
