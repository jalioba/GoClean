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
