package ui

import (
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1048576 * 5, "5.00 MB"},
		{1073741824 * 3, "3.00 GB"},
	}

	for _, tt := range tests {
		res := FormatBytes(tt.bytes)
		if res != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.bytes, res, tt.expected)
		}
	}
}

func TestRenderBar(t *testing.T) {
	bar := RenderBar(0.5, 10, false)
	if !strings.Contains(bar, "█████") {
		t.Errorf("expected 50%% bar of width 10 to contain 5 full blocks, got %s", bar)
	}
}
