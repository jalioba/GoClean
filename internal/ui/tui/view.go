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
)

func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	header := fmt.Sprintf(" GoClean v1.0 ─ Directory: %s ", m.Current.Path)
	sb.WriteString(headerBoxStyle.Width(m.Width).Render(header))
	sb.WriteString("\n")

	tableHeader := fmt.Sprintf("  %-30s  %10s  %-15s  %6s  %s",
		"Name", "Size", "% Usage", "Items", "Badge")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(tableHeader))
	sb.WriteString("\n")

	items := m.filteredChildren()
	maxRows := m.Height - 6
	if maxRows < 5 {
		maxRows = 10
	}

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
		item := items[i]
		ratio := float64(item.Size) / float64(parentSize)
		bar := ui.RenderBar(ratio, 12, true)
		pct := ui.FormatPercent(ratio)

		icon := "📁 "
		if item.Type == model.NodeTypeFile {
			icon = "📄 "
		}

		name := item.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}

		tag := ""
		if item.IsCache {
			tag = fmt.Sprintf("[CACHE: %s]", item.CacheKind)
		}

		row := fmt.Sprintf(" %s%-28s  %10s  [%s] %s  %6s  %s",
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

	// Status / Helper bar
	if m.StatusMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD600")).Render("⚡ " + m.StatusMsg))
		sb.WriteString("\n")
	}

	help := "[↑/↓/j/k] Navigate | [Enter/→] Open | [Esc/←] Back | [s] Sort | [c] Clean Caches | [d] Delete | [q] Quit"
	if m.Filtering {
		help = fmt.Sprintf("🔍 Search: %s_ [Esc to cancel]", m.Filter)
	}
	sb.WriteString(footerBoxStyle.Width(m.Width).Render(help))

	if m.Dialog == DialogDeleteTarget && len(items) > 0 && m.Cursor < len(items) {
		return renderDeleteDialog(items[m.Cursor])
	}
	if m.Dialog == DialogCleanAllCaches {
		return renderCleanCachesDialog(m.TargetCaches, m.CacheBytes)
	}

	return sb.String()
}
