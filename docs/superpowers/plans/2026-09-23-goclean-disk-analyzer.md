# GoClean Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a high-performance parallel disk space analyzer (`ncdu` / `dust` clone) in Go with worker-pool scanning, pseudographic usage bars, cache detection, and an interactive TUI.

**Architecture:** A concurrent worker pool traverses directory trees using `os.ReadDir`, tracks tasks with atomic counters to avoid deadlock, aggregates file sizes bottom-up (post-order) into an in-memory node hierarchy, detects build/package caches (`node_modules`, `target`, `.cache`, etc.), and renders either a stylized CLI tree or an interactive full-screen Bubbletea TUI with keyboard navigation and safe deletion.

**Tech Stack:** Go 1.27, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`, standard library (`os`, `sync`, `path/filepath`, `context`, `time`, `testing`).

---

## File Structure

- `go.mod`, `go.sum` — Go module definition and dependencies.
- `internal/model/node.go` — Node representation (`Node`, `NodeType`), tree sorting, post-order aggregation.
- `internal/model/stats.go` — Scan statistics and counters (`ScanStats`).
- `internal/model/node_test.go` — Unit tests for node aggregation and sorting.
- `internal/ui/format.go` — Human-readable byte formatting (B, KB, MB, GB, TB) and item count formatters.
- `internal/ui/bar.go` — Pseudographic usage bar generation with color thresholds.
- `internal/ui/bar_test.go` — Unit tests for progress bar characters and percentages.
- `internal/cleaner/rules.go` — Known cache folder patterns (npm, cargo, python, gradle, etc.).
- `internal/cleaner/detector.go` — Detection functions matching nodes to cache rules.
- `internal/cleaner/detector_test.go` — Unit tests for cache detection.
- `internal/cleaner/cleaner.go` — Safe deletion engine with dry-run support and protected path checks.
- `internal/cleaner/cleaner_test.go` — Unit tests for deletion safety.
- `internal/scanner/symlink.go` — Symlink and junction loop protection.
- `internal/scanner/pool.go` — Concurrent worker pool and task dispatcher.
- `internal/scanner/scanner.go` — Directory traversal orchestrator.
- `internal/scanner/scanner_test.go` — Unit and race tests (`-race`) for concurrent scanning.
- `internal/ui/cli.go` — Dust-style hierarchical tree renderer with bars and cache tags.
- `internal/ui/cli_test.go` — Unit tests for CLI tree output formatting.
- `internal/ui/tui/model.go` — Bubbletea state model for the interactive ncdu-like TUI.
- `internal/ui/tui/update.go` — TUI keyboard and mouse events handling.
- `internal/ui/tui/view.go` — Lipgloss layout rendering for the TUI.
- `internal/ui/tui/dialog.go` — Confirmation popup dialogs for cache and folder deletion.
- `cmd/goclean/main.go` — CLI flag parsing (`-i`, `-w`, `-d`, `--clean-cache`, `--dry-run`, `-y`) and mode switcher.

---

### Task 1: Initialize Go Module & Dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
go mod init goclean
```

- [ ] **Step 2: Add charmbracelet dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@v1.3.4 github.com/charmbracelet/lipgloss@v1.0.0 github.com/charmbracelet/bubbles@v0.20.0
```

- [ ] **Step 3: Verify module compiles**

Run:
```bash
go mod tidy
```
Expected: `go.mod` and `go.sum` generated without errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: initialize go module and charmbracelet dependencies"
```

---

### Task 2: Models & Post-Order Tree Aggregation

**Files:**
- Create: `internal/model/node.go`
- Create: `internal/model/stats.go`
- Test: `internal/model/node_test.go`

- [ ] **Step 1: Write the failing test for Node, aggregation, and sorting**

Create `internal/model/node_test.go`:
```go
package model

import (
	"testing"
)

func TestPostOrderAggregateAndSort(t *testing.T) {
	root := &Node{Name: "root", Path: "/root", Type: NodeTypeDir}
	childA := &Node{Name: "dirA", Path: "/root/dirA", Type: NodeTypeDir}
	childB := &Node{Name: "fileB", Path: "/root/fileB", Type: NodeTypeFile, Size: 500, ItemCount: 1}
	childA1 := &Node{Name: "fileA1", Path: "/root/dirA/fileA1", Type: NodeTypeFile, Size: 1500, ItemCount: 1}

	childA.AddChild(childA1)
	root.AddChild(childA)
	root.AddChild(childB)

	root.PostOrderAggregate()

	if childA.Size != 1500 {
		t.Fatalf("expected childA size 1500, got %d", childA.Size)
	}
	if childA.ItemCount != 1 {
		t.Fatalf("expected childA items 1, got %d", childA.ItemCount)
	}
	if root.Size != 2000 {
		t.Fatalf("expected root size 2000, got %d", root.Size)
	}
	if root.ItemCount != 3 { // fileA1, dirA, fileB
		t.Fatalf("expected root items 3, got %d", root.ItemCount)
	}

	root.SortChildren(SortBySize)
	if root.Children[0].Name != "dirA" {
		t.Fatalf("expected largest child dirA first, got %s", root.Children[0].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/model
```
Expected: FAIL (types and methods undefined).

