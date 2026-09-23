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
	content := fmt.Sprintf("%s\n\nВы действительно хотите безвозвратно удалить:\n📁 %s\nРазмер: %s (%s файлов)?\n\n[Enter / y] Подтвердить   [Esc / n] Отмена",
		modalTitleStyle.Render("⚠️  ПОДТВЕРЖДЕНИЕ УДАЛЕНИЯ"),
		node.Path,
		ui.FormatBytes(node.Size),
		ui.FormatNumber(node.ItemCount),
	)
	return modalBoxStyle.Render(content)
}

func renderCleanCachesDialog(caches []*model.Node, totalBytes int64) string {
	content := fmt.Sprintf("%s\n\nНайдено кэшей для очистки: %d папок\nБудет освобождено: %s\n\nУдалить все обнаруженные кэши безвозвратно?\n\n[Enter / y] Очистить   [Esc / n] Отмена",
		modalTitleStyle.Render("🧹 ОЧИСТКА ВСЕХ КЭШЕЙ"),
		len(caches),
		ui.FormatBytes(totalBytes),
	)
	return modalBoxStyle.Render(content)
}
