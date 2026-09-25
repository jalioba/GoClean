# 🚀 GoClean

**GoClean** is a blazing-fast, concurrent disk space analyzer and developer cache cleaner (`ncdu` + `dust` built on Go goroutines).

It scales filesystem metadata traversal across all CPU cores using an adaptive worker pool, visualizes disk usage with pseudographic percentage bars, and provides instant detection and cleaning of bloated developer caches (`node_modules`, `target`, `.cache`, `.next`, `__pycache__`, etc.).

---

## ✨ Key Features

* ⚡ **Concurrent Goroutine Scanner**:
  * Traverses directory trees with a dynamic, starvation-free task queue and symlink/junction loop detection.
  * Significantly faster than single-threaded `filepath.WalkDir` on multi-core CPUs and fast NVMe/SSDs.
  * Configurable number of concurrent workers (defaults to `NumCPU * 2`).
* 📊 **`dust`-Style CLI Mode**:
  * Hierarchical directory tree rendering with branches (`├──`, `└──`).
  * Unicode usage bars `[████████░░░░] 68%` with colored thresholds (green → yellow → orange → red).
  * Direct badge highlights for trash/cache targets: `[CACHE: npm/node]`, `[CACHE: rust/cargo]`, etc.
* 🖥️ **Full-Screen Interactive TUI (`-i`)**:
  * Powered by `charmbracelet/bubbletea` and `lipgloss`.
  * Smooth arrow key navigation (`↑`/`↓`/`Enter`/`Esc` or `j`/`k`/`l`/`h`).
  * Instant sorting by size, name, or item count (`s`).
  * Live filter/search by filename (`/`).
  * One-key cache cleaning (`c`) with confirmation dialog.
  * Safe file/folder deletion (`d`) with protection prompt.
* 🧹 **Smart & Safe Cache Cleaner**:
  * Automatic detection rules: Node.js (`node_modules`, `.next`, `.nuxt`, `.turbo`, `.pnpm-store`), Rust (`target`), Python (`__pycache__`, `.pytest_cache`, `.venv`), Java/Kotlin (`.gradle`, `build`), Go/.NET/C++ (`.cache`, `bin`, `obj`), etc.
  * Dry-run mode (`--dry-run`): inspect what would be deleted and the reclaimable size without touching disk.
  * Protected root paths safeguard (`C:\`, `/`, `Windows`, `Program Files`, `$HOME` entirely).

---

## 🛠️ Installation & Build

Requires **Go 1.22+**:

```bash
# Clone the repository
git clone https://github.com/your-username/GoClean.git
cd GoClean

# Build executable
go build -o goclean.exe ./cmd/goclean
```

---

## 📖 Usage Examples

### 1. Fast Disk Analysis (`dust` mode)
```bash
goclean
```
Or specify path and depth limit:
```bash
goclean -d 2 C:\Users\user\Projects
```

Sample output:
```text
🔍 Scanning C:\Users\user\Projects (workers: 16)...
Size        % Usage                  Items  Path
────────────────────────────────────────────────
14.82 GB    [████████████████████] 100.0%  142,503  📁 Projects
7.41 GB     [██████████░░░░░░░░░░]  50.0%   78,410  ├── 📁 web-app
4.20 GB     [█████░░░░░░░░░░░░░░░]  28.3%   64,120  │   ├── 📁 node_modules  [CACHE: npm/node]
2.10 GB     [███░░░░░░░░░░░░░░░░░]  14.2%   12,300  │   └── 📁 .next  [CACHE: nextjs]
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
| `Esc` / `←` / `h` | Go up to parent directory |
| `s` | Cycle sort criteria (by size / by name / by items count) |
| `/` | Live filter / search by name |
| `c` | Open dialog to clean all detected caches in current folder |
| `d` | Delete selected directory or file |
| `q` / `Ctrl+C` | Exit |

### 3. Cleaning Caches via CLI

**Dry-run (simulate and calculate reclaimable space without deleting):**
```bash
goclean --clean-cache --dry-run
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

## ⚙️ Command-Line Flags

| Flag | Description | Default |
|---|---|---|
| `-i` | Launch interactive full-screen TUI mode (like ncdu) | `false` |
| `-w <int>` | Number of concurrent scanner workers | `NumCPU * 2` |
| `-d <int>` | Maximum directory tree display depth | `2` |
| `--clean-cache` | Scan for and remove project caches (`node_modules`, `target`, etc.) | `false` |
| `--dry-run` | Simulate deletion without removing files from disk | `false` |
| `-y` | Automatically confirm deletion prompts | `false` |

---

## 🧪 Running Tests

All modules are built with Test-Driven Development and verified with the Go race detector:

```bash
go test -v -race ./...
```