- [ ] **Step 3: Implement Node and Stats**

Create `internal/model/stats.go`:
```go
package model

import "time"

type ScanStats struct {
	TotalFiles   int64
	TotalDirs    int64
	TotalBytes   int64
	ErrorsCount  int64
	Duration     time.Duration
	CacheBytes   int64
	CacheDirs    int64
}
```

Create `internal/model/node.go`:
```go
package model

import (
	"sort"
	"sync"
	"time"
)

type NodeType uint8

const (
	NodeTypeDir NodeType = iota
	NodeTypeFile
	NodeTypeSymlink
)

type SortCriteria int

const (
	SortBySize SortCriteria = iota
	SortByName
	SortByItems
)

type Node struct {
	Name      string
	Path      string
	Size      int64
	ItemCount int64
	Type      NodeType
	ModTime   time.Time
	IsCache   bool
	CacheKind string
	Parent    *Node `json:"-"`
	Children  []*Node
	mu        sync.Mutex
}

func (n *Node) AddChild(child *Node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	child.Parent = n
	n.Children = append(n.Children, child)
}

func (n *Node) PostOrderAggregate() {
	if n.Type != NodeTypeDir {
		return
	}
	var totalSize int64
	var totalItems int64

	for _, child := range n.Children {
		if child.Type == NodeTypeDir {
			child.PostOrderAggregate()
			totalSize += child.Size
			totalItems += child.ItemCount + 1
		} else {
			totalSize += child.Size
			totalItems += 1
		}
	}
	n.Size = totalSize
	n.ItemCount = totalItems
}

func (n *Node) SortChildren(criteria SortCriteria) {
	for _, child := range n.Children {
		if child.Type == NodeTypeDir {
			child.SortChildren(criteria)
		}
	}

	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		switch criteria {
		case SortBySize:
			if a.Size == b.Size {
				return a.Name < b.Name
			}
			return a.Size > b.Size
		case SortByName:
			return a.Name < b.Name
		case SortByItems:
			if a.ItemCount == b.ItemCount {
				return a.Size > b.Size
			}
			return a.ItemCount > b.ItemCount
		default:
			return a.Size > b.Size
		}
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/model
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model
git commit -m "feat(model): add Node tree with post-order aggregation and sorting"
```

---

### Task 3: Formatting & Pseudographic Usage Bars

**Files:**
- Create: `internal/ui/format.go`
- Create: `internal/ui/bar.go`
- Test: `internal/ui/bar_test.go`

- [ ] **Step 1: Write failing test for formatters and usage bars**

Create `internal/ui/bar_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/ui
```
Expected: FAIL.

- [ ] **Step 3: Implement format.go and bar.go**

Create `internal/ui/format.go`:
```go
package ui

import (
	"fmt"
	"strings"
)

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatNumber(n int64) string {
	in := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(in)+len(in)/3)
	leading := len(in) % 3
	if leading > 0 {
		out = append(out, in[:leading]...)
		if len(in) > 3 {
			out = append(out, ',')
		}
	}
	for i := leading; i < len(in); i += 3 {
		out = append(out, in[i:i+3]...)
		if i+3 < len(in) {
			out = append(out, ',')
		}
	}
	return string(out)
}
```

