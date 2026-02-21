# Application Discovery System

MenuWorks can automatically discover installed applications and generate a `config.yaml` file.

## Overview

The `generate` subcommand scans the system for installed applications using platform-specific sources, then writes a MenuWorks-compatible `config.yaml` grouped by category.

```
menuworks generate [flags]
```

## Architecture

The discovery system lives entirely in the `discover/` package tree and is **completely isolated** from the existing menu/UI code. It does not import any packages from `config/`, `menu/`, `ui/`, or `exec/`.

```
discover/
    discover.go              # Core types: Source, DiscoveredApp, Category, Registry
    writer.go                # Generates config.yaml from discovered apps
    discover_test.go         # Core tests (registry, writer)
    windows/
        startmenu.go         # Start Menu shortcut (.lnk) discovery
        steam.go             # Steam library manifest parsing
        programfiles.go      # Program Files .exe scanning
        windows_test.go      # Windows source tests
    linux/                   # (future)
    darwin/                  # (future)
```

### Key Types

```go
// Source discovers applications from a specific location.
type Source interface {
    Name() string                        // e.g. "steam", "startmenu"
    Category() string                    // e.g. "Games", "Applications"
    Discover() ([]DiscoveredApp, error)  // find apps on this system
    Available() bool                     // is this source present?
}

// DiscoveredApp represents a single discovered application.
type DiscoveredApp struct {
    Name     string   // display name
    Exec     string   // command to launch (platform-specific)
    Source   string   // which source found it ("steam", "startmenu", etc.)
    Category string   // grouping category
}

// Registry holds all known sources and orchestrates discovery.
type Registry struct { ... }
```

### Isolation Principle

The discovery code generates YAML output directly — it does **not** depend on `config.Config` or any other existing MenuWorks types. This ensures:
- Changes to the menu/UI code never break discovery
- Changes to discovery never break the menu/UI
- The discovery system can be tested independently

The only integration point is `cmd/menuworks/main.go`, which checks for the `generate` subcommand before entering the TUI.

## Usage

### Basic Usage

```
menuworks generate
```

Scans all available sources and writes `config.yaml` to the current directory.

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--output` | Output file path | `config.yaml` |
| `--sources` | Comma-separated list of sources to use | all available |
| `--list-sources` | List available sources and exit | |
| `--dry-run` | Print generated config to stdout instead of writing a file | |
| `--merge` | Merge discovered items into an existing config file | |

### Examples

```bash
# Discover everything and write config.yaml
menuworks generate

# Only scan Steam library
menuworks generate --sources steam

# Preview what would be generated
menuworks generate --dry-run

# Write to a specific file
menuworks generate --output my-config.yaml

# List available discovery sources
menuworks generate --list-sources

# Merge new discoveries into existing config
menuworks generate --merge --output existing-config.yaml
```

## Sources

### Windows

#### Start Menu (`startmenu`)
- **Category:** Applications
- **Scans:** `%ProgramData%\Microsoft\Windows\Start Menu\Programs` and `%APPDATA%\Microsoft\Windows\Start Menu\Programs`
- **Method:** Resolves `.lnk` shortcut files to extract target executable paths
- **Filters:** Skips uninstallers, updaters, and documentation shortcuts

#### Steam (`steam`)
- **Category:** Games
- **Scans:** Steam library folders via `libraryfolders.vdf` and app manifests (`appmanifest_*.acf`)
- **Method:** Parses Valve's VDF format to find installed games
- **Launch:** Uses `steam://rungameid/<appid>` protocol for launching

#### Program Files (`programfiles`)
- **Category:** Applications
- **Scans:** `C:\Program Files` and `C:\Program Files (x86)`
- **Method:** Finds `.exe` files in top-level subdirectories (non-recursive beyond one level)
- **Filters:** Skips uninstallers, updaters, helper executables, DLL hosts

### Linux (Future)

Planned sources:
- **Desktop entries** (`.desktop` files in XDG directories)
- **Flatpak** applications
- **Snap** packages
- **Steam** (Linux variant)

### macOS (Future)

Planned sources:
- **Applications folder** (`/Applications/*.app`)
- **Homebrew Cask** applications
- **Steam** (macOS variant)

## Generated Config Format

The generator produces a standard MenuWorks `config.yaml`:

```yaml
title: "MenuWorks 3.X"
theme: "dark"
themes:
  dark:
    background: "blue"
    text: "silver"
    border: "aqua"
    highlight_bg: "navy"
    highlight_fg: "white"
    hotkey: "yellow"
    shadow: "gray"
    disabled: "gray"

items:
  - type: submenu
    label: "Games"
    target: "games"
  - type: submenu
    label: "Applications"
    target: "applications"
  - type: separator
  - type: back
    label: "Quit"

menus:
  games:
    title: "Games"
    items:
      - type: command
        label: "Half-Life 2"
        exec:
          windows: "start steam://rungameid/220"
      - type: back
        label: "Back"
  applications:
    title: "Applications"
    items:
      - type: command
        label: "Notepad++"
        exec:
          windows: "start \"\" \"C:\\Program Files\\Notepad++\\notepad++.exe\""
      - type: back
        label: "Back"
```

## Merge Mode

When `--merge` is used with an existing config file:
- Existing menu structure is preserved
- New discovered items are appended to matching category submenus
- New categories create new submenus
- Duplicate detection by executable path prevents adding the same app twice
- A backup of the original file is created (`config.yaml.bak`)

## Adding New Sources

To add a new discovery source:

1. Create a new file in the appropriate platform directory (e.g., `discover/windows/newsource.go`)
2. Implement the `Source` interface
3. Register it in the platform's `init()` or registration function
4. Add tests in the platform's test file
5. Update this document

```go
package windows

import "github.com/benworks/menuworks/discover"

type MySource struct{}

func (s *MySource) Name() string     { return "mysource" }
func (s *MySource) Category() string { return "Applications" }
func (s *MySource) Available() bool  { /* check if source exists */ }

func (s *MySource) Discover() ([]discover.DiscoveredApp, error) {
    // Scan and return discovered apps
}
```
