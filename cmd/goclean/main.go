package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"goclean/internal/cleaner"
	"goclean/internal/scanner"
	"goclean/internal/ui"
	"goclean/internal/ui/tui"
)

func main() {
	interactive := flag.Bool("i", false, "Запустить интерактивный TUI режим (как ncdu)")
	workers := flag.Int("w", 0, "Количество параллельных воркеров (по умолчанию NumCPU * 2)")
	maxDepth := flag.Int("d", 2, "Глубина отображения дерева каталогов")
	cleanCache := flag.Bool("clean-cache", false, "Найти и удалить все кэши (node_modules, target, etc.)")
	dryRun := flag.Bool("dry-run", false, "Сухой прогон (не удалять файлы на диске)")
	yes := flag.Bool("y", false, "Автоматическое подтверждение удаления без запроса")
	flag.Parse()

	targetPath := "."
	if flag.NArg() > 0 {
		targetPath = flag.Arg(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("🔍 Сканирование %s (воркеры: %d)...\n", targetPath, *workers)

	sc := scanner.New(scanner.Options{
		Workers: *workers,
	})

	root, stats, err := sc.Scan(ctx, targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сканирования: %v\n", err)
		os.Exit(1)
	}

	cacheBytes, cacheDirs := cleaner.TagCaches(root)
	stats.CacheBytes = cacheBytes
	stats.CacheDirs = cacheDirs

	if *interactive {
		p := tea.NewProgram(tui.NewModel(root), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка TUI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *cleanCache {
		caches := cleaner.CollectCaches(root)
		if len(caches) == 0 {
			fmt.Println("✨ Директорий кэша не обнаружено!")
			return
		}

		fmt.Printf("\n🧹 Найдено %d директорий кэша суммарным объемом %s:\n", len(caches), ui.FormatBytes(cacheBytes))
		for _, c := range caches {
			fmt.Printf("  • %-30s [%s] (%s, %s файлов)\n", c.Path, c.CacheKind, ui.FormatBytes(c.Size), ui.FormatNumber(c.ItemCount))
		}

		if *dryRun {
			fmt.Printf("\n[DRY-RUN] Файлы не удалены. Запустите без --dry-run для очистки.\n")
			return
		}

		if !*yes {
			fmt.Printf("\nУдалить эти директории безвозвратно? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("Отмена очистки.")
				return
			}
		}

		var reclaimed int64
		for _, c := range caches {
			if err := cleaner.DeleteSafely(c.Path, false); err != nil {
				fmt.Fprintf(os.Stderr, "Ошибка удаления %s: %v\n", c.Path, err)
			} else {
				reclaimed += c.Size
			}
		}
		fmt.Printf("✅ Успешно очищено: %s!\n", ui.FormatBytes(reclaimed))
		return
	}

	// Default CLI output mode (dust style)
	treeOutput := ui.RenderCLITree(root, *maxDepth, 20)
	fmt.Print(treeOutput)
	fmt.Println(ui.RenderSummary(stats))
}
