//go:build !unix

package scanner

import "os"

// VisitedMap is a lightweight no-op on non-Unix platforms where symlinks
// and junctions are already skipped via IsSymlink.
type VisitedMap struct{}

func (v *VisitedMap) MarkRoot(info os.FileInfo) {}

func (v *VisitedMap) MarkDirEntry(entry os.DirEntry) bool {
	return true
}

