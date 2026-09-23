package model

import (
	"testing"
)

func TestPostOrderAggregateAndSort(t *testing.T) {
	root := &Node{Name: "root", Path: "/root", Type: NodeTypeDir}
	childA := &Node{Name: "dirA", Path: "/root/dirA", Type: NodeTypeDir}
	childB := &Node{Name: "fileB", Path: "/root/fileB", Type: NodeTypeFile, Size: 500, ItemCount: 1}
	childA1 := &Node{Name: "fileA1", Path: "/root/dirA/fileA1", Type: NodeTypeFile, Size: 1500, ItemCount: 1}

	childA.AddChild(childA1)
	root.AddChild(childA)
	root.AddChild(childB)

	root.PostOrderAggregate()

	if childA.Size != 1500 {
		t.Fatalf("expected childA size 1500, got %d", childA.Size)
	}
	if childA.ItemCount != 1 {
		t.Fatalf("expected childA items 1, got %d", childA.ItemCount)
	}
	if root.Size != 2000 {
		t.Fatalf("expected root size 2000, got %d", root.Size)
	}
	if root.ItemCount != 3 { // fileA1, dirA, fileB
		t.Fatalf("expected root items 3, got %d", root.ItemCount)
	}

	root.SortChildren(SortBySize)
	if root.Children[0].Name != "dirA" {
		t.Fatalf("expected largest child dirA first, got %s", root.Children[0].Name)
	}
}
