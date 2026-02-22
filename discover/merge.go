package discover

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// fullConfig is used for merge operations. It includes all known config fields
// to preserve base config values during YAML round-trip.
type fullConfig struct {
	Title        string               `yaml:"title"`
	Theme        string               `yaml:"theme,omitempty"`
	Themes       map[string]yamlTheme `yaml:"themes,omitempty"`
	Items        []fullItem           `yaml:"items"`
	Menus        map[string]fullMenu  `yaml:"menus,omitempty"`
	MouseSupport *bool                `yaml:"mouse_support,omitempty"`
	InitialMenu  string               `yaml:"initial_menu,omitempty"`
	SplashScreen *bool                `yaml:"splash_screen,omitempty"`
}

// fullItem includes all known item fields to preserve base config values.
type fullItem struct {
	Type       string    `yaml:"type"`
	Label      string    `yaml:"label,omitempty"`
	Hotkey     string    `yaml:"hotkey,omitempty"`
	Target     string    `yaml:"target,omitempty"`
	Exec       *fullExec `yaml:"exec,omitempty"`
	ShowOutput *bool     `yaml:"showOutput,omitempty"`
	Help       string    `yaml:"help,omitempty"`
}

// fullExec includes all known exec fields.
type fullExec struct {
	Windows string `yaml:"windows,omitempty"`
	Linux   string `yaml:"linux,omitempty"`
	Mac     string `yaml:"mac,omitempty"`
	WorkDir string `yaml:"workdir,omitempty"`
}

// fullMenu includes all known menu fields.
type fullMenu struct {
	Title string     `yaml:"title"`
	Items []fullItem `yaml:"items"`
}

// MergeWithBase merges discovered apps into a base config YAML.
// The base config takes priority: its title, theme, items, and menus are preserved.
// Generated content (discovered app categories and menus) fills in the gaps.
func MergeWithBase(baseYAML []byte, apps []DiscoveredApp) ([]byte, error) {
	var base fullConfig
	if err := yaml.Unmarshal(baseYAML, &base); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	gen, err := generatedToFull(apps)
	if err != nil {
		return nil, fmt.Errorf("failed to build generated config: %w", err)
	}

	merged := mergeConfigs(base, gen)

	data, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged config: %w", err)
	}
	return data, nil
}

// RenderMergedConfig merges discovered apps with a base config and writes YAML to w.
func RenderMergedConfig(baseYAML []byte, apps []DiscoveredApp, w io.Writer) error {
	data, err := MergeWithBase(baseYAML, apps)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// WriteMergedConfig merges discovered apps with a base config and writes to the given path.
func WriteMergedConfig(baseYAML []byte, apps []DiscoveredApp, outputPath string) error {
	data, err := MergeWithBase(baseYAML, apps)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// generatedToFull converts discovered apps to a fullConfig via YAML round-trip.
// This reuses buildYAMLConfig (which uses yaml.Node for menu ordering) and
// converts the result to fullConfig (which uses maps for easy merging).
func generatedToFull(apps []DiscoveredApp) (fullConfig, error) {
	cfg := buildYAMLConfig(apps)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fullConfig{}, err
	}
	var full fullConfig
	if err := yaml.Unmarshal(data, &full); err != nil {
		return fullConfig{}, err
	}
	return full, nil
}

// mergeConfigs merges base and generated configs. Base takes priority on all conflicts.
func mergeConfigs(base, gen fullConfig) fullConfig {
	result := base

	// Scalars: base wins if non-empty
	if result.Title == "" {
		result.Title = gen.Title
	}
	if result.Theme == "" {
		result.Theme = gen.Theme
	}

	// Themes: merge by key, base wins per-key
	result.Themes = mergeThemes(base.Themes, gen.Themes)

	// Root items: insert generated submenu entries before trailing separator/back block
	result.Items = mergeRootItems(base.Items, gen.Items)

	// Menus: merge by key, base wins per-key
	result.Menus = mergeMenus(base.Menus, gen.Menus)

	// Other fields (MouseSupport, InitialMenu, SplashScreen) are preserved from base
	return result
}

// mergeThemes merges theme maps. Base themes take priority per-key.
func mergeThemes(base, gen map[string]yamlTheme) map[string]yamlTheme {
	if base == nil && gen == nil {
		return nil
	}
	result := make(map[string]yamlTheme)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range gen {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// mergeRootItems merges root menu items. Base items are preserved in order.
// Generated submenu entries for new categories are inserted before the trailing
// separator/back block in the base items.
func mergeRootItems(base, gen []fullItem) []fullItem {
	// Collect existing submenu targets in base
	existingTargets := make(map[string]bool)
	for _, item := range base {
		if item.Type == "submenu" && item.Target != "" {
			existingTargets[item.Target] = true
		}
	}

	// Collect new submenu entries from generated that don't exist in base
	var newItems []fullItem
	for _, item := range gen {
		if item.Type == "submenu" && item.Target != "" {
			if !existingTargets[item.Target] {
				newItems = append(newItems, item)
			}
		}
	}

	if len(newItems) == 0 {
		return base
	}

	// Find insertion point: before trailing separator/back block
	insertIdx := findInsertionPoint(base)

	result := make([]fullItem, 0, len(base)+len(newItems))
	result = append(result, base[:insertIdx]...)
	result = append(result, newItems...)
	result = append(result, base[insertIdx:]...)

	return result
}

// findInsertionPoint returns the index where new items should be inserted,
// which is just before the trailing block of separator/back items.
func findInsertionPoint(items []fullItem) int {
	idx := len(items)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Type == "separator" || items[i].Type == "back" {
			idx = i
		} else {
			break
		}
	}
	return idx
}

// mergeMenus merges menu maps. Base menus take priority per-key.
func mergeMenus(base, gen map[string]fullMenu) map[string]fullMenu {
	if base == nil && gen == nil {
		return nil
	}
	result := make(map[string]fullMenu)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range gen {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
