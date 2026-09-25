package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EEEEEE")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#555555"))

	cacheTagStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5252")).
			Background(lipgloss.Color("#331111")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

func RenderCLITree(root *model.Node, maxDepth int, barWidth int) string {
	if barWidth <= 0 {
		barWidth = 15
	}
	root.SortChildren(model.SortBySize)

	var sb strings.Builder

	header := fmt.Sprintf("%-10s  %-*s  %6s  %s", "Size", barWidth+2, "% Usage", "Items", "Path")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	var printNode func(n *model.Node, depth int, prefix string, isLast bool)
	printNode = func(n *model.Node, depth int, prefix string, isLast bool) {
		if depth > maxDepth {
			return
		}

		var branch string
		if depth > 0 {
			if isLast {
				branch = prefix + "└── "
			} else {
				branch = prefix + "├── "
			}
		}

		ratio := float64(0)
		if root.Size > 0 {
			ratio = float64(n.Size) / float64(root.Size)
		}

		bar := RenderBar(ratio, barWidth, true)
		pctStr := FormatPercent(ratio)
		sizeStr := FormatBytes(n.Size)
		itemsStr := FormatNumber(n.ItemCount)

		var icon string
		if n.Type == model.NodeTypeDir {
			icon = "📁 "
		} else {
			icon = "📄 "
		}

		line := fmt.Sprintf("%-10s  [%s] %s  %6s  %s%s%s",
			sizeStr,
			bar,
			pctStr,
			itemsStr,
			branch,
			icon,
			n.Name,
		)

		if n.IsCache {
			line += " " + cacheTagStyle.Render(fmt.Sprintf("[CACHE: %s]", n.CacheKind))
		}

		sb.WriteString(line)
		sb.WriteString("\n")

		if n.Type == model.NodeTypeDir && depth < maxDepth {
			var newPrefix string
			if depth > 0 {
				if isLast {
					newPrefix = prefix + "    "
				} else {
					newPrefix = prefix + "│   "
				}
			}
			for i, child := range n.Children {
				printNode(child, depth+1, newPrefix, i == len(n.Children)-1)
			}
		}
	}

	printNode(root, 0, "", true)
	return sb.String()
}

func RenderSummary(stats *model.ScanStats) string {
	sep := dimStyle.Render(" | ")
	summary := fmt.Sprintf("⚡ Scan time: %v%s📁 Dirs: %s%s📄 Files: %s%s💾 Total: %s",
		stats.Duration.Round(10*time.Millisecond),
		sep,
		FormatNumber(stats.TotalDirs),
		sep,
		FormatNumber(stats.TotalFiles),
		sep,
		FormatBytes(stats.TotalBytes),
	)

	if stats.CacheDirs > 0 {
		summary += fmt.Sprintf("\n💡 Found caches to clean: %s (%s)",
			FormatBytes(stats.CacheBytes),
			FormatNumber(stats.CacheDirs)+" dirs",
		)
	}

	return summary
}
