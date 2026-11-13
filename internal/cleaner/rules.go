package cleaner

import (
	"strings"

	"goclean/internal/model"
)

// DefaultCacheRules contains directory/file names that are unambiguously
// regenerable caches or build artifacts regardless of parent context.
var DefaultCacheRules = map[string]string{
	"node_modules":  "npm/node",
	".next":         "nextjs",
	".nuxt":         "nuxtjs",
	".turbo":        "turborepo",
	".pnpm-store":   "pnpm",
	".vite":         "vite",
	".parcel-cache": "parcel",
	".angular":      "angular",
	".svelte-kit":   "sveltekit",
	".astro":        "astro",
	".nyc_output":   "js-coverage",
	"__pycache__":   "python",
	".pytest_cache": "pytest",
	".mypy_cache":   "mypy",
	".ruff_cache":   "ruff",
	".tox":          "python-tox",
	".nox":          "python-nox",
	".venv":         "python-venv",
	"venv":          "python-venv",
	".gradle":       "gradle",
	".dart_tool":    "dart/flutter",
	".terraform":    "terraform",
	".zig-cache":    "zig",
	"zig-cache":     "zig",
	"zig-out":       "zig",
	"cmakefiles":    "cmake",
	"deriveddata":   "xcode",
	".cache":        "general-cache",
	".ds_store":     "system",
	"thumbs.db":     "system",
}

// MatchCacheRule checks unambiguous cache directory names.
func MatchCacheRule(dirName string) (bool, string) {
	lower := strings.ToLower(dirName)
	if kind, ok := DefaultCacheRules[lower]; ok {
		return true, kind
	}
	return false, ""
}

// MatchCacheNode checks both unambiguous cache rules and context-aware rules
// (e.g., "target" next to Cargo.toml/pom.xml, "bin"/"obj" next to .NET project files,
// "build" next to pubspec.yaml/build.gradle/CMakeLists.txt, "Pods" next to Podfile).
func MatchCacheNode(n *model.Node) (bool, string) {
	if n == nil {
		return false, ""
	}
	lower := strings.ToLower(n.Name)
	if kind, ok := DefaultCacheRules[lower]; ok {
		return true, kind
	}

	if n.Parent == nil {
		return false, ""
	}

	switch lower {
	case "target":
		if hasSibling(n.Parent, "Cargo.toml") {
			return true, "rust/cargo"
		}
		if hasSibling(n.Parent, "pom.xml") {
			return true, "java/maven"
		}
		if hasSibling(n.Parent, "build.sbt") {
			return true, "scala/sbt"
		}
	case "bin", "obj":
		if hasSiblingSuffix(n.Parent, ".csproj", ".fsproj", ".vbproj") {
			return true, "dotnet-" + lower
		}
	case "build":
		if hasSibling(n.Parent, "pubspec.yaml") {
			return true, "flutter-build"
		}
		if hasSibling(n.Parent, "build.gradle") || hasSibling(n.Parent, "build.gradle.kts") {
			return true, "gradle-build"
		}
		if hasSibling(n.Parent, "CMakeLists.txt") {
			return true, "cmake-build"
		}
	case "pods":
		if hasSibling(n.Parent, "Podfile") {
			return true, "cocoapods"
		}
	}

	return false, ""
}

func hasSibling(parent *model.Node, name string) bool {
	for _, c := range parent.Children {
		if c.Type == model.NodeTypeFile && strings.EqualFold(c.Name, name) {
			return true
		}
	}
	return false
}

func hasSiblingSuffix(parent *model.Node, suffixes ...string) bool {
	for _, c := range parent.Children {
		if c.Type != model.NodeTypeFile {
			continue
		}
		lower := strings.ToLower(c.Name)
		for _, s := range suffixes {
			if strings.HasSuffix(lower, s) {
				return true
			}
		}
	}
	return false
}
