# Copilot Instructions — Retro MenuWorks‑Style TUI (Go)

- **important** Be concise, use context wisely.
- **important** Go is installed in ./bin/go, (e.g bin\go\bin\go) not in PATH. Use that path for all Go commands.
- **important** This is built on windows; the `head` command is not available. 
- **important** Build via `.\build.ps1 -Target windows -Version 1.0.0` on windows.
- **important** Run tests via `\.\test.ps1` (defaults to `./config` and `./menu`), or pass packages: `\.\test.ps1 -Packages ./config,./menu`.
## Project Goal
Build a **single self‑contained Go binary** for **Windows, Linux, and macOS** that replicates the core functionality and user experience of **MenuWorks 2.10**, with a recognisable 1988 DOS aesthetic.  
The UI should be retro, clean, responsive, and centered around **hierarchical menus** and **menu chaining**.

## Core Features
- **Single binary**, no external runtime dependencies.
- **YAML configuration file** for user‑editable menus.
- **Embedded default config**; generate it on first run if missing.
- **Hierarchical menus** (menus defined under `menus:`).
- **Menu chaining** (menu items link to other menus via `target:`).
- **Item types:** `command`, `submenu`, `back`, `separator`.
- **Hotkeys** (explicit or auto‑assigned).
- **Pop‑up dialogs** for errors, confirmations, and post‑command messages.
- **Retro UI:** double‑line borders, drop shadows, VGA 16‑color palette, 80×25 layout.
- **Config reload** via keypress (`R`).
- **Cross‑platform command execution** (Windows: `cmd /c`, Unix: `sh -c`).

## Config File Location

**Filename:** `config.yaml`  
**Location:** Same directory as the binary.

The binary name is `menuworks` (or `menuworks.exe` on Windows).

## YAML Schema (Explicit, Type‑Based)
Each menu item must include a `type` field.  
Type‑specific metadata lives in a namespaced block.

### Root menu
The top‑level `title:` and `items:` define the root menu.

### Example:
```yaml
title: "MenuWorks 3.0"

items:
  - type: submenu
    label: "System Tools"
    hotkey: "S"
    target: "system"

  - type: submenu
    label: "Utilities"
    target: "utils"

  - type: back
    label: "Quit"

menus:
  system:
    title: "System Tools"
    items:
      - type: command
        label: "Show Date"
        exec:
          windows: "echo Current date is %DATE%"
          linux: "date"
          mac: "date"

  utils:
    title: "Utilities"
    items:
      - type: command
        label: "List Files"
        exec:
          windows: "dir"
          linux: "ls -la"
          mac: "ls -la"
```

### Item Types
#### `command`
Runs a shell command with OS-specific variants.
```yaml
- type: command
  label: "Show Date"
  hotkey: "D"
  exec:
    windows: "echo Current date is %DATE% %TIME%"
    linux: "date"
    mac: "date"
```

**Exec Field:**
Each command must define at least one OS variant:
- `windows` — Command for Windows (executed via cmd.exe)
- `linux` — Command for Linux (executed via sh)
- `mac` — Command for macOS (executed via sh)

If the current OS has no defined variant, the item appears disabled in the menu and cannot be selected.

#### `submenu`
Links to another menu.
```yaml
- type: submenu
  label: "Tools"
  target: "tools"
```

#### `back`
Returns to the parent menu. In the root menu, exits the application.
```yaml
- type: back
  label: "Return"
```

#### `separator`
Visual separator line (non-selectable). Separators require only the `type` field; labels, hotkeys, and all other metadata fields are prohibited.
```yaml
- type: separator
```

**Missing Target Handling:** If a `submenu` references a `target:` that doesn't exist in `menus:`, the item is shown but disabled (dimmed) to indicate its presence. An error pop-up is displayed the *first time* the menu containing the broken item is opened, then not repeated on subsequent visits to that menu unless the config is reloaded. If a user attempts to activate a disabled item (e.g., by pressing Enter or its hotkey), a brief error message confirms it cannot be accessed. Deeper menu chains do not break unless the user explicitly tries to traverse through the invalid link.

## UI Requirements
- **UTF‑8 box‑drawing characters**.
- **Double‑line borders** for main windows.
- **Single‑line borders** for dialogs.
- **Drop shadows:** space character with dark‑gray background, offset +1 row/+2 columns. Shadows are clipped cleanly at terminal boundaries, meaning shadow characters do not render beyond the rightmost column or bottom row of the 80×25 layout, preventing overflow or wrapping artifacts.
- **VGA 16‑color palette**:
  - Background: dark blue  
  - Borders: bright cyan  
  - Highlight: white on blue  
  - Text: light gray  
  - Shadow: dark gray  
  - Hotkey: brighter foreground color (e.g., bright white or yellow)
