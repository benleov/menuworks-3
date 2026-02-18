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
		"command missing exec.command",
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
