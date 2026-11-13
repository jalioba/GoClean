package cleaner

import (
	"path/filepath"
	"strings"

	"goclean/internal/model"
)

func DetectCache(dirName string) (bool, string) {
	return MatchCacheRule(dirName)
}

func DetectCacheNode(n *model.Node) (bool, string) {
	return MatchCacheNode(n)
}

// TagCaches walks the tree and marks cache directories without double-counting
// nested caches (e.g., node_modules inside node_modules).
func TagCaches(root *model.Node) (totalCacheBytes int64, totalCacheDirs int64) {
	if root == nil {
		return 0, 0
	}
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type != model.NodeTypeDir {
			return
		}
		if isCache, kind := DetectCacheNode(n); isCache {
			n.IsCache = true
			n.CacheKind = kind
			totalCacheBytes += n.Size
			totalCacheDirs++
			// Do not recurse into subdirectories of a cache directory
			// so nested caches are not double-counted.
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return totalCacheBytes, totalCacheDirs
}

func CollectCaches(root *model.Node) []*model.Node {
	if root == nil {
		return nil
	}
	var caches []*model.Node
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type == model.NodeTypeDir {
			if n.IsCache {
				caches = append(caches, n)
				// Don't recurse into subdirectories of a cache directory
				return
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
	}
	walk(root)
	return caches
}

// FilterCachesByKind filters cache nodes by a comma-separated list of kinds or directory names
// (e.g. "npm,rust,python" or "node_modules,target").
func FilterCachesByKind(caches []*model.Node, filterCSV string) ([]*model.Node, int64) {
	rawTokens := strings.Split(filterCSV, ",")
	tokens := make([]string, 0, len(rawTokens))
	for _, t := range rawTokens {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) == 0 {
		var total int64
		for _, c := range caches {
			total += c.Size
		}
		return caches, total
	}

	var filtered []*model.Node
	var totalBytes int64
	for _, c := range caches {
		kindLower := strings.ToLower(c.CacheKind)
		nameLower := strings.ToLower(c.Name)
		matched := false
		for _, tok := range tokens {
			if kindLower == tok || strings.Contains(kindLower, tok) || nameLower == tok {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, c)
			totalBytes += c.Size
		}
	}
	return filtered, totalBytes
}

// CollectFilesByExt collects all files matching a comma-separated list of extensions
// (e.g. ".log,.tmp,bak").
func CollectFilesByExt(root *model.Node, extCSV string) ([]*model.Node, int64) {
	if root == nil {
		return nil, 0
	}
	rawExts := strings.Split(extCSV, ",")
	extSet := make(map[string]struct{}, len(rawExts))
	for _, e := range rawExts {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[e] = struct{}{}
	}
	if len(extSet) == 0 {
		return nil, 0
	}

	var files []*model.Node
	var totalBytes int64
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type == model.NodeTypeFile {
			ext := strings.ToLower(filepath.Ext(n.Name))
			if _, ok := extSet[ext]; ok {
				files = append(files, n)
				totalBytes += n.Size
			}
			return
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(root)
	return files, totalBytes
}
