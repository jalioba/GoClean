package cleaner

import (
	"goclean/internal/model"
)

func DetectCache(dirName string) (bool, string) {
	return MatchCacheRule(dirName)
}

func TagCaches(root *model.Node) (totalCacheBytes int64, totalCacheDirs int64) {
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if n.Type == model.NodeTypeDir {
			if isCache, kind := DetectCache(n.Name); isCache {
				n.IsCache = true
				n.CacheKind = kind
				totalCacheBytes += n.Size
				totalCacheDirs++
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
	}
	walk(root)
	return totalCacheBytes, totalCacheDirs
}

func CollectCaches(root *model.Node) []*model.Node {
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
