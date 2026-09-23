package ui

import (
	"strings"
	"testing"

	"goclean/internal/model"
)

func TestRenderCLITree(t *testing.T) {
	root := &model.Node{
		Name: "project",
		Path: "/project",
		Type: model.NodeTypeDir,
	}
	cacheNode := &model.Node{
		Name:      "node_modules",
		Path:      "/project/node_modules",
		Type:      model.NodeTypeDir,
		Size:      1000,
		IsCache:   true,
		CacheKind: "npm/node",
	}
	srcNode := &model.Node{
		Name: "src",
		Path: "/project/src",
		Type: model.NodeTypeDir,
		Size: 500,
	}
	root.AddChild(cacheNode)
	root.AddChild(srcNode)
	root.PostOrderAggregate()

	out := RenderCLITree(root, 2, 20)
	if !strings.Contains(out, "node_modules") {
		t.Errorf("expected tree to contain node_modules, got %s", out)
	}
	if !strings.Contains(out, "[CACHE: npm/node]") {
		t.Errorf("expected tree to highlight cache tag, got %s", out)
	}
}