Create `internal/ui/bar.go`:
```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorGreen  = lipgloss.Color("#00E676")
	colorYellow = lipgloss.Color("#FFD600")
	colorOrange = lipgloss.Color("#FF9100")
	colorRed    = lipgloss.Color("#FF5252")
)

func RenderBar(ratio float64, width int, colored bool) string {
	if width <= 0 {
		return ""
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	totalSteps := float64(width) * ratio
	fullBlocks := int(totalSteps)
	remainder := totalSteps - float64(fullBlocks)

	var sb strings.Builder
	sb.WriteString(strings.Repeat("█", fullBlocks))

	if fullBlocks < width {
		if remainder >= 0.5 {
			sb.WriteString("▌")
			fullBlocks++
		}
		if width-fullBlocks > 0 {
			sb.WriteString(strings.Repeat("░", width-fullBlocks))
		}
	}

	barStr := sb.String()
	if !colored {
		return barStr
	}

	var style lipgloss.Style
	switch {
	case ratio >= 0.50:
		style = lipgloss.NewStyle().Foreground(colorRed)
	case ratio >= 0.25:
		style = lipgloss.NewStyle().Foreground(colorOrange)
	case ratio >= 0.10:
		style = lipgloss.NewStyle().Foreground(colorYellow)
	default:
		style = lipgloss.NewStyle().Foreground(colorGreen)
	}

	return style.Render(barStr)
}

func FormatPercent(ratio float64) string {
	pct := ratio * 100
	return fmt.Sprintf("%5.1f%%", pct)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/ui
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui
git commit -m "feat(ui): add byte formatting and pseudographic usage bar renderer"
```

---

### Task 4: Cache Detector & Matching Rules

**Files:**
- Create: `internal/cleaner/rules.go`
- Create: `internal/cleaner/detector.go`
- Test: `internal/cleaner/detector_test.go`

- [ ] **Step 1: Write failing test for cache detection**

Create `internal/cleaner/detector_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/cleaner
```
Expected: FAIL.

- [ ] **Step 3: Implement rules.go and detector.go**

Create `internal/cleaner/rules.go`:
```go
package cleaner

import "strings"

type CacheRule struct {
	Name string
	Kind string
}

var DefaultCacheRules = map[string]string{
	"node_modules":    "npm/node",
	".next":           "nextjs",
	".nuxt":           "nuxtjs",
	".turbo":          "turborepo",
	".pnpm-store":     "pnpm",
	"target":          "rust/cargo",
	"__pycache__":     "python",
	".pytest_cache":   "pytest",
	".mypy_cache":     "mypy",
	".venv":           "python-venv",
	"venv":            "python-venv",
	".gradle":         "gradle",
	"build":           "build-dir",
	"dist":            "dist-dir",
	".cache":          "general-cache",
	"bin":             "bin-cache",
	"obj":             "dotnet-obj",
	".DS_Store":       "system",
	"Thumbs.db":       "system",
}

func MatchCacheRule(dirName string) (bool, string) {
	lower := strings.ToLower(dirName)
	if kind, ok := DefaultCacheRules[lower]; ok {
		return true, kind
	}
	return false, ""
}
```

Create `internal/cleaner/detector.go`:
```go
package cleaner

import (
	"goclean/internal/model"
)

func DetectCache(dirName string) (bool, string) {
	return MatchCacheRule(dirName)
}

func TagCaches(root *model.Node) (totalCacheBytes int64, totalCacheDirs int64) {
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type == model.NodeTypeDir {
			if isCache, kind := DetectCache(n.Name); isCache {
				n.IsCache = true
				n.CacheKind = kind
				totalCacheBytes += n.Size
				totalCacheDirs++
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
	}
	walk(root)
	return totalCacheBytes, totalCacheDirs
}

func CollectCaches(root *model.Node) []*model.Node {
	var caches []*model.Node
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type == model.NodeTypeDir {
			if n.IsCache {
				caches = append(caches, n)
				// Don't recurse into subdirectories of a cache directory (e.g. node_modules/foo/node_modules)
				return
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
	}
	walk(root)
	return caches
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/cleaner
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cleaner
git commit -m "feat(cleaner): add cache detection rules and collector"
```

---

### Task 5: Cache Cleaner with Dry-Run & Safety Guards

**Files:**
- Create: `internal/cleaner/cleaner.go`
- Test: `internal/cleaner/cleaner_test.go`

- [ ] **Step 1: Write failing test for safe deletion and root protection**

Create `internal/cleaner/cleaner_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/cleaner
```
Expected: FAIL.

- [ ] **Step 3: Implement cleaner.go**

Create `internal/cleaner/cleaner.go`:
```go
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
		if strings.EqualFold(upper, filepath.Clean(p)) {
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/cleaner
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cleaner
git commit -m "feat(cleaner): add safe deletion engine and protected paths validator"
```

---

### Task 6: Concurrent Scanner Engine with Worker Pool

**Files:**
- Create: `internal/scanner/symlink.go`
- Create: `internal/scanner/pool.go`
- Create: `internal/scanner/scanner.go`
- Test: `internal/scanner/scanner_test.go`

- [ ] **Step 1: Write failing test for concurrent scanner**

Create `internal/scanner/scanner_test.go`:
```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/scanner
```
Expected: FAIL.

- [ ] **Step 3: Implement symlink protection, pool, and scanner**

