package menu

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/benworks/menuworks/config"
)

// Navigator manages menu navigation state and selection memory
type Navigator struct {
	cfg              *config.Config
	menuPath         []string           // Stack of menu names, e.g., ["root", "system"]
	selectionIndex   map[string]int    // Remembers selection index for each menu
	disabledItems    map[string]bool   // Tracks disabled submenu key names (e.g., "system:target_name")
	errorReported    map[string]bool   // Track which missing targets have been reported
	hotkeyMap        map[string]map[string]int // hotkeyMap[menuName][hotkey] = itemIndex
}

// NewNavigator creates a new Navigator from a config
func NewNavigator(cfg *config.Config) *Navigator {
	nav := &Navigator{
		cfg:            cfg,
		menuPath:       []string{"root"},
		selectionIndex: make(map[string]int),
		disabledItems:  make(map[string]bool),
		errorReported:  make(map[string]bool),
		hotkeyMap:      make(map[string]map[string]int),
	}

	// Build hotkey maps for all menus
	nav.buildHotkeys("root", cfg.Items)
	if cfg.Menus != nil {
		for name, menu := range cfg.Menus {
			nav.buildHotkeys(name, menu.Items)
		}
	}

	// Validate submenu targets and mark disabled items
	nav.validateTargets()

	// Initialize selection to first selectable item
	nav.selectionIndex["root"] = nav.firstSelectableIndex("root")

	return nav
}

// buildHotkeys builds hotkey map for a menu
func (n *Navigator) buildHotkeys(menuName string, items []config.MenuItem) {
	n.hotkeyMap[menuName] = make(map[string]int)
	usedHotkeys := make(map[string]bool)

	// First pass: mark explicitly defined hotkeys
	for i, item := range items {
		if item.Hotkey != "" {
			hotkey := strings.ToUpper(item.Hotkey)
			n.hotkeyMap[menuName][hotkey] = i
			usedHotkeys[hotkey] = true
		}
	}

	// Second pass: auto-assign hotkeys
	for i, item := range items {
		if item.Type == "separator" {
			continue
		}
		if item.Hotkey != "" {
			// Already explicitly set
			continue
		}

		// Scan label left-to-right for first available letter
		for _, ch := range item.Label {
			if unicode.IsLetter(ch) {
				hotkey := strings.ToUpper(string(ch))
				if !usedHotkeys[hotkey] {
					n.hotkeyMap[menuName][hotkey] = i
					usedHotkeys[hotkey] = true
					break
				}
			}
		}
	}
}

// validateTargets checks that all submenu targets exist and marks disabled items
func (n *Navigator) validateTargets() {
	n.checkMenuTargets("root", n.cfg.Items)
	if n.cfg.Menus != nil {
		for name, menu := range n.cfg.Menus {
			n.checkMenuTargets(name, menu.Items)
		}
	}
}

// checkMenuTargets checks targets in a menu's items
func (n *Navigator) checkMenuTargets(menuName string, items []config.MenuItem) {
	osType := getOSType()
	for i, item := range items {
		if item.Type == "submenu" {
			if n.cfg.Menus == nil {
				// Target doesn't exist - mark as disabled
				disabledKey := fmt.Sprintf("%s:%d", menuName, i)
				n.disabledItems[disabledKey] = true
			} else if _, exists := n.cfg.Menus[item.Target]; !exists {
				// Target doesn't exist in menus map - mark as disabled
				disabledKey := fmt.Sprintf("%s:%d", menuName, i)
				n.disabledItems[disabledKey] = true
			}
		} else if item.Type == "command" {
			// Check if command has a variant for the current OS
			if item.Exec.CommandForOS(osType) == "" {
				// No variant for this OS - mark as disabled
				disabledKey := fmt.Sprintf("%s:%d", menuName, i)
				n.disabledItems[disabledKey] = true
			}
		}
	}
}

// getOSType returns the current OS type string
func getOSType() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	default:
		return runtime.GOOS
	}
}

// GetCurrentMenu returns the current menu items
func (n *Navigator) GetCurrentMenu() []config.MenuItem {
	if len(n.menuPath) == 0 || n.menuPath[len(n.menuPath)-1] == "root" {
		return n.cfg.Items
	}

	menuName := n.menuPath[len(n.menuPath)-1]
	if n.cfg.Menus != nil {
		if menu, exists := n.cfg.Menus[menuName]; exists {
			return menu.Items
		}
	}
	return n.cfg.Items
}

// GetCurrentMenuName returns the name of the current menu
func (n *Navigator) GetCurrentMenuName() string {
	if len(n.menuPath) == 0 {
		return "root"
	}
	return n.menuPath[len(n.menuPath)-1]
}

// GetCurrentMenuTitle returns the title of the current menu
func (n *Navigator) GetCurrentMenuTitle() string {
	menuName := n.GetCurrentMenuName()
	if menuName == "root" {
		return n.cfg.Title
	}

	if n.cfg.Menus != nil {
		if menu, exists := n.cfg.Menus[menuName]; exists {
			return menu.Title
		}
	}
	return ""
}

