package model

import (
	"container/heap"
	"path/filepath"
	"sort"
	"strings"
)

type ExtStat struct {
	Ext   string
	Size  int64
	Count int64
}

type fileMinHeap []*Node

func (h fileMinHeap) Len() int { return len(h) }
func (h fileMinHeap) Less(i, j int) bool {
	if h[i].Size == h[j].Size {
		return h[i].Name > h[j].Name
	}
	return h[i].Size < h[j].Size
}
func (h fileMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *fileMinHeap) Push(x any)   { *h = append(*h, x.(*Node)) }
func (h *fileMinHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return item
}

// TopFiles returns the top limit largest files in the subtree rooted at n,
// sorted by Size descending, using an O(N log K) min-heap.
func (n *Node) TopFiles(limit int) []*Node {
	if n == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	h := make(fileMinHeap, 0, limit+1)
	heap.Init(&h)

	var walk func(curr *Node)
	walk = func(curr *Node) {
		if curr.Type == NodeTypeFile {
			if len(h) < limit {
				heap.Push(&h, curr)
			} else if curr.Size > h[0].Size {
				h[0] = curr
				heap.Fix(&h, 0)
			}
			return
		}
		for _, ch := range curr.Children {
			walk(ch)
		}
	}
	walk(n)

	res := make([]*Node, len(h))
	for i := len(h) - 1; i >= 0; i-- {
		res[i] = heap.Pop(&h).(*Node)
	}
	return res
}

// ExtensionStats aggregates file size and count by file extension across the subtree rooted at n,
// returning up to limit entries (0 = all) sorted by Size descending.
func (n *Node) ExtensionStats(limit int) []ExtStat {
	if n == nil {
		return nil
	}

	byExt := make(map[string]*ExtStat, 64)

	var walk func(curr *Node)
	walk = func(curr *Node) {
		if curr.Type == NodeTypeFile {
			ext := strings.ToLower(filepath.Ext(curr.Name))
			if ext == "" {
				ext = "[no ext]"
			}
			st, ok := byExt[ext]
			if !ok {
				st = &ExtStat{Ext: ext}
				byExt[ext] = st
			}
			st.Size += curr.Size
			st.Count++
			return
		}
		for _, ch := range curr.Children {
			walk(ch)
		}
	}
	walk(n)

	stats := make([]ExtStat, 0, len(byExt))
	for _, st := range byExt {
		stats = append(stats, *st)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Size == stats[j].Size {
			if stats[i].Count == stats[j].Count {
				return stats[i].Ext < stats[j].Ext
			}
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Size > stats[j].Size
	})

	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}
	return stats
}