Create `internal/scanner/symlink.go`:
```go
package scanner

import (
	"os"
	"path/filepath"
	"sync"
)

type VisitedMap struct {
	visited sync.Map
}

func (v *VisitedMap) MarkIfNew(path string) bool {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = filepath.Clean(path)
	}
	_, loaded := v.visited.LoadOrStore(realPath, struct{}{})
	return !loaded
}

func IsSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}
```

Create `internal/scanner/pool.go`:
```go
package scanner

import (
	"context"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"goclean/internal/model"
)

type Options struct {
	Workers int
}

type dirTask struct {
	path string
	node *model.Node
}

type Scanner struct {
	workers int
	visited VisitedMap
}

func New(opts Options) *Scanner {
	w := opts.Workers
	if w <= 0 {
		w = runtime.NumCPU() * 2
	}
	return &Scanner{
		workers: w,
	}
}
```

Create `internal/scanner/scanner.go`:
```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"goclean/internal/model"
)

func (s *Scanner) Scan(ctx context.Context, rootPath string) (*model.Node, *model.ScanStats, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		absRoot = rootPath
	}

	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return nil, nil, err
	}

	startTime := time.Now()
	rootNode := &model.Node{
		Name:    filepath.Base(absRoot),
		Path:    absRoot,
		Type:    model.NodeTypeDir,
		ModTime: rootInfo.ModTime(),
	}

	if !rootInfo.IsDir() {
		rootNode.Type = model.NodeTypeFile
		rootNode.Size = rootInfo.Size()
		rootNode.ItemCount = 1
		stats := &model.ScanStats{
			TotalFiles: 1,
			TotalBytes: rootNode.Size,
			Duration:   time.Since(startTime),
		}
		return rootNode, stats, nil
	}

	s.visited.MarkIfNew(absRoot)

	taskQueue := make(chan dirTask, 10000)
	var inFlight atomic.Int64
	var totalFiles atomic.Int64
	var totalDirs atomic.Int64
	var totalErrors atomic.Int64

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-taskQueue:
					if !ok {
						return
					}
					s.processDir(ctx, task, taskQueue, &inFlight, &totalFiles, &totalDirs, &totalErrors)
					if inFlight.Add(-1) == 0 {
						close(taskQueue)
						return
					}
				}
			}
		}()
	}

	inFlight.Add(1)
	taskQueue <- dirTask{path: absRoot, node: rootNode}

	wg.Wait()

	rootNode.PostOrderAggregate()

	stats := &model.ScanStats{
		TotalFiles:  totalFiles.Load(),
		TotalDirs:   totalDirs.Load(),
		TotalBytes:  rootNode.Size,
		ErrorsCount: totalErrors.Load(),
		Duration:    time.Since(startTime),
	}

	return rootNode, stats, nil
}

func (s *Scanner) processDir(
	ctx context.Context,
	task dirTask,
	taskQueue chan<- dirTask,
	inFlight *atomic.Int64,
	totalFiles *atomic.Int64,
	totalDirs *atomic.Int64,
	totalErrors *atomic.Int64,
) {
	entries, err := os.ReadDir(task.path)
	if err != nil {
		totalErrors.Add(1)
		return
	}

	totalDirs.Add(1)

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		childPath := filepath.Join(task.path, entry.Name())

		if IsSymlink(entry) {
			continue
		}

		if entry.IsDir() {
			if !s.visited.MarkIfNew(childPath) {
				continue
			}

			childNode := &model.Node{
				Name:    entry.Name(),
				Path:    childPath,
				Type:    model.NodeTypeDir,
				Parent:  task.node,
			}
			task.node.AddChild(childNode)

			inFlight.Add(1)
			taskQueue <- dirTask{path: childPath, node: childNode}
		} else {
			info, err := entry.Info()
			var size int64
			var modTime time.Time
			if err != nil {
				totalErrors.Add(1)
			} else {
				size = info.Size()
				modTime = info.ModTime()
			}

			childNode := &model.Node{
				Name:      entry.Name(),
				Path:      childPath,
				Size:      size,
				Type:      model.NodeTypeFile,
				ModTime:   modTime,
				ItemCount: 1,
				Parent:    task.node,
			}
			task.node.AddChild(childNode)
			totalFiles.Add(1)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes with race detector**

Run:
```bash
go test -v -race ./internal/scanner
```
Expected: PASS with 0 data races.

- [ ] **Step 5: Commit**

```bash
git add internal/scanner
git commit -m "feat(scanner): add parallel worker pool scanner with race-safe queue"
```

---

### Task 7: CLI Tree Renderer with Usage Bars & Cache Tags

**Files:**
- Create: `internal/ui/cli.go`
- Test: `internal/ui/cli_test.go`

- [ ] **Step 1: Write test for CLI tree renderer**

Create `internal/ui/cli_test.go`:
```go
package ui

