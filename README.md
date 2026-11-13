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
  * Smooth navigation (`↑`/`↓`/`Enter`/`Esc` or `j`/`k`/`l`/`h`) with parent cursor position memory when going back.
  * Instant lazy sorting of the active directory by size, name, or item count (`s`).
  * Live UTF-8 filter/search by filename (`/`).
  * Non-blocking asynchronous deletion (`d`) and batch cache cleaning (`c`) with centered confirmation modals and a **mini-tree preview** of the files and folders to be removed.
* **Smart & Safe Cache Cleaner**:
  * **Unambiguous cache rules**: Node.js (`node_modules`, `.next`, `.nuxt`, `.turbo`, `.pnpm-store`), Python (`__pycache__`, `.pytest_cache`, `.mypy_cache`, `.venv`, `venv`), Java/Kotlin (`.gradle`), general (`.cache`, `.ds_store`, `thumbs.db`).
  * **Context-aware rules**: Rust `target` (matched only next to `Cargo.toml`), Maven `target` (next to `pom.xml`), .NET `bin`/`obj` (next to `.csproj`/`.fsproj`/`.vbproj`) — avoiding accidental deletion of generic `build`, `dist`, or `bin` folders.
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
| `Enter` / `→` / `l` | Enter selected directory |
| `Esc` / `←` / `h` | Go up to parent directory (restores cursor position) |
| `s` | Cycle sort criteria (by size / by name / by items count) |
| `/` | Live filter / search by name |
| `c` | Open dialog with mini-tree preview to clean all detected caches in current folder |
| `d` | Open dialog with mini-tree preview to delete selected directory or file |
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
goclean --clean-cache -y
```

---

## Command-Line Flags

| Flag | Description | Default |
|---|---|---|
| `-i` | Launch interactive full-screen TUI mode (like ncdu) | `false` |
| `-w <int>` | Number of concurrent scanner workers | `NumCPU * 2` |
| `-d <int>` | Maximum directory tree display depth | `2` |
| `-n <int>` | Maximum items to display per directory in CLI tree (`0` = all) | `15` |
| `--clean-cache` | Scan for and remove project caches (`node_modules`, `target`, etc.) | `false` |
| `--dry-run` | Simulate deletion without removing files from disk | `false` |
| `-y` | Automatically confirm deletion prompts | `false` |

---
