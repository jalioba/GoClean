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
			Width(65)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			MarginBottom(1)
)

func renderDeleteDialog(node *model.Node) string {
	content := fmt.Sprintf("%s\n\nAre you sure you want to permanently delete:\n📁 %s\nSize: %s (%s items)?\n\n[Enter / y] Confirm   [Esc / n] Cancel",
		modalTitleStyle.Render("⚠️  CONFIRM DELETION"),
		node.Path,
		ui.FormatBytes(node.Size),
		ui.FormatNumber(node.ItemCount),
	)
	return modalBoxStyle.Render(content)
}

func renderCleanCachesDialog(caches []*model.Node, totalBytes int64) string {
	content := fmt.Sprintf("%s\n\nFound caches to clean: %d directories\nReclaimable space: %s\n\nPermanently delete all detected caches?\n\n[Enter / y] Clean   [Esc / n] Cancel",
		modalTitleStyle.Render("🧹 CLEAN ALL CACHES"),
		len(caches),
		ui.FormatBytes(totalBytes),
	)
	return modalBoxStyle.Render(content)
}
