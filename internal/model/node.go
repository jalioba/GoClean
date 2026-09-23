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
