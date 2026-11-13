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
	interactive := flag.Bool("i", false, "Launch interactive TUI mode (like ncdu)")
	workers := flag.Int("w", 0, "Number of concurrent workers (default: NumCPU * 2)")
	maxDepth := flag.Int("d", 2, "Maximum directory tree display depth")
	maxItems := flag.Int("n", 15, "Maximum items to display per directory in CLI tree (0 = all)")
	topFiles := flag.Bool("top-files", false, "Display top largest individual files")
	showTypes := flag.Bool("types", false, "Display disk usage grouped by file extension")
	cleanCache := flag.Bool("clean-cache", false, "Scan for and remove development caches (node_modules, target, etc.)")
	dryRun := flag.Bool("dry-run", false, "Simulate deletion without removing files from disk")
	yes := flag.Bool("y", false, "Automatic yes to confirmation prompts")
	flag.Parse()

	targetPath := "."
	if flag.NArg() > 0 {
		targetPath = flag.Arg(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinIdx := 0

	sc := scanner.New(scanner.Options{
		Workers: *workers,
		OnProgress: func(dirs, files, errs int64) {
			frame := spinnerFrames[spinIdx%len(spinnerFrames)]
			spinIdx++
			fmt.Fprintf(os.Stderr, "\r\033[K%s Scanning %s... 📁 %s dirs | 📄 %s files",
				frame, targetPath, ui.FormatNumber(dirs), ui.FormatNumber(files))
		},
	})

	fmt.Fprintf(os.Stderr, "🔍 Scanning %s (workers: %d)...", targetPath, sc.Workers())

	root, stats, err := sc.Scan(ctx, targetPath)
	fmt.Fprintf(os.Stderr, "\r\033[K")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan error: %v\n", err)
		os.Exit(1)
	}

	cacheBytes, cacheDirs := cleaner.TagCaches(root)
	stats.CacheBytes = cacheBytes
	stats.CacheDirs = cacheDirs

	if *interactive {
		p := tea.NewProgram(tui.NewModel(root), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *cleanCache {
		caches := cleaner.CollectCaches(root)
		if len(caches) == 0 {
			fmt.Println("✨ No cache directories found!")
			return
		}

		fmt.Printf("\n🧹 Found %d cache directories totaling %s:\n\n", len(caches), ui.FormatBytes(cacheBytes))
		fmt.Println(ui.RenderCacheTreePreview(root, caches, 0, 4))

		if *dryRun {
			fmt.Printf("\n[DRY-RUN] No files were deleted. Run without --dry-run to clean.\n")
			return
		}

		if !*yes {
			fmt.Printf("\nPermanently delete these directories? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("Cleaning cancelled.")
				return
			}
		}

		reclaimed, _, errs := cleaner.DeleteNodesParallel(caches, sc.Workers(), false)
		for _, delErr := range errs {
			fmt.Fprintf(os.Stderr, "Delete error: %v\n", delErr)
		}
		fmt.Printf("✅ Successfully reclaimed: %s!\n", ui.FormatBytes(reclaimed))
		return
	}

	// Default CLI output mode (dust style)
	if *topFiles {
		fmt.Print(ui.RenderTopFiles(root, *maxItems, 20))
		fmt.Println(ui.RenderSummary(stats))
		return
	}
	if *showTypes {
		fmt.Print(ui.RenderExtensionStats(root, *maxItems, 20))
		fmt.Println(ui.RenderSummary(stats))
		return
	}
	treeOutput := ui.RenderCLITreeWithLimit(root, *maxDepth, 20, *maxItems)
	fmt.Print(treeOutput)
	fmt.Println(ui.RenderSummary(stats))
}
