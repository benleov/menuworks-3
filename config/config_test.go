package config

import (
	"os"
	"strings"
	"testing"
)

func containsAny(haystack []string, needle string) bool {
	for _, item := range haystack {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func TestValidateErrors(t *testing.T) {
	cfg := &Config{
		Title: "Root",
		Items: []MenuItem{
			{Type: "command", Label: "", Exec: ExecConfig{}},
			{Type: "submenu", Label: "Sub", Target: ""},
			{Type: "separator", Label: "-", Hotkey: "S"},
			{Type: "weird"},
		},
	}

	errs := Validate(cfg)
	if len(errs) != 5 {
		t.Fatalf("expected 5 errors, got %d: %v", len(errs), errs)
	}

	expected := []string{
		"command missing label",
		"command missing exec variant",
		"submenu missing target",
		"separator must not have label or hotkey",
		"unknown type",
	}

	for _, want := range expected {
		if !containsAny(errs, want) {
			t.Fatalf("expected error containing %q, got %v", want, errs)
		}
	}
}

func TestValidateMissingTargetNoMenus(t *testing.T) {
	cfg := &Config{
		Title: "Root",
		Items: []MenuItem{
			{Type: "submenu", Label: "Tools", Target: "tools"},
		},
		Menus: nil,
	}

	errs := Validate(cfg)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !containsAny(errs, "submenu target 'tools' not found") {
		t.Fatalf("expected missing target error, got %v", errs)
	}
}

func TestValidateMissingTargetWithMenusIgnored(t *testing.T) {
	cfg := &Config{
		Title: "Root",
		Items: []MenuItem{
			{Type: "submenu", Label: "Tools", Target: "tools"},
		},
		Menus: map[string]Menu{},
	}

	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestCommandForOS(t *testing.T) {
	exec := ExecConfig{
		Windows: "echo Hello from Windows",
		Linux:   "echo Hello from Linux",
		Mac:     "echo Hello from macOS",
	}

	tests := []struct {
		os       string
		expected string
	}{
		{"windows", "echo Hello from Windows"},
		{"linux", "echo Hello from Linux"},
		{"darwin", "echo Hello from macOS"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		result := exec.CommandForOS(tt.os)
		if result != tt.expected {
			t.Errorf("CommandForOS(%s): expected %q, got %q", tt.os, tt.expected, result)
		}
	}
}

func TestCommandForOSFallbackEmpty(t *testing.T) {
	exec := ExecConfig{
		Windows: "echo Windows only",
		Linux:   "",
		Mac:     "",
	}

	result := exec.CommandForOS("linux")
	if result != "" {
		t.Errorf("CommandForOS(linux) with empty variant: expected empty string, got %q", result)
	}
}

func TestMenuItemHelpField(t *testing.T) {
	// Test that MenuItem with Help field can be created
	item := MenuItem{
		Type:  "command",
		Label: "Test Command",
		Help:  "This is a test help message.",
		Exec: ExecConfig{
			Windows: "echo test",
		},
	}

	if item.Help != "This is a test help message." {
		t.Errorf("expected help %q, got %q", "This is a test help message.", item.Help)
	}

	if item.Type != "command" {
		t.Errorf("expected type command, got %q", item.Type)
	}

	if item.Label != "Test Command" {
		t.Errorf("expected label %q, got %q", "Test Command", item.Label)
	}
}

func TestLoadFromCustomPath(t *testing.T) {
	// Create a temp directory with a custom config
	dir := t.TempDir()
	customPath := dir + "/custom.yaml"

	yamlContent := `title: "Custom Config"
items:
  - type: command
    label: "Hello"
    exec:
      windows: "echo hello"
      linux: "echo hello"
      mac: "echo hello"
`
	if err := os.WriteFile(customPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write custom config: %v", err)
	}

	cfg, created, err := Load(customPath)
	if err != nil {
		t.Fatalf("failed to load custom config: %v", err)
	}
	if created {
		t.Fatalf("expected created=false for existing custom config")
	}
	if cfg.Title != "Custom Config" {
		t.Errorf("expected title %q, got %q", "Custom Config", cfg.Title)
	}
	if len(cfg.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cfg.Items))
	}
	if cfg.Items[0].Label != "Hello" {
		t.Errorf("expected label %q, got %q", "Hello", cfg.Items[0].Label)
	}
}

func TestLoadCreatesDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	missingPath := dir + "/nonexistent.yaml"

	cfg, created, err := Load(missingPath)
	if err != nil {
		t.Fatalf("failed to load (should create default): %v", err)
	}
	if !created {
		t.Fatalf("expected created=true for missing config")
	}
	if cfg.Title == "" {
		t.Errorf("expected non-empty title from default config")
	}

	// Verify the file was actually created
	if _, err := os.Stat(missingPath); os.IsNotExist(err) {
		t.Errorf("expected default config file to be created at %s", missingPath)
	}
}

func TestMouseSupportConfig(t *testing.T) {
	// Test default (omitted) — should be enabled
	cfg := &Config{}
	if !cfg.IsMouseEnabled() {
		t.Errorf("expected mouse enabled by default when omitted")
	}

	// Test explicit true
	trueVal := true
	cfg.MouseSupport = &trueVal
	if !cfg.IsMouseEnabled() {
		t.Errorf("expected mouse enabled when set to true")
	}

	// Test explicit false
	falseVal := false
	cfg.MouseSupport = &falseVal
	if cfg.IsMouseEnabled() {
		t.Errorf("expected mouse disabled when set to false")
	}
}

func TestSplashScreenConfig(t *testing.T) {
	// Test default (omitted) — should be enabled
	cfg := &Config{}
	if !cfg.IsSplashEnabled() {
		t.Errorf("expected splash enabled by default when omitted")
	}

	// Test explicit true
	trueVal := true
	cfg.SplashScreen = &trueVal
	if !cfg.IsSplashEnabled() {
		t.Errorf("expected splash enabled when set to true")
	}

	// Test explicit false
	falseVal := false
	cfg.SplashScreen = &falseVal
	if cfg.IsSplashEnabled() {
		t.Errorf("expected splash disabled when set to false")
	}
}

func TestInitialMenuConfig(t *testing.T) {
	yamlData := `
title: "Test"
initial_menu: "games"
items:
  - type: back
    label: "Quit"
menus:
  games:
    title: "Games"
    items:
      - type: back
        label: "Back"
`
	dir := t.TempDir()
	path := dir + "/config.yaml"
	if err := os.WriteFile(path, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.InitialMenu != "games" {
		t.Errorf("expected initial_menu='games', got '%s'", cfg.InitialMenu)
	}
}

