# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [3.1.1] - 2026-02-21

### Fixed
- Wrapped config error dialog text and split the error header onto its own line
- Added backup protection and user-facing warnings for default config recovery
- Retried config load after dialog actions instead of exiting prematurely

---

## [3.1.0] - 2026-02-19

### Added
- Initial release of MenuWorks 3.0
- Single self-contained Go binary for Windows, Linux, and macOS
- Retro DOS-style hierarchical menu TUI with 80×25 layout
- YAML configuration file for user-editable menus
- Embedded default config with fallback on first run
- Hierarchical menu chaining via `target:` field
- Menu item types: `command`, `submenu`, `back`, `separator`
- Explicit and auto-assigned hotkeys for menu items
- Pop-up dialogs for errors, confirmations, and command output viewing
- Retro UI with double-line borders, drop shadows, and VGA 16-color palette
- Customizable color themes with runtime reload
- Dynamic config reload via `R` hotkey
- Selection memory during session
- Cross-platform command execution (Windows cmd.exe, Unix sh)
- Scrollable command output viewer with pagination
- Terminal resize detection and graceful recovery
- Build scripts for all platforms (PowerShell and Bash)

### Fixed
- Proper event queue handling on macOS/Linux to prevent startup hangs
- Terminal size validation with user-friendly recovery flow
- Menu selection memory preservation across submenu traversal

---
