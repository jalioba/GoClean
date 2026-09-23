package cleaner

import (
	"testing"
)

func TestIsCacheDir(t *testing.T) {
	tests := []struct {
		name      string
		expected  bool
		cacheKind string
	}{
		{"node_modules", true, "npm/node"},
		{".next", true, "nextjs"},
		{"target", true, "rust/cargo"},
		{"__pycache__", true, "python"},
		{".venv", true, "python-venv"},
		{".cache", true, "general-cache"},
		{"src", false, ""},
		{"docs", false, ""},
	}

	for _, tt := range tests {
		isCache, kind := DetectCache(tt.name)
		if isCache != tt.expected {
			t.Errorf("DetectCache(%s) isCache = %v; want %v", tt.name, isCache, tt.expected)
		}
		if isCache && kind != tt.cacheKind {
			t.Errorf("DetectCache(%s) kind = %s; want %s", tt.name, kind, tt.cacheKind)
		}
	}
}
