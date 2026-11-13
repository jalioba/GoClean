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

func TestFullPathAndRemoveChild(t *testing.T) {
	root := &Node{Name: "root", Path: "/tmp/root", Type: NodeTypeDir}
	dirA := &Node{Name: "dirA", Type: NodeTypeDir}
	fileA1 := &Node{Name: "fileA1.txt", Type: NodeTypeFile, Size: 500, ItemCount: 1}
	dirA.AddChild(fileA1)
	root.AddChild(dirA)
	root.PostOrderAggregate()

	if got := fileA1.FullPath(); got != "/tmp/root/dirA/fileA1.txt" {
		t.Fatalf("unexpected FullPath: %s", got)
	}

	if !dirA.RemoveChild(fileA1) {
		t.Fatal("expected RemoveChild to return true")
	}
	if root.Size != 0 || dirA.Size != 0 {
		t.Fatalf("expected sizes to be 0 after removal, got root=%d dirA=%d", root.Size, dirA.Size)
	}
	if root.ItemCount != 1 || dirA.ItemCount != 0 {
		t.Fatalf("expected root items=1 dirA items=0, got root=%d dirA=%d", root.ItemCount, dirA.ItemCount)
	}
}

func TestTopFilesAndExtensionStats(t *testing.T) {
	root := &Node{Name: "root", Path: "/tmp/root", Type: NodeTypeDir}
	f1 := &Node{Name: "a.go", Type: NodeTypeFile, Size: 100, ItemCount: 1}
	f2 := &Node{Name: "b.go", Type: NodeTypeFile, Size: 300, ItemCount: 1}
	f3 := &Node{Name: "video.mp4", Type: NodeTypeFile, Size: 1000, ItemCount: 1}
	root.AddChild(f1)
	root.AddChild(f2)
	root.AddChild(f3)
	root.PostOrderAggregate()

	top := root.TopFiles(2)
	if len(top) != 2 || top[0].Name != "video.mp4" || top[1].Name != "b.go" {
		t.Fatalf("unexpected TopFiles result: %+v", top)
	}

	exts := root.ExtensionStats(0)
	if len(exts) != 2 || exts[0].Ext != ".mp4" || exts[1].Ext != ".go" || exts[1].Size != 400 || exts[1].Count != 2 {
		t.Fatalf("unexpected ExtensionStats result: %+v", exts)
	}
}
