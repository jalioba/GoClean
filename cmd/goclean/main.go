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

func confirmAction(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	return ans == "y" || ans == "yes"
}

func main() {
	interactive := flag.Bool("i", false, "Launch interactive TUI mode (like ncdu)")
	workers := flag.Int("w", 0, "Number of concurrent workers (default: NumCPU * 2)")
	maxDepth := flag.Int("d", 2, "Maximum directory tree display depth")
	maxItems := flag.Int("n", 15, "Maximum items to display per directory in CLI tree (0 = all)")
	topFiles := flag.Bool("top-files", false, "Display top largest individual files")
	showTypes := flag.Bool("types", false, "Display disk usage grouped by file extension")
	cleanCache := flag.Bool("clean-cache", false, "Scan for and remove development caches (node_modules, target, etc.)")
	forceClean := flag.Bool("force-clean", false, "Unconditionally delete detected caches (or --clean-ext files) without confirmation")
	cleanKind := flag.String("clean-kind", "", "Comma-separated cache types or names to delete (e.g. 'npm,rust,python')")
	cleanExt := flag.String("clean-ext", "", "Comma-separated file extensions to delete (e.g. '.log,.tmp,.bak')")
	dryRun := flag.Bool("dry-run", false, "Simulate deletion without removing files from disk")
	yes := flag.Bool("y", false, "Automatic yes to confirmation prompts")
	flag.Parse()

	if *forceClean {
		*yes = true
		if *cleanExt == "" {
			*cleanCache = true
		}
	}
	if *cleanKind != "" {
		*cleanCache = true
	}

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

	if *cleanExt != "" {
		files, extBytes := cleaner.CollectFilesByExt(root, *cleanExt)
		if len(files) == 0 {
			fmt.Printf("✨ No files matching extensions [%s] found!\n", *cleanExt)
			return
		}

		fmt.Printf("\n🧹 Found %d files matching [%s] totaling %s:\n", len(files), *cleanExt, ui.FormatBytes(extBytes))
		previewLimit := len(files)
		if previewLimit > 20 {
			previewLimit = 20
		}
		for i := 0; i < previewLimit; i++ {
			f := files[i]
			fmt.Printf("  • 📄 %s (%s)\n", f.FullPath(), ui.FormatBytes(f.Size))
		}
		if len(files) > previewLimit {
			fmt.Printf("  └── ... and %d more files\n", len(files)-previewLimit)
		}

		if *dryRun {
			fmt.Printf("\n[DRY-RUN] No files were deleted. Run without --dry-run to clean.\n")
			return
		}

		if !*yes && !confirmAction("\nPermanently delete these files? [y/N]: ") {
			fmt.Println("Cleaning cancelled.")
			return
		}

		reclaimed, deleted, errs := cleaner.DeleteNodesParallel(files, sc.Workers(), false)
		for _, delErr := range errs {
			fmt.Fprintf(os.Stderr, "Delete error: %v\n", delErr)
		}
		fmt.Printf("✅ Deleted %d files, reclaimed: %s!\n", len(deleted), ui.FormatBytes(reclaimed))
		if !*cleanCache {
			return
		}
	}

	if *cleanCache {
		caches := cleaner.CollectCaches(root)
		targetBytes := cacheBytes
		if *cleanKind != "" {
			caches, targetBytes = cleaner.FilterCachesByKind(caches, *cleanKind)
		}
		if len(caches) == 0 {
			if *cleanKind != "" {
				fmt.Printf("✨ No cache directories matching [%s] found!\n", *cleanKind)
			} else {
				fmt.Println("✨ No cache directories found!")
			}
			return
		}

		fmt.Printf("\n🧹 Found %d cache directories totaling %s:\n\n", len(caches), ui.FormatBytes(targetBytes))
		fmt.Println(ui.RenderCacheTreePreview(root, caches, 0, 4))

		if *dryRun {
			fmt.Printf("\n[DRY-RUN] No files were deleted. Run without --dry-run to clean.\n")
			return
		}

		if !*yes && !confirmAction("\nPermanently delete these directories? [y/N]: ") {
			fmt.Println("Cleaning cancelled.")
			return
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
