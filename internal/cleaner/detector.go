package cleaner

import (
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
