# MenuWorks 3.0

A **retro DOS-style hierarchical menu TUI application** for Windows, Linux, and macOS. Built in Go with a single, self-contained binary that requires no external dependencies.

![Version](https://img.shields.io/badge/version-1.0.0-blue) ![License](https://img.shields.io/badge/license-MIT-green) ![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)

## Features

- **Single Self-Contained Binary** — No runtime dependencies, no external files required (except config)
- **Retro DOS Aesthetic** — 80×25 terminal layout with double-line borders, drop shadows, and VGA colors
- **Hierarchical Menus** — Unlimited menu nesting with menu chaining via `target`
- **Hotkeys** — Explicit hotkey assignment or auto-generated from menu labels
- **Configuration** — YAML-based config file (`config.yaml`) with embedded default fallback
- **Cross-Platform Commands** — Execute shell commands (auto-detects Windows cmd.exe vs sh)
- **Command Output Viewer** — Scrollable full-screen display of command output with ↑/↓ and PgUp/PgDn navigation
- **Dynamic Config Reload** — Press `R` in any menu to reload config without restarting
- **Selection Memory** — Current menu position preserved during session (resets on config reload)
- **Graceful Error Handling** — Clear error dialogs for missing config, invalid YAML, and broken menu links

## Installation

### Option 1: Download Pre-Built Binary

Download the latest binary for your platform:
- **Windows**: `dist/menuworks-windows.exe`
- **Linux**: `dist/menuworks-linux`
- **macOS (Intel)**: `dist/menuworks-macos`
- **macOS (ARM/M1)**: `dist/menuworks-macos-arm64`

Place the binary anywhere, optionally alongside a `config.yaml` file.

### Option 2: Build from Source

**Requirements:**
- Go 1.21+ (portable installation included: `bin/go/`)
- Windows, Linux, or macOS

**Build All Platforms:**
```powershell
# PowerShell (Windows)
.\build.ps1

# Bash (Unix/Linux/macOS)
chmod +x build.sh
./build.sh
```

**Build Single Target:**
```powershell
# PowerShell
.\build.ps1 -Target windows -Version 1.0.0

# Bash
./build.sh linux
```

Binaries are output to `dist/`.

## Configuration

MenuWorks uses a **YAML configuration file** located in the same directory as the binary (or specified via environment).

### Default Config

On first run, if `config.yaml` is missing, MenuWorks creates one with sample menus and commands.

### Configuration Schema

```yaml
title: "MenuWorks 2.0"

items:
  - type: submenu
    label: "System Tools"
    hotkey: "S"           # Optional; auto-assigned if omitted
    target: "system"      # Menu name to open

  - type: command
    label: "Show Date"
    hotkey: "D"
    exec:
      windows: "echo Current date is %DATE%"  # Windows command
      linux: "date"                            # Linux command
      mac: "date"                              # macOS command
    showOutput: false   # disable output from command (e.g for app links)

  - type: separator       # Visual divider (no label or hotkey)

  - type: back
    label: "Quit"         # Root menu "back" = exit; submenu "back" = return

menus:
  system:
    title: "System Tools"
    items:
      - type: command
        label: "List Files"
        exec:
          windows: "dir"
          linux: "ls -la"
          mac: "ls -la"
      
      - type: back
        label: "Back"
```

### Item Types

| Type | Purpose | Fields |
|------|---------|--------|
| `command` | Run shell command | `label`, `exec` (OS variants), `hotkey` (optional), `showOutput` (optional) |
| `submenu` | Open another menu | `label`, `target` (menu name), `hotkey` (optional) |
| `back` | Return to parent (or quit if root) | `label` |
| `separator` | Visual divider | *(no other fields)* |

### Cross-Platform Command Execution

MenuWorks supports **OS-specific commands** via the `exec` field. Each command item must define variants for the operating systems you want to support:

```yaml
- type: command
  label: "Show System Info"
  exec:
    windows: "systeminfo"
    linux: "uname -a"
    mac: "uname -a"
```

**Behavior:**
- If the current OS has a defined variant, that command executes
- If the current OS variant is missing, the item appears **disabled** (dimmed) in the menu
- At least one OS variant must be defined for each command

**Supported OS identifiers:**
- `windows` — Windows (cmd.exe)
- `linux` — Linux (sh)
- `mac` — macOS (sh)

### Hotkeys

- **Explicit assignment**: Use `hotkey: "S"` on any item
- **Auto-assignment**: Left-to-right scan of the label for the first unused letter
  - Non-alphabetic characters are skipped
  - Example: "Run (Backup)" → scans R, U, N, B, A, C, K, U, P → uses first available

### Command Output Display

By default, all commands display their output in a scrollable full-screen viewer after execution. To hide output for a command (e.g., for background tasks), set `showOutput: false`:

```yaml
- type: command
  label: "Silent Task"
  exec:
    command: "some-background-task.sh"
  showOutput: false  # Output will not be displayed
```

## Usage

### Running

```bash
./menuworks-windows.exe      # Windows
./menuworks-linux            # Linux
./menuworks-macos            # macOS
```

### Navigation

| Key | Action |
|-----|--------|
| **↑ / ↓** | Move selection (in menu); scroll up/down (in output viewer) |
| **→ / Enter** | Select/open submenu or execute command |
| **← / Esc** | Return to parent menu (or quit at root); return to menu from output viewer |
| **PgUp / PgDn** | Page up/down in output viewer |
| **R** | Reload config (in menu view only) |
| **Hotkey** (A-Z) | Directly activate menu item |
| **Any Other Key** | Return to menu from output viewer |

### Terminal Requirements

- **Minimum**: 80×25 character terminal
- **Resize handling**: If terminal is too small, an error dialog appears; resize and the UI auto-recovers
- **On resize dialog**: Press **Esc** to quit, or resize terminal to continue

## Examples

### Example 1: Simple Admin Menu (Cross-Platform)

```yaml
title: "Admin Panel"

items:
  - type: submenu
    label: "System"
    target: "sys"
  
  - type: submenu
    label: "Network"
    target: "net"
  
  - type: separator
  
  - type: back
    label: "Exit"

menus:
  sys:
    title: "System Tools"
    items:
      - type: command
        label: "Disk Usage"
        exec:
          windows: "wmic logicaldisk get name,size,freespace"
          linux: "df -h"
          mac: "df -h"
        showOutput: true  # Default - show output in viewer
      
      - type: command
        label: "Processes"
        exec:
          windows: "tasklist"
          linux: "ps aux"
          mac: "ps aux"
      
      - type: back
        label: "Back"

  net:
    title: "Network Tools"
    items:
      - type: command
        label: "Ping Google"
        exec:
          windows: "ping 8.8.8.8 -n 1"
          linux: "ping -c 1 8.8.8.8"
          mac: "ping -c 1 8.8.8.8"
      
      - type: back
        label: "Back"
```

### Example 2: Development Tools

```yaml
title: "Dev Tools"

items:
  - type: submenu
    label: "Build & Deploy"
    target: "build"
  
  - type: submenu
    label: "Testing"
    target: "test"

menus:
  build:
    title: "Build & Deploy"
    items:
      - type: command
        label: "Build Project"
        exec:
          windows: "cargo build --release"
          linux: "cargo build --release"
          mac: "cargo build --release"
      
      - type: command
        label: "Deploy to Staging"
        exec:
          windows: ".\\scripts\\deploy-staging.bat"
          linux: "./scripts/deploy-staging.sh"
          mac: "./scripts/deploy-staging.sh"
      
      - type: back
        label: "Back"

  test:
    title: "Testing"
    items:
      - type: command
        label: "Run Unit Tests"
        exec:
          windows: "cargo test"
          linux: "cargo test"
          mac: "cargo test"
      
      - type: command
        label: "Run Integration Tests"
        exec:
          windows: "cargo test --test '*'"
          linux: "cargo test --test '*'"
          mac: "cargo test --test '*'"
      
      - type: back
        label: "Back"
```

## Troubleshooting

### Config File Not Found

MenuWorks looks for `config.yaml` in:
1. Same directory as the binary
2. Current working directory

If neither exists, MenuWorks creates the embedded default config.

### YAML Parse Error

MenuWorks shows a dialog with:
- **Retry** — Fix the file and try again
- **Use Default** — Load embedded default config
- **Exit** — Quit application

### Menu Item Not Appearing

Check:
- Item type is valid: `command`, `submenu`, `back`, `separator`
- For `submenu` items: `target` menu exists in `menus:`
- YAML indentation is correct (spaces, not tabs)
- No invalid field names in the config

### Terminal Resize Issue

MenuWorks automatically handles terminal resize. If the terminal is too small (<80×25), an error dialog appears. Resize your terminal to at least 80×25 and it auto-recovers.

## Architecture

```
menuworks/
├── cmd/menuworks/
│   └── main.go              # Entry point, event loop
├── config/
│   └── config.go            # YAML loading, validation, embedding
├── menu/
│   └── navigator.go         # Menu navigation state, hotkey assignment
├── ui/
│   ├── screen.go            # Terminal rendering (tcell wrapper)
│   └── menu.go              # Menu/dialog drawing
├── exec/
│   └── exec.go              # Cross-platform command execution
├── assets/
│   └── config.yaml          # Embedded default config
├── build.ps1                # PowerShell build script
├── build.sh                 # Bash build script
└── go.mod                   # Go module definition
```

### Key Design Decisions

- **Single Binary**: All assets (config template, logo) embedded via `//go:embed`
- **No Mouse**: Keyboard-only navigation for retro feel
- **Deterministic Rendering**: No flicker, smooth 400ms splash screen
- **Selection Memory**: Per-session tracking allows quick menu traversal
- **Config Reload**: Live reload without losing user's current menu depth

## Dependencies

- **tcell/v2** — Terminal rendering library
- **gopkg.in/yaml.v3** — YAML parsing

Both are included in `go.mod` and automatically downloaded during build.

## Performance

- **Binary Size**: ~8-15 MB (single executable)
- **Startup Time**: <100ms (excluding splash screen delay)
- **Memory Usage**: ~5-10 MB
- **Rendering**: 60 FPS (smooth on all platforms)

## Platform Notes

### Windows

- Assumes `cmd.exe` for command execution
- Windows Terminal or ConEmu recommended for best colors
- Tested on Windows 10/11

### Linux

- Uses `/bin/sh` for command execution
- Tested on Ubuntu 20.04+, Fedora, Debian
- SSH terminal support: Yes

### macOS

- Uses `/bin/sh` for command execution
- Intel (amd64) and Apple Silicon (arm64) builds provided
- Tested on macOS 11+

## Building for Different Architectures

```powershell
# Windows - ARM64 (future-proofing)
$env:GOOS = "windows"; $env:GOARCH = "arm64"
go build -o dist/menuworks-arm64.exe cmd/menuworks/main.go

# Linux - ARM64 (Raspberry Pi, etc.)
$env:GOOS = "linux"; $env:GOARCH = "arm64"
go build -o dist/menuworks-arm64 cmd/menuworks/main.go
```

## Future Enhancements

- [ ] Mouse support (optional)
- [ ] Colored menu items
- [ ] Command aliases
- [ ] Menu search/filter
- [ ] User config directory
- [ ] Menu item descriptions/help text
- [ ] Command history/logging

## License

MIT License — See LICENSE file for details.

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Commit with clear messages
4. Submit a pull request

---

**MenuWorks 2.0** — Because menus never go out of style.
