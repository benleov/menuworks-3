package discover

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func init() {
	// Override OS for deterministic test output
	writerOS = "windows"
}

// --- MergeWithBase Tests ---

func TestMergeWithBaseEmptyBase(t *testing.T) {
	// Empty base should produce output equivalent to normal generate
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte("{}"), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if cfg.Title != "MenuWorks 3.X" {
		t.Errorf("expected generated title, got %q", cfg.Title)
	}
	if len(cfg.Menus) == 0 {
		t.Error("expected generated menus")
	}
}

func TestMergeWithBasePreservesTitle(t *testing.T) {
	base := `title: "My Custom Menu"`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if cfg.Title != "My Custom Menu" {
		t.Errorf("expected base title 'My Custom Menu', got %q", cfg.Title)
	}
}

func TestMergeWithBasePreservesTheme(t *testing.T) {
	base := `
title: "Test"
theme: "custom"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if cfg.Theme != "custom" {
		t.Errorf("expected base theme 'custom', got %q", cfg.Theme)
	}
}

func TestMergeWithBasePreservesBaseThemes(t *testing.T) {
	base := `
title: "Test"
themes:
  dark:
    background: "black"
    text: "white"
    border: "red"
    highlight_bg: "gray"
    highlight_fg: "white"
    hotkey: "green"
    shadow: "black"
    disabled: "gray"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Base "dark" theme should be preserved (not overwritten by generated "dark")
	dark, exists := cfg.Themes["dark"]
	if !exists {
		t.Fatal("expected 'dark' theme to exist")
	}
	if dark.Background != "black" {
		t.Errorf("expected base dark.background 'black', got %q", dark.Background)
	}
	if dark.Border != "red" {
		t.Errorf("expected base dark.border 'red', got %q", dark.Border)
	}
}

