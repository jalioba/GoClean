# GoClean

**GoClean** is a blazing-fast, concurrent disk space analyzer and developer cache cleaner (`ncdu` + `dust` built on Go goroutines).

It scales filesystem metadata traversal across all CPU cores using a low-contention ring-buffer worker pool, visualizes disk usage with pseudographic percentage bars, and provides context-aware detection and parallel cleaning of bloated developer caches (`node_modules`, `target`, `.cache`, `.next`, `__pycache__`, etc.).

---

## Key Features

* **Concurrent Goroutine Scanner**:
  * Traverses directory trees with a dynamic ring-buffer task queue, batched directory pushes, and symlink/bind-mount loop detection (`(Dev, Ino)` on Unix).
  * Minimal memory footprint & GC pressure via compact tree nodes and on-demand path reconstruction (`FullPath()`).
  * Live progress spinner with real-time scanned directory and file counters.
  * Configurable number of concurrent workers (defaults to `NumCPU * 2`).
* **`dust`-Style CLI Mode**:
  * Hierarchical directory tree rendering with branches (`├──`, `└──`).
  * Unicode usage bars `[████████░░░░] 68%` with colored thresholds (green → yellow → orange → red).
  * Configurable per-directory item limit (`-n`, defaults to `15`) that collapses long tails into `└── ... and X more items`.
  * Direct badge highlights for trash/cache targets: `[CACHE: npm/node]`, `[CACHE: rust/cargo]`, etc.
* **Full-Screen Interactive TUI (`-i`)**:
  * Powered by `charmbracelet/bubbletea` and `lipgloss`.
  * Smooth navigation (`↑`/`↓`/`PgUp`/`PgDn`/`Home`/`End`/`Enter`/`Esc` or `j`/`k`/`g`/`G`/`l`/`h`) with parent cursor position memory when going back.
  * **Multi-Select (`Space`)**: mark multiple files or folders `[✓]` with live total size counter and delete them in a single batch (`d`).
  * **Interactive Cache Picker (`c`)**: toggle `[✓]` / `[ ]` checkboxes on individual detected caches (`Space` or `a` for all/none) with a live mini-tree preview of the highlighted cache before cleaning.
  * **Built-in Analytics Views**: press `f` for **Top Largest Files** or `t` for **File Extension Breakdown** right inside the TUI.
  * Instant lazy sorting of the active directory by size, name, or item count (`s`) and live UTF-8 filter/search (`/`).
