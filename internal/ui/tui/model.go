package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
)

type Model struct {
	Root         *model.Node
	Current      *model.Node
	Cursor       int
	Sort         model.SortCriteria
	Filter       string
	Filtering    bool
	Width        int
	Height       int
	Dialog       DialogKind
	Deleting     bool
	StatusMsg    string
	TargetCaches []*model.Node
	CacheBytes   int64
}

func NewModel(root *model.Node) Model {
	if root != nil {
		root.SortChildrenDepth(model.SortBySize, 1)
	}
	caches := cleaner.CollectCaches(root)
	var cacheBytes int64
	if len(caches) == 0 && root != nil {
		cacheBytes, _ = cleaner.TagCaches(root)
	} else {
		for _, c := range caches {
			cacheBytes += c.Size
		}
	}
	return Model{
		Root:       root,
		Current:    root,
		Cursor:     0,
		Sort:       model.SortBySize,
		CacheBytes: cacheBytes,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
