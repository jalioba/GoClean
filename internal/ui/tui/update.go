package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
	"goclean/internal/ui"
)

type deleteDoneMsg struct {
	deleted   []*model.Node
	reclaimed int64
	err       error
	isBatch   bool
}

func deleteTargetCmd(target *model.Node) tea.Cmd {
	return func() tea.Msg {
		err := cleaner.DeleteSafely(target.FullPath(), false)
		if err != nil {
			return deleteDoneMsg{err: err}
		}
		return deleteDoneMsg{
			deleted:   []*model.Node{target},
			reclaimed: target.Size,
		}
	}
}

func cleanCachesCmd(caches []*model.Node) tea.Cmd {
	return func() tea.Msg {
		reclaimed, deleted, errs := cleaner.DeleteNodesParallel(caches, 0, false)
		var firstErr error
		if len(errs) > 0 {
			firstErr = errs[0]
		}
		return deleteDoneMsg{
			deleted:   deleted,
			reclaimed: reclaimed,
			err:       firstErr,
			isBatch:   true,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case deleteDoneMsg:
		m.Deleting = false
		for _, node := range msg.deleted {
			m.removeNodeFromParent(node)
		}
		if msg.isBatch {
			if msg.err != nil {
				m.StatusMsg = fmt.Sprintf("Cleaned %d caches (%s), with error: %v", len(msg.deleted), ui.FormatBytes(msg.reclaimed), msg.err)
			} else {
				m.StatusMsg = fmt.Sprintf("Cleaned %d caches, reclaimed %s", len(msg.deleted), ui.FormatBytes(msg.reclaimed))
			}
			m.TargetCaches = nil
		} else {
			if msg.err != nil {
				m.StatusMsg = fmt.Sprintf("Delete error: %v", msg.err)
			} else if len(msg.deleted) > 0 {
				m.StatusMsg = fmt.Sprintf("Deleted: %s (%s)", msg.deleted[0].Name, ui.FormatBytes(msg.reclaimed))
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.Dialog != DialogNone {
			return m.handleDialogKeys(msg)
		}

		if m.Filtering {
			return m.handleFilterKeys(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			items := m.filteredChildren()
			if m.Cursor < len(items)-1 {
				m.Cursor++
			}

		case "enter", "right", "l":
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				selected := items[m.Cursor]
				if selected.Type == model.NodeTypeDir {
					if len(selected.Children) == 0 {
						m.StatusMsg = fmt.Sprintf("Directory %q is empty", selected.Name)
					} else {
						selected.SortChildrenDepth(m.Sort, 1)
						m.Current = selected
						m.Cursor = 0
						m.Filter = ""
						m.StatusMsg = ""
					}
				}
			}

		case "esc", "backspace", "left", "h":
			if m.Current.Parent != nil {
				prev := m.Current
				parent := m.Current.Parent
				parent.SortChildrenDepth(m.Sort, 1)
				m.Current = parent
				m.Filter = ""
				m.Cursor = 0
				for i, c := range parent.Children {
					if c == prev {
						m.Cursor = i
						break
					}
				}
			}

		case "s":
			switch m.Sort {
			case model.SortBySize:
				m.Sort = model.SortByName
				m.StatusMsg = "Sort: by name"
			case model.SortByName:
				m.Sort = model.SortByItems
				m.StatusMsg = "Sort: by item count"
			case model.SortByItems:
				m.Sort = model.SortBySize
				m.StatusMsg = "Sort: by size"
			}
			m.Current.SortChildrenDepth(m.Sort, 1)

		case "/":
			m.Filtering = true
			m.Filter = ""

		case "d":
			if m.Deleting {
				return m, nil
			}
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				m.Dialog = DialogDeleteTarget
			}

		case "c":
			if m.Deleting {
				return m, nil
			}
			caches := cleaner.CollectCaches(m.Current)
			if len(caches) == 0 {
				m.StatusMsg = "No caches found in current directory"
			} else {
				var total int64
				for _, c := range caches {
					total += c.Size
				}
				m.TargetCaches = caches
				m.CacheBytes = total
				m.Dialog = DialogCleanAllCaches
			}
		}
	}
	return m, nil
}

func (m Model) handleDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "N":
		m.Dialog = DialogNone
		return m, nil

	case "enter", "y", "Y":
		if m.Dialog == DialogDeleteTarget {
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				target := items[m.Cursor]
				m.Dialog = DialogNone
				m.Deleting = true
				m.StatusMsg = fmt.Sprintf("Deleting %s...", target.Name)
				return m, deleteTargetCmd(target)
			}
		} else if m.Dialog == DialogCleanAllCaches {
			caches := m.TargetCaches
			m.Dialog = DialogNone
			m.Deleting = true
			m.StatusMsg = fmt.Sprintf("Cleaning %d cache directories...", len(caches))
			return m, cleanCachesCmd(caches)
		}
		m.Dialog = DialogNone
		return m, nil
	}
	return m, nil
}

func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Filtering = false
	case "backspace":
		if len(m.Filter) > 0 {
			runes := []rune(m.Filter)
			m.Filter = string(runes[:len(runes)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			m.Filter += string(msg.Runes)
			m.Cursor = 0
		}
	}
	return m, nil
}

func (m Model) filteredChildren() []*model.Node {
	if m.Filter == "" {
		return m.Current.Children
	}
	lowerFilter := strings.ToLower(m.Filter)
	res := make([]*model.Node, 0, len(m.Current.Children))
	for _, child := range m.Current.Children {
		if strings.Contains(strings.ToLower(child.Name), lowerFilter) {
			res = append(res, child)
		}
	}
	return res
}

func (m *Model) removeNodeFromParent(target *model.Node) {
	if target.Parent == nil {
		return
	}
	p := target.Parent
	p.RemoveChild(target)
	items := m.filteredChildren()
	if m.Cursor >= len(items) && m.Cursor > 0 {
		m.Cursor = len(items) - 1
	}
}