import (
	"strings"
	"testing"

	"goclean/internal/model"
)

func TestRenderCLITree(t *testing.T) {
	root := &model.Node{
		Name: "project",
		Path: "/project",
		Type: model.NodeTypeDir,
	}
	cacheNode := &model.Node{
		Name:      "node_modules",
		Path:      "/project/node_modules",
		Type:      model.NodeTypeDir,
		Size:      1000,
		IsCache:   true,
		CacheKind: "npm/node",
	}
	srcNode := &model.Node{
		Name: "src",
		Path: "/project/src",
		Type: model.NodeTypeDir,
		Size: 500,
	}
	root.AddChild(cacheNode)
	root.AddChild(srcNode)
	root.PostOrderAggregate()

	out := RenderCLITree(root, 2, 20)
	if !strings.Contains(out, "node_modules") {
		t.Errorf("expected tree to contain node_modules, got %s", out)
	}
	if !strings.Contains(out, "[CACHE: npm/node]") {
		t.Errorf("expected tree to highlight cache tag, got %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/ui
```
Expected: FAIL.

- [ ] **Step 3: Implement cli.go**

Create `internal/ui/cli.go`:
```go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EEEEEE")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#555555"))

	cacheTagStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			Background(lipgloss.Color("#331111")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

func RenderCLITree(root *model.Node, maxDepth int, barWidth int) string {
	if barWidth <= 0 {
		barWidth = 15
	}
	root.SortChildren(model.SortBySize)

	var sb strings.Builder

	header := fmt.Sprintf("%-10s  %-*s  %6s  %s", "Размер", barWidth+2, "% Использования", "Файлы", "Путь")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	var printNode func(n *model.Node, depth int, prefix string, isLast bool)
	printNode = func(n *model.Node, depth int, prefix string, isLast bool) {
		if depth > maxDepth {
			return
		}

		var branch string
		if depth > 0 {
			if isLast {
				branch = prefix + "└── "
			} else {
				branch = prefix + "├── "
			}
		}

		ratio := float64(0)
		if root.Size > 0 {
			ratio = float64(n.Size) / float64(root.Size)
		}

		bar := RenderBar(ratio, barWidth, true)
		pctStr := FormatPercent(ratio)
		sizeStr := FormatBytes(n.Size)
		itemsStr := FormatNumber(n.ItemCount)

		var icon string
		if n.Type == model.NodeTypeDir {
			icon = "📁 "
		} else {
			icon = "📄 "
		}

		line := fmt.Sprintf("%-10s  [%s] %s  %6s  %s%s%s",
			sizeStr,
			bar,
			pctStr,
			itemsStr,
			branch,
			icon,
			n.Name,
		)

		if n.IsCache {
			line += " " + cacheTagStyle.Render(fmt.Sprintf("[CACHE: %s]", n.CacheKind))
		}

		sb.WriteString(line)
		sb.WriteString("\n")

		if n.Type == model.NodeTypeDir && depth < maxDepth {
			var newPrefix string
			if depth > 0 {
				if isLast {
					newPrefix = prefix + "    "
				} else {
					newPrefix = prefix + "│   "
				}
			}
			for i, child := range n.Children {
				printNode(child, depth+1, newPrefix, i == len(n.Children)-1)
			}
		}
	}

	printNode(root, 0, "", true)
	return sb.String()
}

func RenderSummary(stats *model.ScanStats) string {
	sep := dimStyle.Render(" | ")
	summary := fmt.Sprintf("⚡ Время сканирования: %v%s📁 Директорий: %s%s📄 Файлов: %s%s💾 Всего: %s",
		stats.Duration.Round(10*time.Millisecond),
		sep,
		FormatNumber(stats.TotalDirs),
		sep,
		FormatNumber(stats.TotalFiles),
		sep,
		FormatBytes(stats.TotalBytes),
	)

	if stats.CacheDirs > 0 {
		summary += fmt.Sprintf("\n💡 Найдено кэшей к очистке: %s (%s)",
			FormatBytes(stats.CacheBytes),
			FormatNumber(stats.CacheDirs)+" папок",
		)
	}

	return summary
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/ui
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui
git commit -m "feat(ui): add CLI tree renderer with usage bars and summary"
```

---

### Task 8: Interactive Full-Screen TUI (Bubbletea + Lipgloss)

**Files:**
- Create: `internal/ui/tui/model.go`
- Create: `internal/ui/tui/dialog.go`
- Create: `internal/ui/tui/update.go`
- Create: `internal/ui/tui/view.go`

- [ ] **Step 1: Implement TUI dialogs and confirmation states**

Create `internal/ui/tui/dialog.go`:
```go
package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
	"goclean/internal/ui"
)

type DialogKind int

const (
	DialogNone DialogKind = iota
	DialogDeleteTarget
	DialogCleanAllCaches
)

var (
	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5252")).
			Padding(1, 2).
			Width(65)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			MarginBottom(1)
)

func renderDeleteDialog(node *model.Node) string {
	content := fmt.Sprintf("%s\n\nВы действительно хотите безвозвратно удалить:\n📁 %s\nРазмер: %s (%s файлов)?\n\n[Enter / y] Подтвердить   [Esc / n] Отмена",
		modalTitleStyle.Render("⚠️  ПОДТВЕРЖДЕНИЕ УДАЛЕНИЯ"),
		node.Path,
		ui.FormatBytes(node.Size),
		ui.FormatNumber(node.ItemCount),
	)
	return modalBoxStyle.Render(content)
}

func renderCleanCachesDialog(caches []*model.Node, totalBytes int64) string {
	content := fmt.Sprintf("%s\n\nНайдено кэшей для очистки: %d папок\nБудет освобождено: %s\n\nУдалить все обнаруженные кэши безвозвратно?\n\n[Enter / y] Очистить   [Esc / n] Отмена",
		modalTitleStyle.Render("🧹 ОЧИСТКА ВСЕХ КЭШЕЙ"),
		len(caches),
		ui.FormatBytes(totalBytes),
	)
	return modalBoxStyle.Render(content)
}
```

- [ ] **Step 2: Implement TUI Model, Update, and View**

Create `internal/ui/tui/model.go`:
```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
)

type Model struct {
	Root         *model.Node
	Current      *model.Node
	Cursor       int
	Sort         model.SortCriteria
	Filter       string
	Filtering    bool
	Width        int
	Height       int
	Dialog       DialogKind
	StatusMsg    string
	TargetCaches []*model.Node
	CacheBytes   int64
}

func NewModel(root *model.Node) Model {
	root.SortChildren(model.SortBySize)
	cacheBytes, _ := cleaner.TagCaches(root)
	return Model{
		Root:       root,
		Current:    root,
		Cursor:     0,
		Sort:       model.SortBySize,
		CacheBytes: cacheBytes,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
```

Create `internal/ui/tui/update.go`:
```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
	"goclean/internal/ui"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.Dialog != DialogNone {
			return m.handleDialogKeys(msg)
		}

		if m.Filtering {
			return m.handleFilterKeys(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			items := m.filteredChildren()
			if m.Cursor < len(items)-1 {
				m.Cursor++
			}

		case "enter", "right", "l":
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				selected := items[m.Cursor]
				if selected.Type == model.NodeTypeDir && len(selected.Children) > 0 {
					m.Current = selected
					m.Cursor = 0
					m.Filter = ""
				}
			}

		case "esc", "backspace", "left", "h":
			if m.Current.Parent != nil {
				m.Current = m.Current.Parent
				m.Cursor = 0
				m.Filter = ""
			}

		case "s":
			switch m.Sort {
			case model.SortBySize:
				m.Sort = model.SortByName
				m.StatusMsg = "Сортировка: по имени"
			case model.SortByName:
				m.Sort = model.SortByItems
				m.StatusMsg = "Сортировка: по числу файлов"
			case model.SortByItems:
				m.Sort = model.SortBySize
				m.StatusMsg = "Сортировка: по размеру"
			}
			m.Current.SortChildren(m.Sort)

		case "/":
			m.Filtering = true
			m.Filter = ""

		case "d":
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				m.Dialog = DialogDeleteTarget
			}

		case "c":
			caches := cleaner.CollectCaches(m.Current)
			if len(caches) == 0 {
				m.StatusMsg = "В текущей папке кэшей не обнаружено"
			} else {
				var total int64
				for _, c := range caches {
					total += c.Size
				}
				m.TargetCaches = caches
				m.CacheBytes = total
				m.Dialog = DialogCleanAllCaches
			}
		}
	}
	return m, nil
}

