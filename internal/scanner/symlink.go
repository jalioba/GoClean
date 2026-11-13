package scanner

import "os"

func IsSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}
