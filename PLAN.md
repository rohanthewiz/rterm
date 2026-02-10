# rterm — A Block-Based Graphical Terminal

## Vision

rterm is a modern graphical terminal emulator built in Go that combines the
power of the [elvish](https://github.com/elves/elvish) shell with a
Warp-inspired block-based UI. Commands and their output are treated as
discrete, navigable blocks. The command editor supports point-and-click
graphical editing. A folded view lets you see your session as a summary of
commands and exit statuses.

---

## 1. Core Architecture

### 1.1 Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Language** | Go | Matches elvish; single binary distribution; strong concurrency |
| **Shell engine** | Elvish (`pkg/eval`, `pkg/parse`) | Embedded as a Go library — not shelled out to |
| **GUI framework** | [Gio](https://gioui.org) | Immediate-mode GPU-accelerated rendering; pure Go; glyph-level text precision; full control over custom widgets |
| **Terminal emulation** | Custom VT state machine + elvish eval | Blocks capture structured output, not raw VT100 streams |
| **PTY** | `github.com/creack/pty` | Standard Go PTY library for subprocess I/O |

### 1.2 Why Gio Over Alternatives

The UI requires non-standard layout (blocks, folding, inline editors) that
retained-mode toolkits handle poorly. Key reasons for Gio:

- **Immediate-mode rendering** — total control over every frame; no fighting a
  widget tree to get block-based layout
- **GPU-accelerated vector text** — resolution-independent, fast styled text
  rendering with per-glyph advance metrics
- **Character-level click mapping** — glyph metrics allow exact cursor
  placement from mouse coordinates
- **Pure Go** — no CGO dependency for core rendering (unlike Fyne/GTK)
- **Proven feasibility** — [Gritty](https://github.com/viktomas/gritty) is a
  working terminal emulator built on Gio
- **Layout primitives** — `layout.List` for virtualized block scrolling,
  `outlay.Grid` for character grids

Trade-offs accepted:
- Small maintainer team (mitigate by pinning versions)
- API not yet at v1 (accept breaking changes between minor versions)
- More boilerplate than Fyne (worth it for the layout control we need)

### 1.3 Alternative Considered: Wails + xterm.js

A Go backend with a web frontend (xterm.js) was considered. It would yield a
polished UI faster, but introduces:
- IPC latency between PTY and renderer
- Split codebase (Go + TypeScript)
- WebView dependency
- Harder to deeply integrate with elvish's Go packages

This remains a viable fallback if Gio proves too costly for the initial
prototype.

---

## 2. Application Structure

```
rterm/
├── cmd/
│   └── rterm/
│       └── main.go              # Entry point; window creation; event loop
├── internal/
│   ├── app/
│   │   ├── app.go               # Top-level App struct; orchestrates UI + shell
│   │   └── config.go            # User configuration (themes, keybindings)
│   ├── shell/
│   │   ├── engine.go            # Wraps elvish Evaler; command execution
│   │   ├── pty.go               # PTY management for subprocesses
│   │   ├── history.go           # Command history (backed by elvish store)
│   │   └── git.go               # Git status detection for prompt
│   ├── model/
│   │   ├── block.go             # Block data model (command + output + status)
│   │   ├── session.go           # Ordered list of blocks; session state
│   │   └── buffer.go            # Circular buffer for terminal grid rows
│   ├── ui/
│   │   ├── window.go            # Gio window setup and top-level layout
│   │   ├── theme.go             # Colors, fonts, spacing
│   │   ├── editor.go            # Graphical command editor widget
│   │   ├── block_view.go        # Single block renderer (prompt + cmd + output)
│   │   ├── block_list.go        # Scrollable list of blocks
│   │   ├── folded_view.go       # Collapsed summary view
│   │   ├── history_panel.go     # Ctrl-R history search overlay
│   │   ├── prompt.go            # Prompt renderer (with git status)
│   │   └── scrollbar.go         # Custom scrollbar widget
│   └── vt/
│       ├── parser.go            # VT100/ANSI escape sequence parser
│       ├── state.go             # Terminal state machine (cursor, attrs, modes)
│       └── style.go             # ANSI color/style to Gio style mapping
├── go.mod
├── go.sum
├── PLAN.md
└── .gitignore
```

---

## 3. Feature Specifications

### 3.1 Elvish Shell Integration

rterm embeds elvish as a library, not as an external process:

```go
import (
    "src.elv.sh/pkg/eval"
    "src.elv.sh/pkg/parse"
)

// Create an evaluator
ev := eval.NewEvaler()

// Register custom builtins (e.g., for UI interaction)
ev.ExtendGlobal(eval.BuildNs().AddGoFn("rterm:notify", notifyFn))

// Evaluate user input
src := parse.Source{Name: "[interactive]", Code: userInput}
err := ev.Eval(src, eval.EvalCfg{Ports: ports})
```

**Key integration points:**

- **Parsing & highlighting** — Use `pkg/parse` to tokenize input for syntax
  highlighting in the editor. Elvish's AST
  (`Chunk → Pipeline → Form → Compound → Indexing → Primary`) maps directly
  to highlight spans.
- **Completion** — Use `pkg/edit/complete` for context-aware tab completion.
- **History** — Use elvish's `pkg/store` (BoltDB-backed) for persistent
  command history, shared across sessions via the daemon.
- **Evaluation** — Use `pkg/eval` to execute commands. Capture stdout/stderr
  via custom `Port` implementations that feed into block output buffers.

**What we build ourselves (not from elvish):**
- The graphical editor (elvish's editor is TUI-based, not GUI)
- The block model and UI
- The VT parser for subprocess output rendering

### 3.2 Graphical Command Editor

The editor is the input area where commands are typed. Unlike traditional
terminals, it behaves like a code editor:

**Point-and-click editing:**
- Click anywhere in the command text to place the cursor at that position
- Use Gio's glyph advance metrics to map pixel coordinates → character offset
- The cursor is a blinking vertical bar rendered between characters

**Text editing features:**
- Standard keybindings: Home/End, Ctrl-A/E, Ctrl-W (delete word), Ctrl-K (kill to end)
- Shift+Arrow for selection; Ctrl-Shift+Arrow for word selection
- Clipboard: Ctrl-C/V/X (copy/paste/cut)
- Multi-line editing: Shift-Enter inserts a newline; Enter submits
- Undo/redo: Ctrl-Z / Ctrl-Shift-Z
- Syntax highlighting updated on every keystroke via `pkg/parse`
- Tab completion popup positioned at the cursor

**Implementation approach:**
- Custom `EditorWidget` built on Gio's `widget.Editor` as a starting point
- Extend with syntax highlighting overlay (elvish parser provides token spans)
- Add completion popup as a `layout.Stack` overlay

### 3.3 Block Model

Each command execution produces a **Block**:

```go
type Block struct {
    ID        uint64
    Command   string        // The command text as entered
    Output    *Buffer       // Circular buffer of styled rows
    StartTime time.Time
    EndTime   time.Time
    ExitCode  int           // -1 while running
    CWD       string        // Working directory at execution time
    Collapsed bool          // Whether output is hidden in folded view
}
```

**Block lifecycle:**

1. User types command in the editor and presses Enter
2. A new Block is created with the command text; `ExitCode = -1` (running)
3. The command is evaluated via elvish; stdout/stderr feed into `Block.Output`
4. VT escape sequences in the output are parsed and converted to styled text
5. When the command completes, `ExitCode` and `EndTime` are set
6. The block is finalized and the editor reappears for the next command

**Block visual structure:**
```
┌──────────────────────────────────────────────┐
│ ~/…/go/rterm|. master+*  > ls -la       0 ✓  │  ← Prompt + command + status
├──────────────────────────────────────────────┤
│ total 42                                      │
│ drwxr-xr-x  5 user user  160 Feb 10 09:00 .  │  ← Output area
│ -rw-r--r--  1 user user 1234 Feb 10 09:00 go.│
│ ...                                           │
└──────────────────────────────────────────────┘
```

**Status indicators** (right-aligned in the header):
- `✓` (green) — exit code 0
- `✗ 1` (red) — non-zero exit code
- `⟳` (yellow, animated) — still running

**Failed blocks** have a red left border (4px) for immediate visual
distinction, similar to Warp.

### 3.4 Block List (Scrollable Session View)

The main view is a vertical list of blocks, rendered using Gio's
`layout.List` for virtualization (off-screen blocks are not rendered):

- Blocks are ordered chronologically (newest at bottom)
- The command editor is pinned at the bottom of the viewport
- Mouse scroll and scrollbar navigate the block history
- **Block-level navigation**: Ctrl-Up/Down jumps between block headers
- **Sticky header**: When scrolling through a long block, the command header
  pins to the top of the viewport for context

### 3.5 Folded View

A toggle (e.g., Ctrl-Shift-F or a toolbar button) switches to a **folded
view** that shows only block headers:

```
  ~/…/go/rterm > git status                 0 ✓   09:01
  ~/…/go/rterm > go build ./...             0 ✓   09:02
  ~/…/go/rterm > go test ./...              1 ✗   09:03
▸ ~/…/go/rterm > vim main.go                0 ✓   09:05
  ~/…/go/rterm > go test ./...              0 ✓   09:06
```

**Folded view features:**
- Each row shows: prompt, command, exit status, timestamp
- Failed commands highlighted with red text/icon
- Click a row to expand that block inline (shows output)
- Click again or press Escape to re-collapse
- Double-click a row to jump to it in the full session view
- The currently-running block (if any) always shows expanded
- Search (Ctrl-F) filters the folded list by command text

### 3.6 Command History (Ctrl-R)

Pressing Ctrl-R opens a history search overlay panel:

```
┌─ History ──────────────────────────────────┐
│ 🔍 git                                     │
├────────────────────────────────────────────┤
│   git push origin master            09:06  │
│ ▸ git commit -m "fix tests"         09:03  │
│   git add -A                        08:55  │
│   git status                        08:50  │
│   git log --oneline -10             08:42  │
│   ...                                      │
└────────────────────────────────────────────┘
```

**Features:**
- Fuzzy substring search as you type in the filter field
- Results ordered by recency (most recent first)
- Arrow keys navigate; Enter selects and inserts into the editor
- Escape closes without selecting
- Duplicate commands are deduplicated (show most recent occurrence)
- History is persisted via elvish's BoltDB store (`pkg/store`)
- Shared across rterm sessions (via elvish daemon)

**Implementation:**
- Rendered as a `ComboBox`-like overlay (text input + scrollable list)
- Positioned as a floating panel above the editor
- Uses `layout.Stack` with `layout.Stacked` for overlay compositing

### 3.7 Git-Aware Prompt

When the current directory is inside a Git repository, the prompt shows
branch, staged, and unstaged status:

```
~/…/go/rterm|. master+* >
```

**Prompt format:** `<short_path>|<git_subdir> <branch><staged><unstaged> >`

| Component | Source | Example |
|-----------|--------|---------|
| `short_path` | Abbreviate to `~/<first>/…/<last_two>` | `~/…/go/rterm` |
| `git_subdir` | Path from repo root to cwd (`.` if at root) | `.` |
| `branch` | Current branch name | `master` |
| `staged` | `+` if staged changes exist | `+` |
| `unstaged` | `*` if unstaged changes exist | `*` |

**Color scheme:**
- Path: cyan
- Git info (pipe through branch): green
- Prompt character `>`: yellow
- Default text: terminal foreground

**Implementation:**
- Run `git rev-parse --is-inside-work-tree` to detect git repos
- Run `git rev-parse --show-prefix` for subdirectory
- Run `git symbolic-ref --short HEAD` for branch name
- Run `git diff --quiet` and `git diff --cached --quiet` for dirty state
- Cache results and refresh on:
  - Directory change (`cd`)
  - Command completion (any command might modify git state)
- Use `os/exec` with short timeouts (100ms) to avoid blocking the UI on
  large repos

### 3.8 Basic Terminal Features

Beyond the block-based features, rterm provides standard terminal
functionality:

- **ANSI color support**: 16 colors, 256 colors, and 24-bit truecolor
- **Text styles**: Bold, italic, underline, strikethrough, dim, inverse
- **Cursor modes**: Block, underline, bar (for subprocess TUI apps)
- **Terminal resize**: Blocks re-flow on window resize; PTY SIGWINCH sent
- **Environment variables**: `TERM=xterm-256color`, proper `COLUMNS`/`LINES`
- **Signal handling**: Ctrl-C sends SIGINT, Ctrl-Z sends SIGTSTP, Ctrl-D sends EOF
- **Working directory tracking**: Prompt updates on `cd`
- **Tab completion**: Context-aware via elvish's completion engine
- **Alt-screen**: Full-screen apps (vim, less, htop) get a dedicated buffer
  that does not create blocks

---

## 4. Data Flow

```
┌─────────────────────────────────────────────────┐
│                    Gio Window                    │
│                                                  │
│  ┌──────────┐   ┌────────────┐   ┌───────────┐  │
│  │  Editor   │──▸│   Shell    │──▸│    PTY    │  │
│  │  Widget   │   │  Engine    │   │  Manager  │  │
│  └──────────┘   └────────────┘   └───────────┘  │
│       ▲              │                  │        │
│       │              ▼                  ▼        │
│  ┌──────────┐   ┌────────────┐   ┌───────────┐  │
│  │  Block   │◂──│   Block    │◂──│    VT     │  │
│  │  List UI │   │   Model    │   │  Parser   │  │
│  └──────────┘   └────────────┘   └───────────┘  │
│                      │                           │
│                      ▼                           │
│                ┌────────────┐                    │
│                │  History   │                    │
│                │  (BoltDB)  │                    │
│                └────────────┘                    │
└─────────────────────────────────────────────────┘
```

**Flow for a command execution:**

1. User types in the Editor Widget
2. On Enter, the command text is sent to the Shell Engine
3. Shell Engine parses via `pkg/parse` and evaluates via `pkg/eval`
4. For external commands, the PTY Manager spawns the process
5. PTY output is read in a goroutine and fed to the VT Parser
6. VT Parser converts escape sequences to styled text rows
7. Styled rows are appended to the active Block's output buffer
8. The Block List UI re-renders on each new row (frame-throttled)
9. On completion, the block is finalized with exit code
10. The command is added to History

---

## 5. Key Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| `gioui.org` | GUI framework | latest (pin in go.mod) |
| `gioui.org/x` | Extended Gio widgets | latest |
| `src.elv.sh` | Elvish shell packages | v0.21+ |
| `github.com/creack/pty` | PTY allocation | v1.1+ |
| `go.etcd.io/bbolt` | BoltDB (via elvish) | transitive |

Minimal dependency footprint — elvish and Gio are the two major dependencies.

---

## 6. Implementation Phases

### Phase 1: Foundation (Scaffold + Basic Shell)

**Goal:** A window that accepts commands and shows output.

1. Initialize Go module; add Gio and elvish dependencies
2. Create the Gio window with a basic event loop
3. Implement a minimal text editor widget (single-line input)
4. Embed elvish `Evaler`; evaluate commands and capture output
5. Render output as plain unstyled text below the editor
6. Basic Enter-to-submit, output-appears-below cycle

**Milestone:** Type `echo hello` → see `hello` appear below.

### Phase 2: Block Model

**Goal:** Commands and output are organized into blocks.

1. Define `Block` struct and `Session` model
2. Create `BlockView` widget — renders one block (header + output)
3. Create `BlockList` widget — scrollable list of blocks
4. Pin editor at the bottom; new blocks appear above it
5. Add exit code tracking and status indicators (✓/✗)
6. Add block dividers and visual styling

**Milestone:** Run several commands; scroll through blocks; see exit statuses.

### Phase 3: VT Parsing + Styled Output

**Goal:** Terminal output renders with colors and styles.

1. Implement a VT100/ANSI escape sequence parser
2. Map ANSI SGR codes to Gio text styles (color, bold, etc.)
3. Implement the circular buffer for output rows
4. Support 16-color, 256-color, and truecolor
5. Handle cursor movement within a block's output grid

**Milestone:** `ls --color` shows colored output; `curl` progress bars render.

### Phase 4: Graphical Editor

**Goal:** Point-and-click command editing.

1. Extend the editor with multi-line support (Shift-Enter)
2. Implement click-to-position cursor using glyph metrics
3. Add text selection (Shift+Arrow, mouse drag)
4. Implement clipboard integration (Ctrl-C/V/X)
5. Add undo/redo stack
6. Integrate elvish parser for syntax highlighting
7. Add tab completion popup

**Milestone:** Click anywhere in a multi-line command to edit; see syntax colors.

### Phase 5: History + Ctrl-R

**Goal:** Searchable command history panel.

1. Connect to elvish's history store (BoltDB via daemon)
2. Build the history overlay panel (filter input + scrollable list)
3. Implement fuzzy substring matching
4. Wire Ctrl-R to open, Escape to close, Enter to select
5. Deduplicate consecutive identical commands

**Milestone:** Ctrl-R opens history; type to filter; select inserts into editor.

### Phase 6: Folded View

**Goal:** Toggle to a summary view of all blocks.

1. Build `FoldedView` widget — list of block summary rows
2. Implement toggle (Ctrl-Shift-F) between full and folded views
3. Add click-to-expand in folded view
4. Add search/filter in folded view
5. Always show running block expanded

**Milestone:** Toggle to folded view; see all commands with statuses; click to expand.

### Phase 7: Git-Aware Prompt

**Goal:** Prompt shows git branch and dirty state.

1. Implement git status detection (branch, staged, unstaged)
2. Implement path shortening (`~/…/last/two`)
3. Build the prompt renderer with color segments
4. Cache git status; refresh on directory change and command completion
5. Handle non-git directories gracefully (show path only)

**Milestone:** Navigate to a git repo; prompt shows `~/…/go/rterm|. master+* >`.

### Phase 8: Polish + Alt-Screen

**Goal:** Handle full-screen apps; refine UX.

1. Implement alt-screen buffer for TUI apps (vim, less, htop)
2. Add terminal resize handling (SIGWINCH, re-flow)
3. Add scrollbar to block list
4. Implement sticky command header for long blocks
5. Add theming support (dark/light, configurable colors)
6. Add Ctrl-C/Z/D signal forwarding
7. Keyboard shortcut help overlay

**Milestone:** Run `vim` in rterm; resize the window; everything works smoothly.

---

## 7. Open Questions

1. **Elvish daemon**: Should rterm start its own elvish daemon for history
   sharing, or run in daemon-less mode for simplicity in the first version?
   *Recommendation: Start daemon-less; add daemon support in a later phase.*

2. **Config format**: Elvish uses its own scripting language for configuration.
   Should rterm use elvish scripts, TOML, or something else?
   *Recommendation: TOML for rterm-specific config; source `~/.elvish/rc.elv`
   for shell config.*

3. **Plugin system**: Elvish supports modules. Should rterm expose its UI
   features (blocks, notifications) to elvish scripts?
   *Recommendation: Yes, eventually. Add a `rterm:` namespace with Go
   functions. Not in the first version.*

4. **Sixel/image support**: Should blocks support inline image rendering?
   *Recommendation: Defer to a future phase. Gio can render images, so this
   is feasible.*

---

## 8. Non-Goals (For Now)

- **Tabs / split panes** — Single session per window initially
- **Remote / SSH sessions** — Local shell only initially
- **GPU-shader text rendering** — Gio's vector renderer is sufficient; if
  performance is an issue, consider compute-shader approaches later
- **Wails/web fallback** — Only if Gio proves unworkable
- **Windows support** — Target Linux and macOS first; Gio supports Windows
  but testing is deferred

---

## 9. References

- [Elvish Shell](https://github.com/elves/elvish) — Shell engine (Go)
- [Gio UI](https://gioui.org) — GUI framework (Go)
- [Gritty](https://github.com/viktomas/gritty) — Terminal emulator in Go + Gio (reference)
- [Warp Terminal](https://www.warp.dev) — Block-based terminal (inspiration)
- [Warp: Data Structure Behind Terminals](https://www.warp.dev/blog/the-data-structure-behind-terminals) — Circular buffer design
- [creack/pty](https://github.com/creack/pty) — Go PTY library
- [Fyne Terminal](https://github.com/fyne-io/terminal) — Alternative Go terminal (reference)
- [Darktile](https://github.com/liamg/darktile) — GPU-rendered Go terminal (reference, abandoned)
