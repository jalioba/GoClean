package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConcurrentScanner(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goclean_scan_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	sub1 := filepath.Join(tempDir, "sub1")
	sub2 := filepath.Join(tempDir, "sub2")
	sub3 := filepath.Join(sub2, "sub3")
	os.MkdirAll(sub1, 0755)
	os.MkdirAll(sub3, 0755)

	os.WriteFile(filepath.Join(tempDir, "file1.txt"), make([]byte, 100), 0644)
	os.WriteFile(filepath.Join(sub1, "file2.txt"), make([]byte, 200), 0644)
	os.WriteFile(filepath.Join(sub3, "file3.txt"), make([]byte, 300), 0644)

	opts := Options{
		Workers: 4,
	}
	sc := New(opts)
	root, stats, err := sc.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if root.Size != 600 {
		t.Errorf("expected total size 600, got %d", root.Size)
	}
	if stats.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", stats.TotalFiles)
	}
}

func TestScannerDeepNested(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goclean_deep_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	curr := tempDir
	for i := 0; i < 20; i++ {
		curr = filepath.Join(curr, fmt.Sprintf("dir_%d", i))
		if err := os.Mkdir(curr, 0755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(curr, "data.bin"), make([]byte, 50), 0644)
	}

	sc := New(Options{Workers: 8})
	root, stats, err := sc.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if stats.TotalFiles != 20 {
		t.Errorf("expected 20 files, got %d", stats.TotalFiles)
	}
	if root.Size != 1000 {
		t.Errorf("expected size 1000, got %d", root.Size)
	}
}

func TestScannerContextCancel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goclean_cancel_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	sc := New(Options{Workers: 4})
	_, _, err = sc.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("unexpected error on empty dir with timeout: %v", err)
	}
}
