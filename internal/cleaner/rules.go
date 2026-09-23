package cleaner

import "strings"

type CacheRule struct {
	Name string
	Kind string
}

var DefaultCacheRules = map[string]string{
	"node_modules":    "npm/node",
	".next":           "nextjs",
	".nuxt":           "nuxtjs",
	".turbo":          "turborepo",
	".pnpm-store":     "pnpm",
	"target":          "rust/cargo",
	"__pycache__":     "python",
	".pytest_cache":   "pytest",
	".mypy_cache":     "mypy",
	".venv":           "python-venv",
	"venv":            "python-venv",
	".gradle":         "gradle",
	"build":           "build-dir",
	"dist":            "dist-dir",
	".cache":          "general-cache",
	"bin":             "bin-cache",
	"obj":             "dotnet-obj",
	".ds_store":       "system",
	"thumbs.db":       "system",
}

func MatchCacheRule(dirName string) (bool, string) {
	lower := strings.ToLower(dirName)
	if kind, ok := DefaultCacheRules[lower]; ok {
		return true, kind
	}
	return false, ""
}
