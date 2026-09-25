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

	fmt.Printf("🔍 Scanning %s (workers: %d)...\n", targetPath, *workers)

	sc := scanner.New(scanner.Options{
		Workers: *workers,
	})

	root, stats, err := sc.Scan(ctx, targetPath)
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

		fmt.Printf("\n🧹 Found %d cache directories totaling %s:\n", len(caches), ui.FormatBytes(cacheBytes))
		for _, c := range caches {
			fmt.Printf("  • %-30s [%s] (%s, %s items)\n", c.Path, c.CacheKind, ui.FormatBytes(c.Size), ui.FormatNumber(c.ItemCount))
		}

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

		var reclaimed int64
		for _, c := range caches {
			if err := cleaner.DeleteSafely(c.Path, false); err != nil {
				fmt.Fprintf(os.Stderr, "Delete error for %s: %v\n", c.Path, err)
			} else {
				reclaimed += c.Size
			}
		}
		fmt.Printf("✅ Successfully reclaimed: %s!\n", ui.FormatBytes(reclaimed))
		return
	}

	// Default CLI output mode (dust style)
	treeOutput := ui.RenderCLITree(root, *maxDepth, 20)
	fmt.Print(treeOutput)
	fmt.Println(ui.RenderSummary(stats))
}
