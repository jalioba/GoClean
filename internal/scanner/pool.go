package scanner

import (
	"runtime"
)

type Options struct {
	Workers int
}

type Scanner struct {
	workers int
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
