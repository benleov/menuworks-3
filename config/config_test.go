package config

import (
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