// GetSelectionIndex returns the current selection index
func (n *Navigator) GetSelectionIndex() int {
	menuName := n.GetCurrentMenuName()
	if idx, exists := n.selectionIndex[menuName]; exists {
		return idx
	}
	return 0
}

// SetSelectionIndex sets the current selection index
func (n *Navigator) SetSelectionIndex(idx int) {
	menuName := n.GetCurrentMenuName()
	n.selectionIndex[menuName] = idx
}

// IsItemDisabled checks if an item is disabled (submenu with missing target)
func (n *Navigator) IsItemDisabled(itemIndex int) bool {
	menuName := n.GetCurrentMenuName()
	disabledKey := fmt.Sprintf("%s:%d", menuName, itemIndex)
	return n.disabledItems[disabledKey]
}

// IsTargetErrorReported checks if a missing target error has been reported
func (n *Navigator) IsTargetErrorReported(menuName string) bool {
	return n.errorReported[menuName]
}

// MarkTargetErrorReported marks a menu as having reported a missing target error
func (n *Navigator) MarkTargetErrorReported(menuName string) {
	n.errorReported[menuName] = true
}

// firstSelectableIndex returns the index of the first selectable item (not separator)
func (n *Navigator) firstSelectableIndex(menuName string) int {
	var items []config.MenuItem
	if menuName == "root" {
		items = n.cfg.Items
	} else if n.cfg.Menus != nil {
		if menu, exists := n.cfg.Menus[menuName]; exists {
			items = menu.Items
		}
	}

	for i, item := range items {
		if item.Type != "separator" {
			return i
		}
	}
	return 0
}

// NextSelectable moves to next non-separator item
func (n *Navigator) NextSelectable() {
	items := n.GetCurrentMenu()
	currentIdx := n.GetSelectionIndex()
	nextIdx := currentIdx + 1

	// Wrap around
	if nextIdx >= len(items) {
		nextIdx = 0
	}

	// Skip separators
	for i := 0; i < len(items); i++ {
		idx := (nextIdx + i) % len(items)
		if items[idx].Type != "separator" {
			n.SetSelectionIndex(idx)
			return
		}
	}

	// Fallback: stay at current
	n.SetSelectionIndex(currentIdx)
}

// PrevSelectable moves to previous non-separator item
func (n *Navigator) PrevSelectable() {
	items := n.GetCurrentMenu()
	currentIdx := n.GetSelectionIndex()
	prevIdx := currentIdx - 1

	// Wrap around
	if prevIdx < 0 {
		prevIdx = len(items) - 1
	}

	// Skip separators
	for i := 0; i < len(items); i++ {
		idx := (prevIdx - i) % len(items)
		if idx < 0 {
			idx = len(items) + idx
		}
		if items[idx].Type != "separator" {
			n.SetSelectionIndex(idx)
			return
		}
	}

	// Fallback: stay at current
	n.SetSelectionIndex(currentIdx)
}

// GetSelectedItem returns the currently selected item
func (n *Navigator) GetSelectedItem() (config.MenuItem, error) {
	items := n.GetCurrentMenu()
	idx := n.GetSelectionIndex()
	if idx < 0 || idx >= len(items) {
		return config.MenuItem{}, fmt.Errorf("invalid selection index")
	}
	return items[idx], nil
}

// SelectItemByHotkey returns the item index matching a hotkey, or -1 if not found
func (n *Navigator) SelectItemByHotkey(hotkey string) int {
	menuName := n.GetCurrentMenuName()
	hotkeyUpper := strings.ToUpper(hotkey)
	if idx, exists := n.hotkeyMap[menuName][hotkeyUpper]; exists {
		// Don't move selection if disabled
		if !n.IsItemDisabled(idx) {
			return idx
		}
	}
	return -1
}

// Open opens a submenu (moves to submenu if target exists)
func (n *Navigator) Open() error {
	item, err := n.GetSelectedItem()
	if err != nil {
		return err
	}

	if item.Type != "submenu" {
		return fmt.Errorf("item is not a submenu")
	}

	// Check if target is disabled
	currentIdx := n.GetSelectionIndex()
	if n.IsItemDisabled(currentIdx) {
		return fmt.Errorf("submenu target '%s' not found", item.Target)
	}

	// Push menu to path
	n.menuPath = append(n.menuPath, item.Target)

	// Initialize selection for this menu if not already set
	if _, exists := n.selectionIndex[item.Target]; !exists {
		n.selectionIndex[item.Target] = n.firstSelectableIndex(item.Target)
	}

	return nil
}

// Back returns to parent menu
func (n *Navigator) Back() {
	if len(n.menuPath) > 1 {
		n.menuPath = n.menuPath[:len(n.menuPath)-1]
	}
}

// IsAtRoot returns true if at root menu
func (n *Navigator) IsAtRoot() bool {
	return len(n.menuPath) == 1 && n.menuPath[0] == "root"
}

// RememberSelection stores the current selection for recovery after reload
func (n *Navigator) RememberSelection() map[string]int {
	return n.selectionIndex
}

// RecallSelection applies stored selection
func (n *Navigator) RecallSelection(remembered map[string]int) {
	for menuName, idx := range remembered {
		n.selectionIndex[menuName] = idx
	}
}
