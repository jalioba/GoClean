package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"goclean/internal/model"
)

var protectedPaths = []string{
	"/", "/etc", "/usr", "/bin", "/sbin", "/var", "/root", "/home",
	"C:\\", "C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)",
	"C:\\Users", "D:\\",
}

var (
	homeOnce   sync.Once
	cachedHome string
)

func getUserHomeDir() string {
	homeOnce.Do(func() {
		if h, err := os.UserHomeDir(); err == nil {
			cachedHome = filepath.Clean(strings.ToUpper(h))
		}
	})
	return cachedHome
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
	if home := getUserHomeDir(); home != "" && strings.EqualFold(upper, home) {
		return true
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

type DeleteResult struct {
	Node *model.Node
	Err  error
}

// DeleteNodesParallel deletes the given nodes concurrently using a bounded worker pool.
func DeleteNodesParallel(nodes []*model.Node, workers int, dryRun bool) (reclaimed int64, deleted []*model.Node, errs []error) {
	if len(nodes) == 0 {
		return 0, nil, nil
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 8 {
			workers = 8
		}
	}
	if workers > len(nodes) {
		workers = len(nodes)
	}

	jobs := make(chan *model.Node, len(nodes))
	results := make(chan DeleteResult, len(nodes))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range jobs {
				err := DeleteSafely(n.FullPath(), dryRun)
				results <- DeleteResult{Node: n, Err: err}
			}
		}()
	}

	for _, n := range nodes {
		jobs <- n
	}
	close(jobs)

	wg.Wait()
	close(results)

	for res := range results {
		if res.Err != nil {
			errs = append(errs, res.Err)
		} else {
			reclaimed += res.Node.Size
			deleted = append(deleted, res.Node)
		}
	}
	return reclaimed, deleted, errs
}
