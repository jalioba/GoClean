package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedPathProtection(t *testing.T) {
	dangerousPaths := []string{
		"/",
		"C:\\",
		"C:\\Windows",
		"C:\\Program Files",
		"/usr",
		"/bin",
	}

	for _, p := range dangerousPaths {
		if !IsProtectedPath(p) {
			t.Errorf("path %s should be protected!", p)
		}
	}
}

func TestDeleteDirSafely(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goclean_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testSubdir := filepath.Join(tempDir, "node_modules")
	if err := os.Mkdir(testSubdir, 0755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(testSubdir, "package.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Dry run
	err = DeleteSafely(testSubdir, true)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if _, err := os.Stat(testSubdir); os.IsNotExist(err) {
		t.Fatalf("dry run should not delete directory")
	}

	// Real delete
	err = DeleteSafely(testSubdir, false)
	if err != nil {
		t.Fatalf("real delete failed: %v", err)
	}
	if _, err := os.Stat(testSubdir); !os.IsNotExist(err) {
		t.Fatalf("real delete did not delete directory")
	}
}