func (m Model) handleDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "N":
		m.Dialog = DialogNone
		return m, nil

	case "enter", "y", "Y":
		if m.Dialog == DialogDeleteTarget {
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				target := items[m.Cursor]
				err := cleaner.DeleteSafely(target.Path, false)
				if err != nil {
					m.StatusMsg = fmt.Sprintf("Ошибка удаления: %v", err)
				} else {
					m.StatusMsg = fmt.Sprintf("Удалено: %s", target.Name)
					m.removeNodeFromParent(target)
				}
			}
		} else if m.Dialog == DialogCleanAllCaches {
			deletedCount := 0
			var reclaimed int64
			for _, c := range m.TargetCaches {
				if err := cleaner.DeleteSafely(c.Path, false); err == nil {
					deletedCount++
					reclaimed += c.Size
					m.removeNodeFromParent(c)
				}
			}
			m.StatusMsg = fmt.Sprintf("Очищено %d кэшей, освобождено %s", deletedCount, ui.FormatBytes(reclaimed))
		}
		m.Dialog = DialogNone
		m.Root.PostOrderAggregate()
		return m, nil
	}
	return m, nil
}

func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Filtering = false
	case "backspace":
		if len(m.Filter) > 0 {
			m.Filter = m.Filter[:len(m.Filter)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.Filter += msg.String()
			m.Cursor = 0
		}
	}
	return m, nil
}