func TestMergeWithBaseAddsNewMenus(t *testing.T) {
	base := `
title: "Test"
items:
  - type: submenu
    label: "My Scripts"
    target: "scripts"
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
`
	apps := []DiscoveredApp{
		{Name: "Game1", Exec: "game1.exe", Source: "steam", Category: "Games"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Base menu should be preserved
	if _, exists := cfg.Menus["scripts"]; !exists {
		t.Error("expected base menu 'scripts' to be preserved")
	}

	// Generated menu should be added
	if _, exists := cfg.Menus["games"]; !exists {
		t.Error("expected generated menu 'games' to be added")
	}
}

func TestMergeWithBasePreservesExistingMenus(t *testing.T) {
	base := `
title: "Test"
items:
  - type: submenu
    label: "Games"
    target: "games"
  - type: back
    label: "Quit"
menus:
  games:
    title: "My Hand-Picked Games"
    items:
      - type: command
        label: "Custom Game"
        exec:
          windows: "custom.exe"
      - type: back
        label: "Back"
`
	apps := []DiscoveredApp{
		{Name: "Discovered Game", Exec: "disc.exe", Source: "steam", Category: "Games"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Base "games" menu should be untouched
	games := cfg.Menus["games"]
	if games.Title != "My Hand-Picked Games" {
		t.Errorf("expected base games title, got %q", games.Title)
	}
	if len(games.Items) != 2 {
		t.Errorf("expected 2 items in base games menu, got %d", len(games.Items))
	}
	if games.Items[0].Label != "Custom Game" {
		t.Errorf("expected 'Custom Game', got %q", games.Items[0].Label)
	}
}

func TestMergeWithBaseInsertsItemsBeforeTrailingBlock(t *testing.T) {
	base := `
title: "Test"
items:
  - type: submenu
    label: "My Scripts"
    target: "scripts"
  - type: separator
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Expected order: My Scripts, Tools (generated), separator, Quit
	if len(cfg.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(cfg.Items))
	}
	if cfg.Items[0].Label != "My Scripts" {
		t.Errorf("item 0: expected 'My Scripts', got %q", cfg.Items[0].Label)
	}
	if cfg.Items[1].Label != "Tools" {
		t.Errorf("item 1: expected 'Tools' (generated), got %q", cfg.Items[1].Label)
	}
	if cfg.Items[2].Type != "separator" {
		t.Errorf("item 2: expected separator, got %q", cfg.Items[2].Type)
	}
	if cfg.Items[3].Label != "Quit" {
		t.Errorf("item 3: expected 'Quit', got %q", cfg.Items[3].Label)
	}
}

func TestMergeWithBaseSkipsDuplicateTargets(t *testing.T) {
	base := `
title: "Test"
items:
  - type: submenu
    label: "Games"
    target: "games"
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "Game1", Exec: "game1.exe", Source: "steam", Category: "Games"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Should NOT add another "Games" submenu entry since target "games" already exists
	submenuCount := 0
	for _, item := range cfg.Items {
		if item.Type == "submenu" && item.Target == "games" {
			submenuCount++
		}
	}
	if submenuCount != 1 {
		t.Errorf("expected 1 submenu with target 'games', got %d", submenuCount)
	}
}

func TestMergeWithBasePreservesOptionalFields(t *testing.T) {
	base := `
title: "Test"
mouse_support: false
initial_menu: "tools"
splash_screen: false
items:
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if cfg.MouseSupport == nil || *cfg.MouseSupport != false {
		t.Error("expected mouse_support to be preserved as false")
	}
	if cfg.InitialMenu != "tools" {
		t.Errorf("expected initial_menu 'tools', got %q", cfg.InitialMenu)
	}
	if cfg.SplashScreen == nil || *cfg.SplashScreen != false {
		t.Error("expected splash_screen to be preserved as false")
	}
}

func TestMergeWithBasePreservesItemHotkeys(t *testing.T) {
	base := `
title: "Test"
items:
  - type: submenu
    label: "My Tools"
    hotkey: "T"
    target: "tools"
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Games"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if cfg.Items[0].Hotkey != "T" {
		t.Errorf("expected hotkey 'T' preserved, got %q", cfg.Items[0].Hotkey)
	}
}

func TestMergeWithBaseInvalidBaseYAML(t *testing.T) {
	_, err := MergeWithBase([]byte("{{invalid yaml"), nil)
	if err == nil {
		t.Fatal("expected error for invalid base YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse base config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMergeWithBaseNoApps(t *testing.T) {
	base := `
title: "My Menu"
items:
  - type: submenu
    label: "Scripts"
    target: "scripts"
  - type: back
    label: "Quit"
menus:
  scripts:
    title: "Scripts"
    items:
      - type: back
        label: "Back"
`

	result, err := MergeWithBase([]byte(base), nil)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// With no apps, base should be returned essentially unchanged
	if cfg.Title != "My Menu" {
		t.Errorf("expected 'My Menu', got %q", cfg.Title)
	}
	if len(cfg.Menus) != 1 {
		t.Errorf("expected 1 menu, got %d", len(cfg.Menus))
	}
}

func TestMergeWithBaseMultipleCategories(t *testing.T) {
	base := `
title: "Test"
items:
  - type: command
    label: "Open Terminal"
    exec:
      windows: "wt.exe"
  - type: separator
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "Game1", Exec: "game1.exe", Source: "steam", Category: "Games"},
		{Name: "App1", Exec: "app1.exe", Source: "startmenu", Category: "Applications"},
	}

	result, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("MergeWithBase failed: %v", err)
	}

	var cfg fullConfig
	if err := yaml.Unmarshal(result, &cfg); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Expected: Open Terminal, Applications (gen), Games (gen), separator, Quit
	if len(cfg.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(cfg.Items))
	}

	// First item should be the base command
	if cfg.Items[0].Label != "Open Terminal" {
		t.Errorf("item 0: expected 'Open Terminal', got %q", cfg.Items[0].Label)
	}

	// Generated submenus inserted before separator
	genLabels := []string{cfg.Items[1].Label, cfg.Items[2].Label}
	hasApps := false
	hasGames := false
	for _, l := range genLabels {
		if l == "Applications" {
			hasApps = true
		}
		if l == "Games" {
			hasGames = true
		}
	}
	if !hasApps || !hasGames {
		t.Errorf("expected Applications and Games submenu entries, got %v", genLabels)
	}

	// Trailing block preserved
	if cfg.Items[3].Type != "separator" {
		t.Errorf("item 3: expected separator, got %q", cfg.Items[3].Type)
	}
	if cfg.Items[4].Label != "Quit" {
		t.Errorf("item 4: expected 'Quit', got %q", cfg.Items[4].Label)
	}

	// Both generated menus should exist
	if _, exists := cfg.Menus["games"]; !exists {
		t.Error("expected 'games' menu")
	}
	if _, exists := cfg.Menus["applications"]; !exists {
		t.Error("expected 'applications' menu")
	}
}

// --- findInsertionPoint Tests ---

func TestFindInsertionPointTrailingBlock(t *testing.T) {
	items := []fullItem{
		{Type: "submenu", Label: "A"},
		{Type: "separator"},
		{Type: "back", Label: "Quit"},
	}
	idx := findInsertionPoint(items)
	if idx != 1 {
		t.Errorf("expected insertion at 1, got %d", idx)
	}
}

func TestFindInsertionPointNoTrailingBlock(t *testing.T) {
	items := []fullItem{
		{Type: "submenu", Label: "A"},
		{Type: "submenu", Label: "B"},
	}
	idx := findInsertionPoint(items)
	if idx != 2 {
		t.Errorf("expected insertion at end (2), got %d", idx)
	}
}

func TestFindInsertionPointEmpty(t *testing.T) {
	idx := findInsertionPoint(nil)
	if idx != 0 {
		t.Errorf("expected 0 for nil items, got %d", idx)
	}
}

func TestFindInsertionPointAllTrailing(t *testing.T) {
	items := []fullItem{
		{Type: "separator"},
		{Type: "back", Label: "Quit"},
	}
	idx := findInsertionPoint(items)
	if idx != 0 {
		t.Errorf("expected 0 (all trailing), got %d", idx)
	}
}

func TestFindInsertionPointBackOnly(t *testing.T) {
	items := []fullItem{
		{Type: "submenu", Label: "A"},
		{Type: "back", Label: "Quit"},
	}
	idx := findInsertionPoint(items)
	if idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}
}

// --- mergeThemes Tests ---

func TestMergeThemesBothNil(t *testing.T) {
	result := mergeThemes(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMergeThemesBaseNil(t *testing.T) {
	gen := map[string]yamlTheme{
		"dark": {Background: "blue"},
	}
	result := mergeThemes(nil, gen)
	if len(result) != 1 {
		t.Fatalf("expected 1 theme, got %d", len(result))
	}
	if result["dark"].Background != "blue" {
		t.Errorf("expected 'blue', got %q", result["dark"].Background)
	}
}

func TestMergeThemesGenNil(t *testing.T) {
	base := map[string]yamlTheme{
		"custom": {Background: "black"},
	}
	result := mergeThemes(base, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 theme, got %d", len(result))
	}
	if result["custom"].Background != "black" {
		t.Errorf("expected 'black', got %q", result["custom"].Background)
	}
}

func TestMergeThemesBaseWinsOnConflict(t *testing.T) {
	base := map[string]yamlTheme{
		"dark": {Background: "black", Text: "white"},
	}
	gen := map[string]yamlTheme{
		"dark": {Background: "blue", Text: "silver"},
	}
	result := mergeThemes(base, gen)
	if result["dark"].Background != "black" {
		t.Errorf("expected base 'black', got %q", result["dark"].Background)
	}
	if result["dark"].Text != "white" {
		t.Errorf("expected base 'white', got %q", result["dark"].Text)
	}
}

func TestMergeThemesAddsNew(t *testing.T) {
	base := map[string]yamlTheme{
		"custom": {Background: "black"},
	}
	gen := map[string]yamlTheme{
		"dark": {Background: "blue"},
	}
	result := mergeThemes(base, gen)
	if len(result) != 2 {
		t.Fatalf("expected 2 themes, got %d", len(result))
	}
	if result["custom"].Background != "black" {
		t.Error("base theme should be preserved")
	}
	if result["dark"].Background != "blue" {
		t.Error("generated theme should be added")
	}
}

// --- mergeMenus Tests ---

func TestMergeMenusBothNil(t *testing.T) {
	result := mergeMenus(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMergeMenusBaseWinsOnConflict(t *testing.T) {
	base := map[string]fullMenu{
		"games": {Title: "My Games", Items: []fullItem{{Type: "back", Label: "Back"}}},
	}
	gen := map[string]fullMenu{
		"games": {Title: "Games", Items: []fullItem{{Type: "command", Label: "Game1"}}},
	}
	result := mergeMenus(base, gen)
	if result["games"].Title != "My Games" {
		t.Errorf("expected base title 'My Games', got %q", result["games"].Title)
	}
}

func TestMergeMenusAddsNew(t *testing.T) {
	base := map[string]fullMenu{
		"scripts": {Title: "Scripts"},
	}
	gen := map[string]fullMenu{
		"games": {Title: "Games"},
	}
	result := mergeMenus(base, gen)
	if len(result) != 2 {
		t.Fatalf("expected 2 menus, got %d", len(result))
	}
}

// --- mergeRootItems Tests ---

func TestMergeRootItemsNoNewItems(t *testing.T) {
	base := []fullItem{
		{Type: "submenu", Label: "Games", Target: "games"},
		{Type: "back", Label: "Quit"},
	}
	gen := []fullItem{
		{Type: "submenu", Label: "Games", Target: "games"},
	}
	result := mergeRootItems(base, gen)
	if len(result) != 2 {
		t.Errorf("expected 2 items (no additions), got %d", len(result))
	}
}

func TestMergeRootItemsEmptyBase(t *testing.T) {
	gen := []fullItem{
		{Type: "submenu", Label: "Games", Target: "games"},
		{Type: "separator"},
		{Type: "back", Label: "Quit"},
	}
	result := mergeRootItems(nil, gen)
	// With nil base, new submenu entries are appended
	if len(result) != 1 {
		t.Errorf("expected 1 item (just the submenu), got %d", len(result))
	}
	if result[0].Label != "Games" {
		t.Errorf("expected 'Games', got %q", result[0].Label)
	}
}

// --- WriteMergedConfig file test ---

func TestWriteMergedConfig(t *testing.T) {
	dir := t.TempDir()
	outputPath := dir + "/output.yaml"

	base := `
title: "Base"
items:
  - type: back
    label: "Quit"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	if err := WriteMergedConfig([]byte(base), apps, outputPath); err != nil {
		t.Fatalf("WriteMergedConfig failed: %v", err)
	}

	// Verify file was written
	data, err := readFileBytes(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if !strings.Contains(string(data), "Base") {
		t.Error("expected base title in output")
	}
	if !strings.Contains(string(data), "Tools") {
		t.Error("expected generated 'Tools' menu in output")
	}
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// --- Idempotency test ---

func TestMergeWithBaseIdempotent(t *testing.T) {
	// Running merge twice with the same apps should produce the same output
	base := `
title: "Test"
items:
  - type: submenu
    label: "Scripts"
    target: "scripts"
  - type: separator
  - type: back
    label: "Quit"
menus:
  scripts:
    title: "Scripts"
    items:
      - type: back
        label: "Back"
`
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe", Source: "test", Category: "Tools"},
	}

	// First merge
	result1, err := MergeWithBase([]byte(base), apps)
	if err != nil {
		t.Fatalf("first merge failed: %v", err)
	}

	// Second merge using first result as base
	result2, err := MergeWithBase(result1, apps)
	if err != nil {
		t.Fatalf("second merge failed: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("merge is not idempotent.\nFirst:\n%s\nSecond:\n%s", result1, result2)
	}
}
