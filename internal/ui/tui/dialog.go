package tui

import (
	"fmt"

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
			Width(76)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			MarginBottom(1)

	treePreviewBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#444444")).
				Padding(0, 1).
				Width(70)
)

func renderDeleteDialog(node *model.Node) string {
	miniTree := ui.RenderMiniTree(node, 2, 5)
	content := fmt.Sprintf("%s\n\nTarget path:\n%s\n\nFiles & folders to be deleted:\n%s\n\n[Enter / y] Confirm Delete   [Esc / n] Cancel",
		modalTitleStyle.Render("⚠️  CONFIRM DELETION"),
		node.FullPath(),
		treePreviewBoxStyle.Render(miniTree),
	)
	return modalBoxStyle.Render(content)
}

func renderCleanCachesDialog(baseRoot *model.Node, caches []*model.Node, totalBytes int64) string {
	preview := ui.RenderCacheTreePreview(baseRoot, caches, 5, 3)
	content := fmt.Sprintf("%s\n\nFound caches: %d directories  |  Reclaimable space: %s\n\nCache directories & contents to be deleted:\n%s\n\n[Enter / y] Clean All   [Esc / n] Cancel",
		modalTitleStyle.Render("🧹 CLEAN ALL CACHES"),
		len(caches),
		ui.FormatBytes(totalBytes),
		treePreviewBoxStyle.Render(preview),
	)
	return modalBoxStyle.Render(content)
}
