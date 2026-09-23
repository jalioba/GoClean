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
	StatusMsg    string
	TargetCaches []*model.Node
	CacheBytes   int64
}

func NewModel(root *model.Node) Model {
	root.SortChildren(model.SortBySize)
	cacheBytes, _ := cleaner.TagCaches(root)
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