- **Hotkey display:** Highlight the hotkey letter within the label using a brighter foreground color only (no underline or background changes). Example: "**S**ystem Tools" where S is brighter (DOS-style). When a label is truncated, the hotkey is highlighted only if the hotkey character remains visible after truncation. For example, "Save (Backup)" with hotkey S, when truncated to "Save (B…", still displays the S in bright color because it remains visible; if truncation removes the hotkey letter entirely, the highlight is omitted.
- **Label truncation:** If a label exceeds line width in 80-column layout, truncate with ellipsis (…).
- **Empty menus:** Display centered "(No items)" placeholder in both root and submenus whenever a menu contains zero selectable items, whether because the user defined an empty menu or because all items were removed during reload. The placeholder is accompanied by a Back option.
- **Centered 80×25 layout** on larger terminals.
- **Terminal size < 80×25:** Show error pop-up and enter a resize-wait loop. Continuously process resize events from tcell, keeping the error pop-up visible, and automatically re-render the full UI as soon as the terminal reaches at least 80×25. The splash screen does not reappear after recovery (do not exit).

## Runtime Behaviour
### Startup
1. Load `config.yaml` from binary directory.
2. If missing, write embedded default config.
3. Show splash screen (fixed 400ms delay for consistent retro feel, not dismissible by keypress, not user-configurable) with:
   - Project name  
   - Version (injected via `-ldflags` at build time, read from a version variable in main)
   - Optional ASCII logo
   - During the splash screen, key events are consumed and discarded by reading and ignoring tcell events rather than flushing the terminal buffer, ensuring no accidental actions occur afterward  

### Navigation
- Up/Down: move selection  
- Right: open submenu (if applicable)  
- Left or Esc: go back  
- Enter: activate item  
- Hotkey: activate item (case-insensitive)
- `R`: reload config (only in menu view; the key is ignored silently during command execution or dialogs, with no error message or visual indicator that reload is unavailable)

**Selection behavior:**
- When first opening a menu, highlight the first selectable item (skip separators).
- All menus remember their previously highlighted items for the duration of the session, uniformly across all menu depths.
- This memory resets after config reload (structure may have changed).

**Hotkey assignment:**
- Explicitly defined via `hotkey:` field (case-insensitive).
- Auto-assigned: scan the label left-to-right considering only A–Z alphabetic characters, and choose the first letter not already assigned to another item in the same menu. Non-alphabetic characters (punctuation, symbols, digits) are skipped during the scan. This scan happens before any truncation, so a label like "Run Command (Now)" scans as R, U, N, C, O, M, M, A, N, D, N, O, W in that order.
  - Example: "Save File" → first letter is S (unused), hotkey is S
  - Example: "Settings" → first letter is S (if already used), second is e (if unused), hotkey is e
  - Example: ">>>", "123" → no alphabetic characters, no hotkey assigned
  - Example: "Save (Backup)" → scans S, A, V, E, B, A, C, K, U, P in that order; first unused one is assigned
- If no letters are available in the label or all are already assigned in the menu, the item has no hotkey.

### Command Execution
- Switch to full alternate screen buffer (tcell requires explicit enable), clearing the UI entirely.
- Run command using platform‑appropriate shell:
  - Windows: `cmd /c <command>`
  - Unix: `sh -c <command>`
- **Working directory:** User's CWD at binary launch (not binary directory).
- Commands run in a normal terminal environment with full scrollback; long output scrolls naturally as with any shell command.
- After completion, show:
  ```
  Command finished. Press any key to return.
  ```
- Restore UI:
  - Disable alternate screen buffer (tcell requires explicit disable).
  - Restore terminal state (echo mode, colors, cursor visibility).

### Error Handling
**Invalid YAML:**
- Show a three-option error pop-up with detailed message: "Error loading config: <parse error>"
- Navigate with arrow keys and Enter to select
- Options:
  - **Retry:** Re-read the same file (user can fix it externally)
  - **Use default config:** Overwrite with embedded default
  - **Exit:** Quit application
- This distinction ensures fatal errors remain interactive while structural errors (missing targets) are kept lightweight

**Missing submenu target:**
- Show a single-button error pop-up with message: "Error: submenu target '<target>' not found"
- Dismiss with Enter or Esc
- Render the item as disabled (dimmed) in the menu and leave the application running
- The application tracks missing-target errors by keeping an in-memory set of submenu names that have already triggered an error pop-up, so each broken link is reported only once per session until a config reload clears that state. If a user attempts to activate a disabled item (e.g., by pressing Enter or its hotkey), a brief error message confirms it cannot be accessed.

### Config Reload
- Press `R` in menu view to reload `config.yaml`.
- Brief "Config reloaded" message appears in footer.
- Not available during command execution or in dialogs.
- **Selection memory behavior:**
  Following a deterministic flow after config reload:
  - If the current menu still exists and the previously highlighted item still exists at the same index, the highlight is restored.
  - If the menu exists but the previously highlighted item does not (e.g., it was deleted), the highlight moves to the first selectable item in that menu.
  - If the current menu no longer exists, the UI falls back to the nearest surviving parent menu; if no parent survives, the UI returns to the root menu.
  - If the root menu becomes empty after reload (no selectable items), the Back item is relabeled as "Quit" to avoid confusing the user with a meaningless "Back" action.

## Default Config Content
The embedded default config in `/assets/` should contain:
- Main menu: "MenuWorks 3.0"
- Submenus: "System Tools" and "Utilities"
- Safe cross-platform commands using `echo` (works on Windows, Linux, macOS)
- Example separators
- Back/Quit options

