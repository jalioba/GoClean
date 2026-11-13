package model

import (
	"os"
	"sort"
	"strings"
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

// Node represents a file or directory in the scanned tree.
// Path is only stored on the root node; all descendants compute their path via FullPath()
// to minimize memory footprint and GC pressure on large directory trees.
type Node struct {
	Name      string
	Path      string
	Size      int64
	ItemCount int64
	Type      NodeType
	IsCache   bool
	CacheKind string
	Parent    *Node `json:"-"`
	Children  []*Node
}

// FullPath reconstructs the full filesystem path of the node by walking up Parent pointers.
func (n *Node) FullPath() string {
	if n == nil {
		return ""
	}
	if n.Path != "" {
		return n.Path
	}

	// Collect path segments from n up to the ancestor that has Path set (or root).
	var segments []string
	curr := n
	for curr != nil && curr.Path == "" {
		segments = append(segments, curr.Name)
		curr = curr.Parent
	}

	var base string
	if curr != nil {
		base = curr.Path
	}

	var sb strings.Builder
	totalLen := len(base)
	for _, s := range segments {
		totalLen += 1 + len(s)
	}
	sb.Grow(totalLen)

	needSep := false
	if base != "" {
		sb.WriteString(base)
		last := base[len(base)-1]
		needSep = last != os.PathSeparator && last != '/'
	}
	for i := len(segments) - 1; i >= 0; i-- {
		if needSep {
			sb.WriteByte(os.PathSeparator)
		}
		sb.WriteString(segments[i])
		needSep = true
	}
	return sb.String()
}

func (n *Node) AddChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// RemoveChild detaches target from n.Children and subtracts its size and item count
// from n and all ancestors in O(depth) time.
func (n *Node) RemoveChild(target *Node) bool {
	idx := -1
	for i, c := range n.Children {
		if c == target {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}

	n.Children = append(n.Children[:idx], n.Children[idx+1:]...)

	removedItems := target.ItemCount
	if target.Type == NodeTypeDir {
		removedItems++
	}
	for curr := n; curr != nil; curr = curr.Parent {
		curr.Size -= target.Size
		if curr.Size < 0 {
			curr.Size = 0
		}
		curr.ItemCount -= removedItems
		if curr.ItemCount < 0 {
			curr.ItemCount = 0
		}
	}
	return true
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
	n.SortChildrenDepth(criteria, -1)
}

// SortChildrenDepth sorts children up to maxDepth levels (maxDepth < 0 means unlimited).
func (n *Node) SortChildrenDepth(criteria SortCriteria, maxDepth int) {
	if maxDepth == 0 || len(n.Children) == 0 {
		return
	}

	nextDepth := maxDepth - 1
	if maxDepth < 0 {
		nextDepth = -1
	}

	if nextDepth != 0 {
		for _, child := range n.Children {
			if child.Type == NodeTypeDir {
				child.SortChildrenDepth(criteria, nextDepth)
			}
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
