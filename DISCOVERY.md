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
        xbox.go              # Xbox / Microsoft Store game discovery
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
    Source   string   // which source found it ("steam", "Start Menu", "Program Files", etc.)
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
| `--base` | Base config file to merge discovered apps into (base takes priority) | |

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

# Merge discovered apps into your own base config
menuworks generate --base myconfig.yaml --output merged.yaml

# Preview a merge without writing
menuworks generate --base myconfig.yaml --dry-run
```

**Safety:** The generate command refuses to write if the output file already exists.
Choose a different `--output` path or remove the existing file first.

## Sources

### Windows

#### Start Menu (`startmenu`)
- **Category:** Applications
- **Menu label:** `Start Menu`
- **Scans:** `%ProgramData%\Microsoft\Windows\Start Menu\Programs` and `%APPDATA%\Microsoft\Windows\Start Menu\Programs`
- **Method:** Resolves `.lnk` shortcut files to extract target executable paths
- **Filters:** Skips uninstallers, updaters, and documentation shortcuts

#### Steam (`steam`)
- **Category:** Games
- **Scans:** Steam library folders via `libraryfolders.vdf` and app manifests (`appmanifest_*.acf`)
- **Method:** Parses Valve's VDF format to find installed games
- **Launch:** Uses `steam://rungameid/<appid>` protocol for launching

#### Xbox / Microsoft Store (`xbox`)
- **Category:** Games
- **Requires:** PowerShell, Xbox app / Gaming Services installed
- **Scans:** Enumerates AppX packages registered with Windows Gaming Services via `Get-AppxPackage`
- **Method:** Cross-references installed AppX packages with the `GamingServices\GameConfig` registry to identify games. Display names and Application IDs are read from each package's `AppxManifest.xml`.
- **Launch:** Uses AUMID (Application User Model ID) pattern: `explorer.exe shell:AppsFolder\{PackageFamilyName}!{AppId}`
- **Filters:** Removes Xbox infrastructure packages (GamingServices, XboxGameBar, XboxIdentityProvider, etc.)
- **Graceful failure:** If PowerShell is not available or Get-AppxPackage is missing, the source reports as unavailable and discovery continues with other sources

> **Important — AUMID launch details:**
> Store/Xbox apps must be launched with `explorer.exe shell:AppsFolder\...`, not `start` or `cmd /c start`. The `start` command (both cmd.exe's built-in and PowerShell's `Start-Process`) cannot resolve `shell:` URIs and will fail with "file not found".
>
> The Application ID (the part after `!`) is **not** a constant — each game defines its own in `AppxManifest.xml` under `Package > Applications > Application > @Id`. Common values include `Game`, `App`, or game-specific IDs like `AppFrostpunk2Shipping` or `Microsoft.DayoftheTentacleRemastered`. Hardcoding `!App` will silently fail for most games (explorer falls back to opening a generic folder window). Always read the real App ID from the manifest.
>
> The correct registry source for installed Xbox games is `HKLM\SOFTWARE\Microsoft\GamingServices\GameConfig` (not `PackageRepository\Root` or `PackageRepository\Package`, which are incomplete). GameConfig entries are full package names (e.g. `Microsoft.Limitless_1.6.34.0_x64__8wekyb3d8bbwe`); extract the base name before the first `_` to match against `Get-AppxPackage`.

#### Program Files (`programfiles`)
- **Category:** Applications
- **Menu label:** `Program Files`
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

The generator produces a standard MenuWorks `config.yaml`. When a category (e.g. Games) has apps from multiple sources (e.g. Steam and Xbox), source-based submenus are created automatically. Single-source categories remain flat.

All generated submenus include a `separator` item between the last app entry and the `Back` item.

The `Source` field on each discovered app controls the submenu label used when grouping by source (e.g. `"Start Menu"` → label **Start Menu**, `"Program Files"` → label **Program Files**). This is separate from the source's `Name()` identifier (e.g. `startmenu`, `programfiles`), which is only used for `--sources` filtering and `--list-sources` output.

### Multi-source example (Games from Steam + Xbox)

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
    label: "Applications"
    target: "applications"
  - type: submenu
    label: "Games"
    target: "games"
  - type: separator
  - type: back
    label: "Quit"

menus:
  applications:
    title: "Applications"
    items:
      - type: command
        label: "Notepad++"
        exec:
          windows: "start \"\" \"C:\\Program Files\\Notepad++\\notepad++.exe\""
      - type: separator
      - type: back
        label: "Back"
  games:
    title: "Games"
    items:
      - type: submenu
        label: "Steam"
        target: "games_steam"
      - type: submenu
        label: "Xbox"
        target: "games_xbox"
      - type: separator
      - type: back
        label: "Back"
  games_steam:
    title: "Steam"
    items:
      - type: command
        label: "Half-Life 2"
        exec:
          windows: "start steam://rungameid/220"
      - type: separator
      - type: back
        label: "Back"
  games_xbox:
    title: "Xbox"
    items:
      - type: command
        label: "Minecraft"
        exec:
          windows: "start shell:AppsFolder\\Microsoft.MinecraftUWP_8wekyb3d8bbwe!App"
      - type: separator
      - type: back
        label: "Back"
```

### Single-source example (Games from Steam only)

When only one source contributes to a category, no sub-menus are created:

```yaml
menus:
  games:
    title: "Games"
    items:
      - type: command
        label: "Half-Life 2"
        exec:
          windows: "start steam://rungameid/220"
      - type: separator
      - type: back
        label: "Back"
```

## Base Config Merge

When `--base` is used, the specified config file acts as the foundation. Discovered
apps are merged in, with the base config taking priority on all conflicts:

| Config element | Merge behaviour |
|---|---|
| `title` | Base wins if set; otherwise uses generated title |
| `theme` | Base wins if set |
| `themes` | Merged by name — base themes win per-key, generated themes fill gaps |
| Root `items` | Base items preserved in order. Generated category submenu entries inserted before the trailing separator/back block, skipping duplicates by target |
| `menus` | Merged by key — base menus kept untouched, generated menus added for new keys only |
| Other fields (`mouse_support`, `initial_menu`, `splash_screen`) | Base values preserved |

### Example

Given a base config with custom scripts and a separator/quit block:

```yaml
title: "My Launcher"
items:
  - type: command
    label: "Open Terminal"
    exec:
      windows: "wt.exe"
  - type: submenu
    label: "My Scripts"
    target: "scripts"
  - type: separator
  - type: back
    label: "Quit"
menus:
  scripts:
    title: "My Scripts"
    items:
      - type: command
        label: "Deploy"
        exec:
          windows: "deploy.bat"
      - type: back
        label: "Back"
```

After `menuworks generate --base myconfig.yaml --output merged.yaml`, discovered
category submenus (e.g. Games, Applications) are inserted before the separator,
and corresponding generated menus are added. The base title, items, and menus
are untouched.

### Idempotency

Running the merge again with the same base and discovered apps produces identical
output. Using the merged output as the new base also works — already-present menus
and submenu entries are skipped.

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
