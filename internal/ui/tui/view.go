package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
	"goclean/internal/ui"
)

var (
	headerBoxStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#005F87"))

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	footerBoxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#111111")).
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	statusMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD600"))
	selectedBadge    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
)

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	modeLabel := "Browser"
	switch m.Mode {
	case ViewTopFiles:
		modeLabel = "Top Largest Files"
	case ViewFileTypes:
		modeLabel = "File Types"
	}

	header := fmt.Sprintf(" GoClean v2.0 [%s] ─ %s (%s) ",
		modeLabel,
		m.Current.FullPath(),
		ui.FormatBytes(m.Current.Size),
	)
	sb.WriteString(headerBoxStyle.Width(m.Width).Render(header))
	sb.WriteString("\n")

	maxRows := m.Height - 6
	if maxRows < 5 {
		maxRows = 10
	}

	switch m.Mode {
	case ViewTopFiles:
		m.renderTopFilesView(&sb, maxRows)
	case ViewFileTypes:
		m.renderFileTypesView(&sb, maxRows)
	default:
		m.renderTreeView(&sb, maxRows)
	}

	// Selected items indicator + Status bar
	selNodes, selBytes := m.SelectedNodes()
	if len(selNodes) > 0 {
		selInfo := fmt.Sprintf("📌 Selected: %d items (%s) — press [d] to delete selected", len(selNodes), ui.FormatBytes(selBytes))
		sb.WriteString(selectedBadge.Render(selInfo))
		sb.WriteString("\n")
	} else if m.StatusMsg != "" {
		sb.WriteString(statusMsgStyle.Render("⚡ " + m.StatusMsg))
		sb.WriteString("\n")
	}

	help := "[↑/↓] Move | [Space] Select | [Enter/→] Open | [Esc/←] Back | [s] Sort | [f] Top Files | [t] Types | [c] Caches | [d] Delete | [q] Quit"
	if m.Filtering {
		help = fmt.Sprintf("🔍 Search: %s_ [Esc to finish]", m.Filter)
	}
	sb.WriteString(footerBoxStyle.Width(m.Width).Render(help))

	if m.Dialog == DialogDeleteTarget {
		targets := m.currentTargetsForDelete()
		if len(targets) > 0 {
			dialog := renderDeleteDialog(targets)
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, dialog)
		}
	}
	if m.Dialog == DialogCleanAllCaches {
		dialog := renderCleanCachesDialog(m.Current, m.TargetCaches, m.CacheChecked, m.CacheCursor)
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return sb.String()
}

func (m Model) renderTreeView(sb *strings.Builder, maxRows int) {
	tableHeader := fmt.Sprintf("      %-28s  %10s  %-15s  %6s  %s",
		"Name", "Size", "% Usage", "Items", "Badge")
	sb.WriteString(tableHeaderStyle.Render(tableHeader))
	sb.WriteString("\n")

	items := m.filteredChildren()
	start := 0
	if m.Cursor >= maxRows {
		start = m.Cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	parentSize := m.Current.Size
	if parentSize == 0 {
		parentSize = 1
	}

	hasAnySelected := len(m.Selected) > 0

	for i := start; i < end; i++ {
		item := items[i]
		ratio := float64(item.Size) / float64(parentSize)
		bar := ui.RenderBar(ratio, 12, true)
		pct := ui.FormatPercent(ratio)

		icon := "📁 "
		if item.Type == model.NodeTypeFile {
			icon = "📄 "
		}

		mark := "    "
		if m.Selected[item] {
			mark = "[✓] "
		} else if hasAnySelected {
			mark = "[ ] "
		}

		name := truncateRunes(item.Name, 28)

		tag := ""
		if item.IsCache {
			tag = fmt.Sprintf("[CACHE: %s]", item.CacheKind)
		}

		row := fmt.Sprintf(" %s%s%-28s  %10s  [%s] %s  %6s  %s",
			mark,
			icon,
			name,
			ui.FormatBytes(item.Size),
			bar,
			pct,
			ui.FormatNumber(item.ItemCount),
			tag,
		)

		if i == m.Cursor {
			sb.WriteString(selectedRowStyle.Width(m.Width).Render("> " + row))
		} else {
			sb.WriteString(normalRowStyle.Width(m.Width).Render("  " + row))
		}
		sb.WriteString("\n")
	}
}

func (m Model) renderTopFilesView(sb *strings.Builder, maxRows int) {
	tableHeader := fmt.Sprintf("      %-38s  %10s  %-15s",
		"Relative File Path", "Size", "% Usage")
	sb.WriteString(tableHeaderStyle.Render(tableHeader))
	sb.WriteString("\n")

	items := m.TopFilesList
	start := 0
	if m.Cursor >= maxRows {
		start = m.Cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	parentSize := m.Current.Size
	if parentSize == 0 {
		parentSize = 1
	}
	basePath := m.Current.FullPath()
	hasAnySelected := len(m.Selected) > 0

	for i := start; i < end; i++ {
		item := items[i]
		ratio := float64(item.Size) / float64(parentSize)
		bar := ui.RenderBar(ratio, 12, true)
		pct := ui.FormatPercent(ratio)

		mark := "    "
		if m.Selected[item] {
			mark = "[✓] "
		} else if hasAnySelected {
			mark = "[ ] "
		}

		displayPath := item.FullPath()
		if basePath != "" && strings.HasPrefix(displayPath, basePath) && displayPath != basePath {
			rel := strings.TrimPrefix(displayPath, basePath)
			rel = strings.TrimLeft(rel, "/\\")
			if rel != "" {
				displayPath = rel
			}
		}

		row := fmt.Sprintf(" %s📄 %-38s  %10s  [%s] %s",
			mark,
			truncateRunes(displayPath, 38),
			ui.FormatBytes(item.Size),
			bar,
			pct,
		)

		if i == m.Cursor {
			sb.WriteString(selectedRowStyle.Width(m.Width).Render("> " + row))
		} else {
			sb.WriteString(normalRowStyle.Width(m.Width).Render("  " + row))
		}
		sb.WriteString("\n")
	}
}

func (m Model) renderFileTypesView(sb *strings.Builder, maxRows int) {
	tableHeader := fmt.Sprintf("  %-18s  %12s  %-18s  %10s",
		"Extension", "Total Size", "% Usage", "Files")
	sb.WriteString(tableHeaderStyle.Render(tableHeader))
	sb.WriteString("\n")

	items := m.ExtStatsList
	start := 0
	if m.Cursor >= maxRows {
		start = m.Cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	parentSize := m.Current.Size
	if parentSize == 0 {
		parentSize = 1
	}

	for i := start; i < end; i++ {
		st := items[i]
		ratio := float64(st.Size) / float64(parentSize)
		bar := ui.RenderBar(ratio, 12, true)
		pct := ui.FormatPercent(ratio)

		row := fmt.Sprintf("  %-18s  %12s  [%s] %s  %10s",
			truncateRunes(st.Ext, 18),
			ui.FormatBytes(st.Size),
			bar,
			pct,
			ui.FormatNumber(st.Count),
		)

		if i == m.Cursor {
			sb.WriteString(selectedRowStyle.Width(m.Width).Render("> " + row))
		} else {
			sb.WriteString(normalRowStyle.Width(m.Width).Render("  " + row))
		}
		sb.WriteString("\n")
	}
}
