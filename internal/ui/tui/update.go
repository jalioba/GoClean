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

func deleteTargetsCmd(targets []*model.Node, isBatch bool) tea.Cmd {
	return func() tea.Msg {
		if len(targets) == 1 && !isBatch {
			target := targets[0]
			err := cleaner.DeleteSafely(target.FullPath(), false)
			if err != nil {
				return deleteDoneMsg{err: err}
			}
			return deleteDoneMsg{
				deleted:   []*model.Node{target},
				reclaimed: target.Size,
			}
		}

		reclaimed, deleted, errs := cleaner.DeleteNodesParallel(targets, 0, false)
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

func (m Model) activeListLen() int {
	switch m.Mode {
	case ViewTopFiles:
		return len(m.TopFilesList)
	case ViewFileTypes:
		return len(m.ExtStatsList)
	default:
		return len(m.filteredChildren())
	}
}

func (m Model) currentTargetsForDelete() []*model.Node {
	selected, _ := m.SelectedNodes()
	if len(selected) > 0 {
		return selected
	}
	if m.Mode == ViewTopFiles {
		if m.Cursor >= 0 && m.Cursor < len(m.TopFilesList) {
			return []*model.Node{m.TopFilesList[m.Cursor]}
		}
		return nil
	}
	items := m.filteredChildren()
	if m.Cursor >= 0 && m.Cursor < len(items) {
		return []*model.Node{items[m.Cursor]}
	}
	return nil
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
			delete(m.Selected, node)
			m.removeNodeFromParent(node)
		}
		if m.Mode == ViewTopFiles && m.Current != nil {
			m.TopFilesList = m.Current.TopFiles(50)
			if m.Cursor >= len(m.TopFilesList) && m.Cursor > 0 {
				m.Cursor = len(m.TopFilesList) - 1
			}
		} else if m.Mode == ViewFileTypes && m.Current != nil {
			m.ExtStatsList = m.Current.ExtensionStats(50)
		}

		if msg.isBatch {
			if msg.err != nil {
				m.StatusMsg = fmt.Sprintf("Deleted %d items (%s), with error: %v", len(msg.deleted), ui.FormatBytes(msg.reclaimed), msg.err)
			} else {
				m.StatusMsg = fmt.Sprintf("Deleted %d items, reclaimed %s", len(msg.deleted), ui.FormatBytes(msg.reclaimed))
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

		listLen := m.activeListLen()

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < listLen-1 {
				m.Cursor++
			}

		case "pgup":
			m.Cursor -= 10
			if m.Cursor < 0 {
				m.Cursor = 0
			}

		case "pgdown":
			m.Cursor += 10
			if m.Cursor >= listLen {
				m.Cursor = listLen - 1
			}
			if m.Cursor < 0 {
				m.Cursor = 0
			}

		case "home", "g":
			m.Cursor = 0

		case "end", "G":
			if listLen > 0 {
				m.Cursor = listLen - 1
			}

		case " ":
			if m.Mode == ViewTree {
				items := m.filteredChildren()
				if m.Cursor >= 0 && m.Cursor < len(items) {
					target := items[m.Cursor]
					if m.Selected[target] {
						delete(m.Selected, target)
					} else {
						m.Selected[target] = true
					}
					if m.Cursor < len(items)-1 {
						m.Cursor++
					}
				}
			} else if m.Mode == ViewTopFiles {
				if m.Cursor >= 0 && m.Cursor < len(m.TopFilesList) {
					target := m.TopFilesList[m.Cursor]
					if m.Selected[target] {
						delete(m.Selected, target)
					} else {
						m.Selected[target] = true
					}
					if m.Cursor < len(m.TopFilesList)-1 {
						m.Cursor++
					}
				}
			}

		case "f":
			if m.Mode == ViewTopFiles {
				m.Mode = ViewTree
				m.Cursor = 0
				m.StatusMsg = "View: Directory Tree"
			} else {
				m.Mode = ViewTopFiles
				m.TopFilesList = m.Current.TopFiles(50)
				m.Cursor = 0
				m.StatusMsg = "View: Top Largest Files in current subtree"
			}

		case "t":
			if m.Mode == ViewFileTypes {
				m.Mode = ViewTree
				m.Cursor = 0
				m.StatusMsg = "View: Directory Tree"
			} else {
				m.Mode = ViewFileTypes
				m.ExtStatsList = m.Current.ExtensionStats(50)
				m.Cursor = 0
				m.StatusMsg = "View: Disk Usage by File Extension"
			}

		case "enter", "right", "l":
			if m.Mode != ViewTree {
				return m, nil
			}
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
			if m.Mode != ViewTree {
				m.Mode = ViewTree
				m.Cursor = 0
				m.StatusMsg = ""
				return m, nil
			}
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
			if m.Mode != ViewTree {
				return m, nil
			}
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
			if m.Mode == ViewTree {
				m.Filtering = true
				m.Filter = ""
			}

		case "d":
			if m.Deleting || m.Mode == ViewFileTypes {
				return m, nil
			}
			targets := m.currentTargetsForDelete()
			if len(targets) > 0 {
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
				checked := make(map[*model.Node]bool, len(caches))
				for _, c := range caches {
					checked[c] = true
					total += c.Size
				}
				m.TargetCaches = caches
				m.CacheChecked = checked
				m.CacheCursor = 0
				m.CacheBytes = total
				m.Dialog = DialogCleanAllCaches
			}
		}
	}
	return m, nil
}

func (m Model) handleDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Dialog == DialogCleanAllCaches {
		switch msg.String() {
		case "up", "k":
			if m.CacheCursor > 0 {
				m.CacheCursor--
			}
			return m, nil
		case "down", "j":
			if m.CacheCursor < len(m.TargetCaches)-1 {
				m.CacheCursor++
			}
			return m, nil
		case " ":
			if m.CacheCursor >= 0 && m.CacheCursor < len(m.TargetCaches) {
				target := m.TargetCaches[m.CacheCursor]
				m.CacheChecked[target] = !m.CacheChecked[target]
			}
			return m, nil
		case "a":
			allChecked := true
			for _, c := range m.TargetCaches {
				if !m.CacheChecked[c] {
					allChecked = false
					break
				}
			}
			for _, c := range m.TargetCaches {
				m.CacheChecked[c] = !allChecked
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "esc", "n", "N":
		m.Dialog = DialogNone
		return m, nil

	case "enter", "y", "Y":
		if m.Dialog == DialogDeleteTarget {
			targets := m.currentTargetsForDelete()
			if len(targets) > 0 {
				m.Dialog = DialogNone
				m.Deleting = true
				if len(targets) == 1 {
					m.StatusMsg = fmt.Sprintf("Deleting %s...", targets[0].Name)
					return m, deleteTargetsCmd(targets, false)
				}
				m.StatusMsg = fmt.Sprintf("Deleting %d selected items...", len(targets))
				return m, deleteTargetsCmd(targets, true)
			}
		} else if m.Dialog == DialogCleanAllCaches {
			var chosen []*model.Node
			for _, c := range m.TargetCaches {
				if m.CacheChecked[c] {
					chosen = append(chosen, c)
				}
			}
			m.Dialog = DialogNone
			if len(chosen) == 0 {
				m.StatusMsg = "No caches selected for deletion"
				return m, nil
			}
			m.Deleting = true
			m.StatusMsg = fmt.Sprintf("Cleaning %d selected cache directories...", len(chosen))
			return m, deleteTargetsCmd(chosen, true)
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
