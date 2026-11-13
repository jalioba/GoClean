package cleaner

import (
	"testing"

	"goclean/internal/model"
)

func TestIsCacheDir(t *testing.T) {
	tests := []struct {
		name      string
		expected  bool
		cacheKind string
	}{
		{"node_modules", true, "npm/node"},
		{".next", true, "nextjs"},
		{"__pycache__", true, "python"},
		{".venv", true, "python-venv"},
		{".ruff_cache", true, "ruff"},
		{".terraform", true, "terraform"},
		{".dart_tool", true, "dart/flutter"},
		{".zig-cache", true, "zig"},
		{".cache", true, "general-cache"},
		{"src", false, ""},
		{"docs", false, ""},
		// Generic names without context should not match blindly
		{"target", false, ""},
		{"bin", false, ""},
		{"dist", false, ""},
		{"build", false, ""},
		{"Pods", false, ""},
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

func TestContextualCacheAndNoDoubleCount(t *testing.T) {
	root := &model.Node{Name: "proj", Path: "/proj", Type: NodeTypeDirFallback()}
	root.Type = model.NodeTypeDir

	cargoToml := &model.Node{Name: "Cargo.toml", Type: model.NodeTypeFile, Size: 100, ItemCount: 1}
	targetDir := &model.Node{Name: "target", Type: model.NodeTypeDir}
	targetFile := &model.Node{Name: "app", Type: model.NodeTypeFile, Size: 2000, ItemCount: 1}
	targetDir.AddChild(targetFile)

	nmOuter := &model.Node{Name: "node_modules", Type: model.NodeTypeDir}
	nmInner := &model.Node{Name: "node_modules", Type: model.NodeTypeDir}
	nmInnerFile := &model.Node{Name: "pkg.js", Type: model.NodeTypeFile, Size: 500, ItemCount: 1}
	nmInner.AddChild(nmInnerFile)
	nmOuter.AddChild(nmInner)

	root.AddChild(cargoToml)
	root.AddChild(targetDir)
	root.AddChild(nmOuter)
	root.PostOrderAggregate()

	cacheBytes, cacheDirs := TagCaches(root)
	if cacheDirs != 2 {
		t.Fatalf("expected 2 cache dirs (no double-count), got %d", cacheDirs)
	}
	if cacheBytes != 2500 {
		t.Fatalf("expected 2500 cache bytes, got %d", cacheBytes)
	}
}

func NodeTypeDirFallback() model.NodeType {
	return model.NodeTypeDir
}
