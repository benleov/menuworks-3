package menu

import (
	"testing"

	"github.com/benworks/menuworks/config"
)

func TestHotkeyAutoAssignment(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "Save File", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "command", Label: "Settings", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "separator"},
			{Type: "command", Label: ">>>", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
	}

	nav := NewNavigator(cfg)

	if got := nav.SelectItemByHotkey("s"); got != 0 {
		t.Fatalf("expected hotkey S to select index 0, got %d", got)
	}
	if got := nav.SelectItemByHotkey("E"); got != 1 {
		t.Fatalf("expected hotkey E to select index 1, got %d", got)
	}
	if got := nav.SelectItemByHotkey("X"); got != -1 {
		t.Fatalf("expected hotkey X to be unassigned, got %d", got)
	}
}

func TestHotkeyDisabledSubmenu(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "submenu", Label: "Tools", Target: "tools"},
		},
		Menus: nil,
	}

	nav := NewNavigator(cfg)

	if got := nav.SelectItemByHotkey("T"); got != -1 {
		t.Fatalf("expected disabled submenu hotkey to be ignored, got %d", got)
	}
}

func TestDisabledCommandNoOSVariant(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "Linux Only", Exec: config.ExecConfig{Linux: "echo Linux"}},
			{Type: "command", Label: "Cross Platform", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
	}

	nav := NewNavigator(cfg)

	// On Windows, the first item should be disabled (Linux only)
	// The test runs on the current OS, so we check based on that
	isDisabled := nav.IsItemDisabled(0)
	// This test is OS-dependent and validates the disabled marking logic
	// On Linux/Darwin: item should not be disabled
	// On Windows: item should be disabled
	switch getOSType() {
	case "windows":
		if !isDisabled {
			t.Fatalf("expected Linux-only command to be disabled on Windows")
		}
	default:
		if isDisabled {
			t.Fatalf("expected Linux-only command to be enabled on Linux")
		}
	}

	// Second item should always be selectable
	if nav.IsItemDisabled(1) {
		t.Fatalf("expected cross-platform command to not be disabled")
	}
}
