//go:build unix

package scanner

import (
	"os"
	"sync"
	"syscall"
)

type devIno struct {
	dev uint64
	ino uint64
}

// VisitedMap tracks visited directories by (Dev, Ino) to prevent bind-mount
// and directory-hardlink loops without expensive filepath.EvalSymlinks calls.
type VisitedMap struct {
	visited sync.Map
}

func (v *VisitedMap) MarkRoot(info os.FileInfo) {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		v.visited.Store(devIno{dev: uint64(st.Dev), ino: uint64(st.Ino)}, struct{}{})
	}
}

func (v *VisitedMap) MarkDirEntry(entry os.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		// Let os.ReadDir attempt to open it and record the error if unreadable.
		return true
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	key := devIno{dev: uint64(st.Dev), ino: uint64(st.Ino)}
	_, loaded := v.visited.LoadOrStore(key, struct{}{})
	return !loaded
}

