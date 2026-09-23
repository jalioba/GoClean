package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goclean/internal/cleaner"
	"goclean/internal/scanner"
)

func TestEndToEndCleanFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goclean_e2e_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	nm := filepath.Join(tmpDir, "node_modules")
	target := filepath.Join(tmpDir, "target")
	src := filepath.Join(tmpDir, "src")
	os.MkdirAll(nm, 0755)
	os.MkdirAll(target, 0755)
	os.MkdirAll(src, 0755)

	os.WriteFile(filepath.Join(nm, "mod.js"), make([]byte, 1024*10), 0644)
	os.WriteFile(filepath.Join(target, "binary"), make([]byte, 1024*20), 0644)
	os.WriteFile(filepath.Join(src, "main.go"), make([]byte, 1024*5), 0644)

	sc := scanner.New(scanner.Options{Workers: 2})
	root, stats, err := sc.Scan(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	cleaner.TagCaches(root)
	caches := cleaner.CollectCaches(root)
	if len(caches) != 2 {
		t.Fatalf("expected 2 caches, got %d", len(caches))
	}

	if stats.TotalFiles != 3 {
		t.Fatalf("expected 3 files, got %d", stats.TotalFiles)
	}

	// Test dry run delete
	for _, c := range caches {
		if err := cleaner.DeleteSafely(c.Path, true); err != nil {
			t.Fatalf("dry run delete failed: %v", err)
		}
		if _, err := os.Stat(c.Path); os.IsNotExist(err) {
			t.Fatalf("dry run should not delete %s", c.Path)
		}
	}

	// Test real delete
	for _, c := range caches {
		if err := cleaner.DeleteSafely(c.Path, false); err != nil {
			t.Fatalf("real delete failed: %v", err)
		}
		if _, err := os.Stat(c.Path); !os.IsNotExist(err) {
			t.Fatalf("real delete should remove %s", c.Path)
		}
	}
}
