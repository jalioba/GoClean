package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"goclean/internal/model"
)

var (
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
	return RenderCLITreeWithLimit(root, maxDepth, barWidth, 0)
}

func RenderCLITreeWithLimit(root *model.Node, maxDepth int, barWidth int, maxItems int) string {
	if barWidth <= 0 {
		barWidth = 15
	}
	root.SortChildrenDepth(model.SortBySize, maxDepth)

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

			children := n.Children
			limit := len(children)
			truncated := false
			if maxItems > 0 && limit > maxItems {
				limit = maxItems
				truncated = true
			}

			for i := 0; i < limit; i++ {
				childIsLast := !truncated && (i == limit-1)
				printNode(children[i], depth+1, newPrefix, childIsLast)
			}

			if truncated {
				var remSize int64
				var remItems int64
				for i := limit; i < len(children); i++ {
					remSize += children[i].Size
					remItems += children[i].ItemCount
				}
				remCount := len(children) - limit
				remRatio := float64(0)
				if root.Size > 0 {
					remRatio = float64(remSize) / float64(root.Size)
				}
				remBar := RenderBar(remRatio, barWidth, false)
				remLine := fmt.Sprintf("%-10s  [%s] %s  %6s  %s└── %s",
					FormatBytes(remSize),
					remBar,
					FormatPercent(remRatio),
					FormatNumber(remItems),
					newPrefix,
					dimStyle.Render(fmt.Sprintf("... and %d more items", remCount)),
				)
				sb.WriteString(remLine)
				sb.WriteString("\n")
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

// RenderMiniTree builds a compact tree preview of a node and its top children by size,
// used in deletion confirmation dialogs.
func RenderMiniTree(root *model.Node, maxDepth int, maxChildren int) string {
	if root == nil {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxChildren <= 0 {
		maxChildren = 5
	}
	root.SortChildrenDepth(model.SortBySize, maxDepth)

	var sb strings.Builder
	rootIcon := "📁 "
	if root.Type != model.NodeTypeDir {
		rootIcon = "📄 "
	}
	sb.WriteString(fmt.Sprintf("%s%s (%s, %s items)\n",
		rootIcon, root.Name, FormatBytes(root.Size), FormatNumber(root.ItemCount)))

	var walk func(n *model.Node, depth int, prefix string)
	walk = func(n *model.Node, depth int, prefix string) {
		if n.Type != model.NodeTypeDir || depth > maxDepth || len(n.Children) == 0 {
			return
		}
		limit := len(n.Children)
		truncated := false
		if limit > maxChildren {
			limit = maxChildren
			truncated = true
		}

		for i := 0; i < limit; i++ {
			c := n.Children[i]
			isLast := !truncated && (i == limit-1)
			branch := prefix + "├── "
			nextPrefix := prefix + "│   "
			if isLast {
				branch = prefix + "└── "
				nextPrefix = prefix + "    "
			}
			icon := "📁 "
			if c.Type != model.NodeTypeDir {
				icon = "📄 "
			}
			sb.WriteString(fmt.Sprintf("%s%s%s (%s)\n", branch, icon, c.Name, FormatBytes(c.Size)))
			if c.Type == model.NodeTypeDir && depth < maxDepth {
				walk(c, depth+1, nextPrefix)
			}
		}

		if truncated {
			var remSize int64
			for i := limit; i < len(n.Children); i++ {
				remSize += n.Children[i].Size
			}
			remCount := len(n.Children) - limit
			sb.WriteString(fmt.Sprintf("%s└── %s\n",
				prefix,
				dimStyle.Render(fmt.Sprintf("... and %d more items (%s)", remCount, FormatBytes(remSize))),
			))
		}
	}

	walk(root, 1, "")
	return strings.TrimRight(sb.String(), "\n")
}

// RenderCacheTreePreview renders a mini-tree of cache directories and their largest internal files/subfolders.
func RenderCacheTreePreview(baseRoot *model.Node, caches []*model.Node, maxCaches int, maxChildren int) string {
	if len(caches) == 0 {
		return ""
	}
	if maxChildren <= 0 {
		maxChildren = 3
	}

	var basePath string
	if baseRoot != nil {
		basePath = baseRoot.FullPath()
	}

	limitCaches := len(caches)
	truncatedCaches := false
	if maxCaches > 0 && limitCaches > maxCaches {
		limitCaches = maxCaches
		truncatedCaches = true
	}

	var sb strings.Builder
	for idx := 0; idx < limitCaches; idx++ {
		c := caches[idx]
		c.SortChildrenDepth(model.SortBySize, 1)

		isLastCache := !truncatedCaches && (idx == limitCaches-1)
		cacheBranch := "├── "
		childPrefix := "│   "
		if isLastCache {
			cacheBranch = "└── "
			childPrefix = "    "
		}

		displayPath := c.FullPath()
		if basePath != "" && strings.HasPrefix(displayPath, basePath) && displayPath != basePath {
			rel := strings.TrimPrefix(displayPath, basePath)
			rel = strings.TrimLeft(rel, "/\\")
			if rel != "" {
				displayPath = rel
			}
		}

		sb.WriteString(fmt.Sprintf("%s📁 %s [%s] (%s, %s items)\n",
			cacheBranch,
			displayPath,
			c.CacheKind,
			FormatBytes(c.Size),
			FormatNumber(c.ItemCount),
		))

		limit := len(c.Children)
		truncated := false
		if limit > maxChildren {
			limit = maxChildren
			truncated = true
		}
		for i := 0; i < limit; i++ {
			ch := c.Children[i]
			isLast := !truncated && (i == limit-1)
			branch := childPrefix + "├── "
			if isLast {
				branch = childPrefix + "└── "
			}
			icon := "📁 "
			if ch.Type != model.NodeTypeDir {
				icon = "📄 "
			}
			sb.WriteString(fmt.Sprintf("%s%s%s (%s)\n", branch, icon, ch.Name, FormatBytes(ch.Size)))
		}
		if truncated {
			var remSize int64
			for i := limit; i < len(c.Children); i++ {
				remSize += c.Children[i].Size
			}
			remCount := len(c.Children) - limit
			sb.WriteString(fmt.Sprintf("%s└── %s\n",
				childPrefix,
				dimStyle.Render(fmt.Sprintf("... and %d more items (%s)", remCount, FormatBytes(remSize))),
			))
		}
	}

	if truncatedCaches {
		var remSize int64
		for i := limitCaches; i < len(caches); i++ {
			remSize += caches[i].Size
		}
		remCount := len(caches) - limitCaches
		sb.WriteString(fmt.Sprintf("└── %s\n",
			dimStyle.Render(fmt.Sprintf("... and %d more cache directories (%s)", remCount, FormatBytes(remSize))),
		))
	}

	return strings.TrimRight(sb.String(), "\n")
}