func (m Model) filteredChildren() []*model.Node {
	if m.Filter == "" {
		return m.Current.Children
	}
	var res []*model.Node
	for _, child := range m.Current.Children {
		if strings.Contains(strings.ToLower(child.Name), strings.ToLower(m.Filter)) {
			res = append(res, child)
		}
	}
	return res
}

func (m *Model) removeNodeFromParent(target *model.Node) {
	if target.Parent == nil {
		return
	}
	p := target.Parent
	for i, c := range p.Children {
		if c == target {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	if m.Cursor >= len(p.Children) && m.Cursor > 0 {
		m.Cursor--
	}
}
```

Create `internal/ui/tui/view.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
	"goclean/internal/ui"
)

var (
	headerBoxStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#005F87"))

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	footerBoxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#111111")).
			Padding(0, 1)
)

func (m Model) View() string {
	if m.Width == 0 {
		return "Загрузка..."
	}

	var sb strings.Builder

	header := fmt.Sprintf(" GoClean v1.0 ─ Каталог: %s ", m.Current.Path)
	sb.WriteString(headerBoxStyle.Width(m.Width).Render(header))
	sb.WriteString("\n")

	tableHeader := fmt.Sprintf("  %-30s  %10s  %-15s  %6s  %s",
		"Имя", "Размер", "% Использования", "Файлы", "Метка")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(tableHeader))
	sb.WriteString("\n")

	items := m.filteredChildren()
	maxRows := m.Height - 6
	if maxRows < 5 {
		maxRows = 10
	}

	start := 0
	if m.Cursor >= maxRows {
		start = m.Cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	parentSize := m.Current.Size
	if parentSize == 0 {
		parentSize = 1
	}

	for i := start; i < end; i++ {
		item := items[i]
		ratio := float64(item.Size) / float64(parentSize)
		bar := ui.RenderBar(ratio, 12, true)
		pct := ui.FormatPercent(ratio)

		icon := "📁 "
		if item.Type == model.NodeTypeFile {
			icon = "📄 "
		}

		name := item.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}

		tag := ""
		if item.IsCache {
			tag = fmt.Sprintf("[КЭШ: %s]", item.CacheKind)
		}

		row := fmt.Sprintf(" %s%-28s  %10s  [%s] %s  %6s  %s",
			icon,
			name,
			ui.FormatBytes(item.Size),
			bar,
			pct,
			ui.FormatNumber(item.ItemCount),
			tag,
		)

		if i == m.Cursor {
			sb.WriteString(selectedRowStyle.Width(m.Width).Render("> " + row))
		} else {
			sb.WriteString(normalRowStyle.Width(m.Width).Render("  " + row))
		}
		sb.WriteString("\n")
	}

	// Status / Helper bar
	if m.StatusMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD600")).Render("⚡ " + m.StatusMsg))
		sb.WriteString("\n")
	}

	help := "[↑/↓/j/k] Перемещение | [Enter/→] Войти | [Esc/←] Назад | [s] Сортировка | [c] Очистить кэши | [d] Удалить | [q] Выход"
	if m.Filtering {
		help = fmt.Sprintf("🔍 Поиск: %s_ [Esc - закрыть]", m.Filter)
	}
	sb.WriteString(footerBoxStyle.Width(m.Width).Render(help))

	if m.Dialog == DialogDeleteTarget && len(items) > 0 {
		return renderDeleteDialog(items[m.Cursor])
	}
	if m.Dialog == DialogCleanAllCaches {
		return renderCleanCachesDialog(m.TargetCaches, m.CacheBytes)
	}

	return sb.String()
}
```

- [ ] **Step 3: Verify TUI package compiles**

Run:
```bash
go build ./internal/ui/tui
```
Expected: successful build without errors.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/tui
git commit -m "feat(tui): add interactive full-screen bubbletea model and navigation"
```

