package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/model"
	"goclean/internal/ui"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
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
				if selected.Type == model.NodeTypeDir && len(selected.Children) > 0 {
					m.Current = selected
					m.Cursor = 0
					m.Filter = ""
				}
			}

		case "esc", "backspace", "left", "h":
			if m.Current.Parent != nil {
				m.Current = m.Current.Parent
				m.Cursor = 0
				m.Filter = ""
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
			m.Current.SortChildren(m.Sort)

		case "/":
			m.Filtering = true
			m.Filter = ""

		case "d":
			items := m.filteredChildren()
			if len(items) > 0 && m.Cursor < len(items) {
				m.Dialog = DialogDeleteTarget
			}

		case "c":
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
				err := cleaner.DeleteSafely(target.Path, false)
				if err != nil {
					m.StatusMsg = fmt.Sprintf("Delete error: %v", err)
				} else {
					m.StatusMsg = fmt.Sprintf("Deleted: %s", target.Name)
					m.removeNodeFromParent(target)
				}
			}
		} else if m.Dialog == DialogCleanAllCaches {
			deletedCount := 0
			var reclaimed int64
			for _, c := range m.TargetCaches {
				if err := cleaner.DeleteSafely(c.Path, false); err == nil {
					deletedCount++
					reclaimed += c.Size
					m.removeNodeFromParent(c)
				}
			}
			m.StatusMsg = fmt.Sprintf("Cleaned %d caches, reclaimed %s", deletedCount, ui.FormatBytes(reclaimed))
		}
		m.Dialog = DialogNone
		m.Root.PostOrderAggregate()
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
			m.Filter = m.Filter[:len(m.Filter)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.Filter += msg.String()
			m.Cursor = 0
		}
	}
	return m, nil
}

func (m Model) filteredChildren() []*model.Node {
	if m.Filter == "" {
		return m.Current.Children
	}
	var res []*model.Node
	for _, child := range m.Current.Children {
		if strings.Contains(strings.ToLower(child.Name), strings.ToLower(m.Filter)) {
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
	for i, c := range p.Children {
		if c == target {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	if m.Cursor >= len(p.Children) && m.Cursor > 0 {
		m.Cursor--
	}
}
