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

func TestNextSelectable(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "First", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "separator"},
			{Type: "command", Label: "Second", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "command", Label: "Third", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
	}

	nav := NewNavigator(cfg)

	// Start at first item (index 0)
	if nav.GetSelectionIndex() != 0 {
		t.Fatalf("expected initial selection at 0, got %d", nav.GetSelectionIndex())
	}

	// Next should skip separator (index 1) and land on index 2
	nav.NextSelectable()
	if nav.GetSelectionIndex() != 2 {
		t.Fatalf("expected selection at 2 after NextSelectable, got %d", nav.GetSelectionIndex())
	}

	// Next again should go to index 3
	nav.NextSelectable()
	if nav.GetSelectionIndex() != 3 {
		t.Fatalf("expected selection at 3, got %d", nav.GetSelectionIndex())
	}

	// Next should wrap around to index 0
	nav.NextSelectable()
	if nav.GetSelectionIndex() != 0 {
		t.Fatalf("expected selection to wrap to 0, got %d", nav.GetSelectionIndex())
	}
}

func TestPrevSelectable(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "First", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "separator"},
			{Type: "command", Label: "Second", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
			{Type: "command", Label: "Third", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
	}

	nav := NewNavigator(cfg)

	// Start at first item (index 0)
	// Prev should wrap to last item (index 3)
	nav.PrevSelectable()
	if nav.GetSelectionIndex() != 3 {
		t.Fatalf("expected selection to wrap to 3, got %d", nav.GetSelectionIndex())
	}

	// Prev should go to index 2
	nav.PrevSelectable()
	if nav.GetSelectionIndex() != 2 {
		t.Fatalf("expected selection at 2, got %d", nav.GetSelectionIndex())
	}

	// Prev should skip separator (index 1) and land on index 0
	nav.PrevSelectable()
	if nav.GetSelectionIndex() != 0 {
		t.Fatalf("expected selection at 0 after skipping separator, got %d", nav.GetSelectionIndex())
	}
}

func TestBackFromSubmenu(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "submenu", Label: "Tools", Target: "tools"},
		},
		Menus: map[string]config.Menu{
			"tools": {
				Title: "Tools",
				Items: []config.MenuItem{
					{Type: "command", Label: "Date", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
				},
			},
		},
	}

	nav := NewNavigator(cfg)

	// Should start at root
	if !nav.IsAtRoot() {
		t.Fatalf("expected to be at root")
	}

	// Open submenu
	if err := nav.Open(); err != nil {
		t.Fatalf("unexpected error opening submenu: %v", err)
	}
	if nav.IsAtRoot() {
		t.Fatalf("expected to not be at root after opening submenu")
	}
	if nav.GetCurrentMenuName() != "tools" {
		t.Fatalf("expected current menu to be 'tools', got %q", nav.GetCurrentMenuName())
	}

	// Back should return to root
	nav.Back()
	if !nav.IsAtRoot() {
		t.Fatalf("expected to be at root after Back()")
	}
}

func TestBackAtRootStaysAtRoot(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "command", Label: "Test", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
	}

	nav := NewNavigator(cfg)
	if !nav.IsAtRoot() {
		t.Fatalf("expected to be at root")
	}

	// Back at root should stay at root (not panic or crash)
	nav.Back()
	if !nav.IsAtRoot() {
		t.Fatalf("expected to still be at root after Back() at root")
	}
}

func TestOpenDisabledSubmenu(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "submenu", Label: "Missing", Target: "nonexistent"},
		},
		Menus: map[string]config.Menu{},
	}

	nav := NewNavigator(cfg)

	// Opening a submenu with missing target should return error
	err := nav.Open()
	if err == nil {
		t.Fatalf("expected error opening disabled submenu, got nil")
	}

	// Should still be at root
	if !nav.IsAtRoot() {
		t.Fatalf("expected to still be at root after failed Open()")
	}
}

func TestNavigationPreservesSelectionAcrossMenus(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "submenu", Label: "Tools", Target: "tools"},
			{Type: "command", Label: "Second", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
		},
		Menus: map[string]config.Menu{
			"tools": {
				Title: "Tools",
				Items: []config.MenuItem{
					{Type: "command", Label: "A", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
					{Type: "command", Label: "B", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
					{Type: "command", Label: "C", Exec: config.ExecConfig{Windows: "echo", Linux: "echo", Mac: "echo"}},
				},
			},
		},
	}

	nav := NewNavigator(cfg)

	// Move to second item in root
	nav.NextSelectable()
	if nav.GetSelectionIndex() != 1 {
		t.Fatalf("expected root selection at 1, got %d", nav.GetSelectionIndex())
	}

	// Go back to first item and open submenu
	nav.PrevSelectable()
	nav.Open()

	// Move to third item in submenu
	nav.NextSelectable()
	nav.NextSelectable()
	if nav.GetSelectionIndex() != 2 {
		t.Fatalf("expected tools selection at 2, got %d", nav.GetSelectionIndex())
	}

	// Go back to root
	nav.Back()

	// Root selection should still be at 0 (where we left it)
	if nav.GetSelectionIndex() != 0 {
		t.Fatalf("expected root selection preserved at 0, got %d", nav.GetSelectionIndex())
	}

	// Re-enter submenu — selection should be remembered at 2
	nav.Open()
	if nav.GetSelectionIndex() != 2 {
		t.Fatalf("expected tools selection remembered at 2, got %d", nav.GetSelectionIndex())
	}
}

func TestNavigateToMenu(t *testing.T) {
	cfg := &config.Config{
		Title: "Root",
		Items: []config.MenuItem{
			{Type: "submenu", Label: "Games", Target: "games"},
			{Type: "back", Label: "Quit"},
		},
		Menus: map[string]config.Menu{
			"games": {
				Title: "Games",
				Items: []config.MenuItem{
					{Type: "command", Label: "Doom", Exec: config.ExecConfig{Windows: "echo doom"}},
					{Type: "back", Label: "Back"},
				},
			},
		},
	}

	// Valid menu — should navigate
	nav := NewNavigator(cfg)
	if !nav.NavigateToMenu("games") {
		t.Fatal("expected NavigateToMenu to return true for existing menu")
	}
	if nav.GetCurrentMenuName() != "games" {
		t.Fatalf("expected current menu 'games', got '%s'", nav.GetCurrentMenuName())
	}
	if nav.IsAtRoot() {
		t.Fatal("expected not at root after NavigateToMenu")
	}

	// Back should return to root
	nav.Back()
	if !nav.IsAtRoot() {
		t.Fatal("expected at root after Back")
	}

	// Invalid menu — should return false, stay at root
	nav2 := NewNavigator(cfg)
	if nav2.NavigateToMenu("nonexistent") {
		t.Fatal("expected NavigateToMenu to return false for nonexistent menu")
	}
	if !nav2.IsAtRoot() {
		t.Fatal("expected to stay at root when menu not found")
	}

	// Empty string — should return true (root)
	nav3 := NewNavigator(cfg)
	if !nav3.NavigateToMenu("") {
		t.Fatal("expected NavigateToMenu to return true for empty string")
	}
	if !nav3.IsAtRoot() {
		t.Fatal("expected to stay at root for empty string")
	}

	// "root" — should return true
	nav4 := NewNavigator(cfg)
	if !nav4.NavigateToMenu("root") {
		t.Fatal("expected NavigateToMenu to return true for 'root'")
	}
	if !nav4.IsAtRoot() {
		t.Fatal("expected to stay at root for 'root'")
	}
}
