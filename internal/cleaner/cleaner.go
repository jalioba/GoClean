package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var protectedPaths = []string{
	"/", "/etc", "/usr", "/bin", "/sbin", "/var", "/root", "/home",
	"C:\\", "C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)",
	"C:\\Users", "D:\\",
}

func IsProtectedPath(path string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	upper := strings.ToUpper(clean)

	for _, p := range protectedPaths {
		if strings.EqualFold(upper, filepath.Clean(strings.ToUpper(p))) {
			return true
		}
	}

	// Check if path is root of any drive (e.g. "C:\", "D:\", "/")
	vol := filepath.VolumeName(clean)
	if vol != "" && (clean == vol+"\\" || clean == vol+"/") {
		return true
	}
	if clean == "/" || clean == "\\" {
		return true
	}

	// Protect current user home directory directly
	if home, err := os.UserHomeDir(); err == nil {
		if strings.EqualFold(upper, filepath.Clean(strings.ToUpper(home))) {
			return true
		}
	}

	return false
}

func DeleteSafely(path string, dryRun bool) error {
	if IsProtectedPath(path) {
		return fmt.Errorf("refusing to delete protected system path: %s", path)
	}

	if dryRun {
		return nil
	}

	return os.RemoveAll(path)
}
