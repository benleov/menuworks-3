package menu

import (
	"testing"

	"github.com/benworks/menuworks/config"
)

func TestHotkeyAutoAssignment(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "Save File", Exec: config.ExecConfig{Command: "echo"}},
			{Type: "command", Label: "Settings", Exec: config.ExecConfig{Command: "echo"}},
			{Type: "separator"},
			{Type: "command", Label: ">>>", Exec: config.ExecConfig{Command: "echo"}},
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