**Platform-specific commands:** MenuWorks natively supports OS-specific command variants directly in the YAML config. Each command item defines separate command strings for `windows`, `linux`, and `mac` platforms. This allows a single config file to be used across multiple operating systems without duplication or external scripts. At runtime, the OS type is detected and the appropriate variant is selected. If a variant is missing for the current OS, the menu item is silently disabled (shown dimmed) and cannot be selected.

## Directory Structure
```
/cmd/menuworks/main.go    (entry point, version constant)
/ui/                      (drawing, layout, colors, splash, pop-ups)
/menu/                    (menu tree, navigation, state, hotkey assignment)
/config/                  (load, parse, validate, reload)
/exec/                    (cross-platform command execution, alternate screen)
/assets/                  (embedded default config + splash)
```

## Go Technology Requirements
- Terminal library: **tcell**
- Config parsing: **gopkg.in/yaml.v3**
- Embedding: **//go:embed**
- Build: `go build` produces a single binary on all platforms.
- **Version injection:** Use `-ldflags` to inject the version string at build time into a version variable in `main.go` (e.g., `go build -ldflags "-X main.version=1.0.0"`). This keeps the build process clean and automated.

## Code Style Guidelines
- Keep modules small and focused.
- Prefer explicit state structs over globals.
- Avoid unnecessary abstractions.
- Rendering must be deterministic and flicker‑free.
- Use clear, readable names for menu navigation logic.

## Event Handling & Concurrency Guidelines

**Critical architectural patterns to prevent event-related bugs:**

### Single Event Source Pattern
- **Rule:** Establish ONE event poller goroutine immediately after screen initialization.
- **Implementation:** Use `StartEventPoller()` that returns a channel, started once in `main()`.
- **Rationale:** tcell's `PollEvent()` is blocking and cannot be safely called from multiple places. Splitting event consumption between direct calls and channels causes race conditions and lost events.

### Event Poller Initialization Timing
- **Rule:** Start event poller IMMEDIATELY after `screen.Init()`, before any other function that might need events.
- **Wrong:** Starting poller after config load, terminal size check, or splash screen.
- **Right:** Start poller as second step after screen creation, pass channel to all functions that need events.
- **Why:** Functions like `ensureTerminalSize()`, `handleConfigError()`, and splash screen event drain all need the event channel. Starting late causes hangs.

### Goroutine Leak Prevention
- **Anti-pattern:** Creating new goroutines in functions called repeatedly (loops, event handlers).
- **Example of leak:** 
  ```go
  func PollEventWithTimeout(timeout time.Duration) tcell.Event {
      eventChan := make(chan tcell.Event, 1)
      go func() { eventChan <- s.PollEvent() }() // LEAK: goroutine never cleaned up on timeout
      select {
      case ev := <-eventChan: return ev
      case <-time.After(timeout): return nil // orphaned goroutine still blocking on PollEvent
      }
  }
  ```
- **Fix:** Use a single long-lived goroutine created once, or ensure goroutines are properly cleaned up with context cancellation.

### Platform-Specific Event Queue Behavior
- **macOS/Linux:** Terminals generate startup events (SIGWINCH resize, focus events) when tcell initializes. These MUST be consumed or they corrupt the event queue.
- **Windows:** Console API typically does not generate startup events; event queue starts clean.
- **Solution:** During splash screen, continuously drain events from the channel for 400ms. This is harmless on Windows (no events to drain) and critical on macOS (prevents hang).
- **Implementation:** 
  ```go
  splashStart := time.Now()
  for time.Since(splashStart) < 400*time.Millisecond {
      select {
      case <-eventChan: // Discard any startup events
      case <-time.After(10 * time.Millisecond):
      }
  }
  ```

### Consistent Event Channel Usage
- **Rule:** Once event poller starts, ALL event polling must use the channel. Never mix direct `PollEvent()` calls with channel-based polling.
- **Pass eventChan to:** All dialogs, resize handlers, event loops, any function that waits for user input.
- **Function signatures:** Update all dialog/handler functions to accept `eventChan <-chan tcell.Event` parameter.
- **Verification:** Search codebase for `PollEvent()` calls after event poller starts—there should be zero except inside `StartEventPoller()` itself.

### Testing Across Platforms
- **Always test on macOS when making event handling changes**—Windows may work while macOS hangs due to different event queue initialization.
- If Windows works but macOS hangs, suspect event consumption issue (startup events not drained).
- If both platforms hang after changes, suspect goroutine leak or deadlock.

## Non‑Goals
- No mouse support.
- No DOS emulation.
- No pixel‑perfect reproduction.
- No external theme files (themes may be embedded).
- No file locking (config is read-only except first-run initialization).
- Multiple instances can run concurrently.

## Deliverables Copilot Should Help Produce
- Project skeleton with directory structure.
- Rendering engine (borders, shadows, highlight bars).
- Menu tree loader + validator.
- Hierarchical menu + menu chaining logic.
- Config reload mechanism.
- Cross‑platform command execution wrapper.
- Splash screen renderer.
- Build scripts for Windows, Linux, macOS.
