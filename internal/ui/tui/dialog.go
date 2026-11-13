package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
	"goclean/internal/ui"
)

type DialogKind int

const (
	DialogNone DialogKind = iota
	DialogDeleteTarget
	DialogCleanAllCaches
)

var (
	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5252")).
			Padding(1, 2).
			Width(78)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			MarginBottom(1)

	treePreviewBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#444444")).
				Padding(0, 1).
				Width(72)

	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
	uncheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

func renderDeleteDialog(targets []*model.Node) string {
	if len(targets) == 0 {
		return ""
	}
	if len(targets) == 1 {
		node := targets[0]
		miniTree := ui.RenderMiniTree(node, 2, 5)
		content := fmt.Sprintf("%s\n\nTarget path:\n%s\n\nFiles & folders to be deleted:\n%s\n\n[Enter / y] Confirm Delete   [Esc / n] Cancel",
			modalTitleStyle.Render("⚠️  CONFIRM DELETION"),
			node.FullPath(),
			treePreviewBoxStyle.Render(miniTree),
		)
		return modalBoxStyle.Render(content)
	}

	var totalSize int64
	var sb strings.Builder
	limit := len(targets)
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		t := targets[i]
		icon := "📁 "
		if t.Type != model.NodeTypeDir {
			icon = "📄 "
		}
		sb.WriteString(fmt.Sprintf("• %s%s (%s)\n", icon, t.FullPath(), ui.FormatBytes(t.Size)))
	}
	for _, t := range targets {
		totalSize += t.Size
	}
	if len(targets) > limit {
		sb.WriteString(fmt.Sprintf("... and %d more selected items\n", len(targets)-limit))
	}

	content := fmt.Sprintf("%s\n\nSelected items: %d  |  Total size: %s\n\nTargets to be deleted:\n%s\n\n[Enter / y] Delete Selected   [Esc / n] Cancel",
		modalTitleStyle.Render("⚠️  CONFIRM BATCH DELETION"),
		len(targets),
		ui.FormatBytes(totalSize),
		treePreviewBoxStyle.Render(strings.TrimRight(sb.String(), "\n")),
	)
	return modalBoxStyle.Render(content)
}

func renderCleanCachesDialog(baseRoot *model.Node, caches []*model.Node, checked map[*model.Node]bool, cursor int) string {
	var selectedCount int
	var selectedBytes int64
	for _, c := range caches {
		if checked[c] {
			selectedCount++
			selectedBytes += c.Size
		}
	}

	var basePath string
	if baseRoot != nil {
		basePath = baseRoot.FullPath()
	}

	maxVisible := 6
	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(caches) {
		end = len(caches)
	}

	var listSB strings.Builder
	for i := start; i < end; i++ {
		c := caches[i]
		box := uncheckedStyle.Render("[ ]")
		if checked[c] {
			box = checkedStyle.Render("[✓]")
		}
		pointer := "  "
		if i == cursor {
			pointer = "> "
		}

		displayPath := c.FullPath()
		if basePath != "" && strings.HasPrefix(displayPath, basePath) && displayPath != basePath {
			rel := strings.TrimPrefix(displayPath, basePath)
			rel = strings.TrimLeft(rel, "/\\")
			if rel != "" {
				displayPath = rel
			}
		}

		line := fmt.Sprintf("%s%s 📁 %-32s [%s] (%s)",
			pointer,
			box,
			truncateRunes(displayPath, 32),
			c.CacheKind,
			ui.FormatBytes(c.Size),
		)
		listSB.WriteString(line + "\n")
	}

	var preview string
	if cursor >= 0 && cursor < len(caches) {
		preview = ui.RenderMiniTree(caches[cursor], 1, 4)
	}

	content := fmt.Sprintf("%s\n\nSelected: %d / %d caches  |  Reclaimable: %s\n\n%s\nPreview of highlighted cache:\n%s\n\n[↑/↓] Move  [Space] Toggle  [a] All/None  [Enter/y] Clean  [Esc/n] Cancel",
		modalTitleStyle.Render("🧹 INTERACTIVE CACHE CLEANER"),
		selectedCount,
		len(caches),
		ui.FormatBytes(selectedBytes),
		treePreviewBoxStyle.Render(strings.TrimRight(listSB.String(), "\n")),
		treePreviewBoxStyle.Render(preview),
	)
	return modalBoxStyle.Render(content)
}
