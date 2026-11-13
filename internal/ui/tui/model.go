package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
)

type ViewMode int

const (
	ViewTree ViewMode = iota
	ViewTopFiles
	ViewFileTypes
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
	Mode         ViewMode
	Dialog       DialogKind
	Deleting     bool
	StatusMsg    string
	Selected     map[*model.Node]bool
	TargetCaches []*model.Node
	CacheChecked map[*model.Node]bool
	CacheCursor  int
	CacheBytes   int64
	TopFilesList []*model.Node
	ExtStatsList []model.ExtStat
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
		Root:         root,
		Current:      root,
		Cursor:       0,
		Sort:         model.SortBySize,
		Mode:         ViewTree,
		Selected:     make(map[*model.Node]bool),
		CacheChecked: make(map[*model.Node]bool),
		CacheBytes:   cacheBytes,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) SelectedNodes() ([]*model.Node, int64) {
	if len(m.Selected) == 0 {
		return nil, 0
	}
	res := make([]*model.Node, 0, len(m.Selected))
	var total int64
	for n, ok := range m.Selected {
		if ok {
			res = append(res, n)
			total += n.Size
		}
	}
	return res, total
}
