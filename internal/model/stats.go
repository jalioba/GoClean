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
