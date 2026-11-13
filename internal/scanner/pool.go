package scanner

import (
	"runtime"
)

type ProgressFunc func(dirs, files, errors int64)

type Options struct {
	Workers    int
	OnProgress ProgressFunc
}

type Scanner struct {
	workers    int
	onProgress ProgressFunc
}

func New(opts Options) *Scanner {
	w := opts.Workers
	if w <= 0 {
		w = runtime.NumCPU() * 2
	}
	return &Scanner{
		workers:    w,
		onProgress: opts.OnProgress,
	}
}

func (s *Scanner) Workers() int {
	return s.workers
}