* **Smart & Safe Cache Cleaner (25+ Ecosystems)**:
  * **Unambiguous cache rules**: Node.js/Web (`node_modules`, `.next`, `.nuxt`, `.turbo`, `.pnpm-store`, `.vite`, `.parcel-cache`, `.angular`, `.svelte-kit`, `.astro`, `.nyc_output`), Python (`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `.tox`, `.nox`, `.venv`, `venv`), JVM (`.gradle`), Dart/Flutter (`.dart_tool`), Infrastructure/Systems (`.terraform`, `.zig-cache`, `zig-cache`, `zig-out`, `CMakeFiles`, `DerivedData`), general (`.cache`, `.ds_store`, `thumbs.db`).
  * **Context-aware rules**: Rust `target` (next to `Cargo.toml`), Maven `target` (next to `pom.xml`), Scala `target` (next to `build.sbt`), .NET `bin`/`obj` (next to `.csproj`/`.fsproj`/`.vbproj`), Flutter/Gradle/CMake `build` (next to `pubspec.yaml`, `build.gradle`, or `CMakeLists.txt`), and iOS `Pods` (next to `Podfile`).
  * Accurate size calculation without double-counting nested caches (e.g., `node_modules` inside `node_modules`).
  * Parallel multi-worker cache deletion for maximum cleanup speed.
  * Dry-run mode (`--dry-run`) and protected system/root path safeguards (`C:\`, `/`, `/usr`, `/bin`, `Windows`, `Program Files`, `$HOME`).

---

## Installation

### Option 1: npm / bun / npx (Cross-Platform)

Run instantly without installation:
```bash
npx @jalioba/goclean
# or with bun:
bunx @jalioba/goclean
```

Or install globally:
```bash
npm install -g @jalioba/goclean
# or with bun:
bun add -g @jalioba/goclean
```

---

### Option 2: Homebrew (macOS / Linux)

```bash
brew install jalioba/tap/goclean
```

---

### Option 3: Shell Installer (Linux & macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/jalioba/GoClean/main/install.sh | bash
```

---

### Option 4: PowerShell Installer (Windows)

```powershell
irm https://raw.githubusercontent.com/jalioba/GoClean/main/install.ps1 | iex
```

---

### Option 5: Go Install (Go 1.22+)

```bash
go install github.com/jalioba/GoClean/cmd/goclean@latest
```

---

### Option 6: Build from Source

```bash
git clone https://github.com/jalioba/GoClean.git
cd GoClean
go build -o goclean.exe ./cmd/goclean
```

---

## Usage Examples

### 1. Fast Disk Analysis (`dust` mode)
```bash
goclean
```
Or specify path, depth limit, and max items per folder:
```bash
goclean -d 2 -n 10 C:\Users\user\Projects
```

Sample output:
```text
Size        % Usage                  Items  Path
────────────────────────────────────────────────
14.82 GB    [████████████████████] 100.0%  142,503  📁 Projects
7.41 GB     [██████████░░░░░░░░░░]  50.0%   78,410  ├── 📁 web-app
4.20 GB     [█████░░░░░░░░░░░░░░░]  28.3%   64,120  │   ├── 📁 node_modules  [CACHE: npm/node]
2.10 GB     [███░░░░░░░░░░░░░░░░░]  14.2%   12,300  │   ├── 📁 .next  [CACHE: nextjs]
1.11 GB     [█▌░░░░░░░░░░░░░░░░░░]   7.5%    1,990  │   └── ... and 12 more items
5.12 GB     [███████░░░░░░░░░░░░░]  34.5%   42,100  └── 📁 rust-service
3.80 GB     [█████░░░░░░░░░░░░░░░]  25.6%   38,400      └── 📁 target  [CACHE: rust/cargo]
⚡ Scan time: 0.42s | 📁 Dirs: 8,412 | 📄 Files: 142,503 | 💾 Total: 14.82 GB
💡 Found caches to clean: 10.10 GB (3 dirs)
```

### 2. Full-Screen Interactive TUI (`ncdu` mode)
```bash
goclean -i
```
Or browse a specific folder:
```bash
goclean -i C:\Users\user\Projects
```

#### TUI Keyboard Shortcuts:
| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Move cursor up / down |
| `PgUp` / `PgDn`, `Home` (`g`) / `End` (`G`) | Fast page scroll / jump to start or end |
| `Space` | Toggle `[✓]` selection on file/directory (or cache in `c` modal) |
| `Enter` / `→` / `l` | Enter selected directory |
| `Esc` / `←` / `h` | Go up to parent directory (restores cursor position) or exit view |
| `s` | Cycle sort criteria (by size / by name / by items count) |
| `f` | Toggle **Top Largest Files** view |
| `t` | Toggle **File Extension Breakdown** view |
| `/` | Live filter / search by name |
| `c` | Open interactive cache cleaner (`[✓]`/`[ ]` checkboxes + mini-tree preview) |
| `d` | Delete selected item(s) with mini-tree preview |
| `q` / `Ctrl+C` | Exit |

### 3. Cleaning Caches via CLI

**Dry-run (inspect mini-tree of cache directories and contents without deleting):**
```bash
goclean --clean-cache --dry-run
```

Sample preview output:
```text
🧹 Found 2 cache directories totaling 8.00 GB:

├── 📁 web-app/node_modules [npm/node] (4.20 GB, 64,120 items)
│   ├── 📁 typescript (180.50 MB)
│   ├── 📁 next (142.10 MB)
│   ├── 📁 @swc (98.40 MB)
│   └── ... and 312 more items (3.79 GB)
└── 📁 rust-service/target [rust/cargo] (3.80 GB, 38,400 items)
    ├── 📁 debug (3.10 GB)
    └── 📁 release (716.80 MB)
```

**Interactive clean (prompts for confirmation `[y/N]`):**
```bash
goclean --clean-cache
```

**Batch clean without prompt (useful for CI/scripts):**
```bash
# Unconditional single-flag cleanup of all detected caches:
goclean --force-clean

# Unconditional cleanup of specific cache types only (e.g., npm and Rust):
goclean --force-clean --clean-kind npm,rust

# Unconditional cleanup of specific file extensions (e.g., logs and temp files):
goclean --force-clean --clean-ext .log,.tmp,.bak
```

---

## Command-Line Flags

| Flag | Description | Default |
|---|---|---|
| `-i` | Launch interactive full-screen TUI mode (like ncdu) | `false` |
| `-w <int>` | Number of concurrent scanner workers | `NumCPU * 2` |
| `-d <int>` | Maximum directory tree display depth | `2` |
| `-n <int>` | Maximum items to display per directory in CLI tree (`0` = all) | `15` |
| `--top-files` | Display top largest individual files across the scanned tree | `false` |
| `--types` | Display disk usage grouped by file extension | `false` |
| `--clean-cache` | Scan for and remove project caches (`node_modules`, `target`, etc.) | `false` |
| `--force-clean` | Unconditionally delete detected caches (or `--clean-ext` files) without prompt | `false` |
| `--clean-kind <list>` | Comma-separated cache types or folder names to clean (`npm,rust,python`) | `""` |
| `--clean-ext <list>` | Comma-separated file extensions to delete (`.log,.tmp,.bak`) | `""` |
| `--dry-run` | Simulate deletion without removing files from disk | `false` |
| `-y` | Automatically confirm deletion prompts | `false` |

---