---

### Task 9: CLI Entry Point & Flag Orchestration

**Files:**
- Create: `cmd/goclean/main.go`

- [ ] **Step 1: Implement main.go with flags and modes**

Create `cmd/goclean/main.go`:
```go
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/scanner"
	"goclean/internal/ui"
	"goclean/internal/ui/tui"
)

func main() {
	interactive := flag.Bool("i", false, "Запустить интерактивный TUI режим (как ncdu)")
	workers := flag.Int("w", 0, "Количество параллельных воркеров (по умолчанию NumCPU * 2)")
	maxDepth := flag.Int("d", 2, "Глубина отображения дерева каталогов")
	cleanCache := flag.Bool("clean-cache", false, "Найти и удалить все кэши (node_modules, target, etc.)")
	dryRun := flag.Bool("dry-run", false, "Сухой прогон (не удалять файлы на диске)")
	yes := flag.Bool("y", false, "Автоматическое подтверждение удаления без запроса")
	flag.Parse()

	targetPath := "."
	if flag.NArg() > 0 {
		targetPath = flag.Arg(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("🔍 Сканирование %s (воркеры: %d)...\n", targetPath, *workers)

	sc := scanner.New(scanner.Options{
		Workers: *workers,
	})

	root, stats, err := sc.Scan(ctx, targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сканирования: %v\n", err)
		os.Exit(1)
	}

	cacheBytes, cacheDirs := cleaner.TagCaches(root)
	stats.CacheBytes = cacheBytes
	stats.CacheDirs = cacheDirs

	if *interactive {
		p := tea.NewProgram(tui.NewModel(root), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка TUI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *cleanCache {
		caches := cleaner.CollectCaches(root)
		if len(caches) == 0 {
			fmt.Println("✨ Директорий кэша не обнаружено!")
			return
		}

		fmt.Printf("\n🧹 Найдено %d директорий кэша суммарным объемом %s:\n", len(caches), ui.FormatBytes(cacheBytes))
		for _, c := range caches {
			fmt.Printf("  • %-30s [%s] (%s, %s файлов)\n", c.Path, c.CacheKind, ui.FormatBytes(c.Size), ui.FormatNumber(c.ItemCount))
		}

		if *dryRun {
			fmt.Printf("\n[DRY-RUN] Файлы не удалены. Запустите без --dry-run для очистки.\n")
			return
		}

		if !*yes {
			fmt.Printf("\nУдалить эти директории безвозвратно? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("Отмена очистки.")
				return
			}
		}

		var reclaimed int64
		for _, c := range caches {
			if err := cleaner.DeleteSafely(c.Path, false); err != nil {
				fmt.Fprintf(os.Stderr, "Ошибка удаления %s: %v\n", c.Path, err)
			} else {
				reclaimed += c.Size
			}
		}
		fmt.Printf("✅ Успешно очищено: %s!\n", ui.FormatBytes(reclaimed))
		return
	}

	// Default CLI output mode (dust style)
	treeOutput := ui.RenderCLITree(root, *maxDepth, 20)
	fmt.Print(treeOutput)
	fmt.Println(ui.RenderSummary(stats))
}
```

- [ ] **Step 2: Build the complete binary**

Run:
```bash
go build -o goclean.exe ./cmd/goclean
```
Expected: `goclean.exe` built successfully.

- [ ] **Step 3: Test execution on current workspace**

Run:
```bash
./goclean.exe -d 1 .
```
Expected: Prints tree with percentage bars and summary.

- [ ] **Step 4: Commit**

```bash
git add cmd/goclean
git commit -m "feat(cli): add CLI entry point with flags for scan, TUI, and cache cleaner"
```

---

### Task 10: End-to-End Verification & Documentation

**Files:**
- Create: `README.md`
- Create: `tests/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `tests/integration_test.go`:
```go
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
}
```

- [ ] **Step 2: Run all tests with race detector**

Run:
```bash
go test -v -race ./...
```
Expected: All tests PASS.

- [ ] **Step 3: Create README.md**

Create `README.md` documenting usage, shortcuts, flags, and architecture.

- [ ] **Step 4: Commit**

```bash
git add README.md tests/
git commit -m "docs: add README and integration test suite"
```
