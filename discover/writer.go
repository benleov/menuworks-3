package discover

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// writerOS is the OS identifier to use for exec commands in generated YAML.
// It can be overridden in tests.
var writerOS = detectOS()

func detectOS() string {
	// Use build-time GOOS detection
	switch os := strings.ToLower(getGOOS()); os {
	case "darwin":
		return "mac"
	case "linux":
		return "linux"
	default:
		return "windows"
	}
}

// yamlConfig mirrors the MenuWorks config structure for marshalling.
// This is intentionally separate from config.Config to maintain package isolation.
type yamlConfig struct {
	Title  string                  `yaml:"title"`
	Theme  string                  `yaml:"theme"`
	Themes map[string]yamlTheme    `yaml:"themes"`
	Items  []yamlItem              `yaml:"items"`
	Menus  yaml.Node               `yaml:"menus"` // use Node to preserve key order
}

type yamlTheme struct {
	Background  string `yaml:"background"`
	Text        string `yaml:"text"`
	Border      string `yaml:"border"`
	HighlightBg string `yaml:"highlight_bg"`
	HighlightFg string `yaml:"highlight_fg"`
	Hotkey      string `yaml:"hotkey"`
	Shadow      string `yaml:"shadow"`
	Disabled    string `yaml:"disabled"`
}

type yamlItem struct {
	Type   string    `yaml:"type"`
	Label  string    `yaml:"label,omitempty"`
	Target string    `yaml:"target,omitempty"`
	Exec   *yamlExec `yaml:"exec,omitempty"`
}

type yamlExec struct {
	Windows string `yaml:"windows,omitempty"`
	Linux   string `yaml:"linux,omitempty"`
	Mac     string `yaml:"mac,omitempty"`
}

type yamlMenu struct {
	Title string     `yaml:"title"`
	Items []yamlItem `yaml:"items"`
}

// WriteConfig generates a MenuWorks config.yaml from discovered apps and writes it to the given path.
func WriteConfig(apps []DiscoveredApp, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	return RenderConfig(apps, f)
}

// WriteConfigStdout generates a MenuWorks config.yaml and writes it to stdout.
func WriteConfigStdout(apps []DiscoveredApp) error {
	return RenderConfig(apps, os.Stdout)
}

// RenderConfig generates the config YAML from apps and writes to w.
// Uses yaml.Marshal to ensure correct escaping of all values.
func RenderConfig(apps []DiscoveredApp, w io.Writer) error {
	cfg := buildYAMLConfig(apps)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = w.Write(data)
	return err
}

// buildYAMLConfig transforms discovered apps into a marshallable config struct.
func buildYAMLConfig(apps []DiscoveredApp) yamlConfig {
	groups := GroupByCategory(apps)

	// Sort category names for deterministic output
	var catNames []string
	for name := range groups {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	osKey := writerOS

	// Build root menu items (submenu entries + separator + quit)
	var rootItems []yamlItem
	for _, name := range catNames {
		rootItems = append(rootItems, yamlItem{
			Type:   "submenu",
			Label:  name,
			Target: sanitizeID(name),
		})
	}
	rootItems = append(rootItems, yamlItem{Type: "separator"})
	rootItems = append(rootItems, yamlItem{Type: "back", Label: "Quit"})

	// Build menus as an ordered yaml.Node to preserve category order
	menusNode := yaml.Node{Kind: yaml.MappingNode}
	for _, name := range catNames {
		catApps := groups[name]

		// Check if this category has apps from multiple sources
		sourceGroups := GroupBySource(catApps)

		if len(sourceGroups) > 1 {
			// Multiple sources: create sub-menus per source
			buildMultiSourceMenus(name, sourceGroups, osKey, &menusNode)
		} else {
			// Single source (or no source): flat list of commands
			buildFlatMenu(name, catApps, osKey, &menusNode)
		}
	}

	return yamlConfig{
		Title: "MenuWorks 3.X",
		Theme: "dark",
		Themes: map[string]yamlTheme{
			"dark": {
				Background:  "blue",
				Text:        "silver",
				Border:      "aqua",
				HighlightBg: "navy",
				HighlightFg: "white",
				Hotkey:      "yellow",
				Shadow:      "gray",
				Disabled:    "gray",
			},
		},
		Items: rootItems,
		Menus: menusNode,
	}
}

// buildFlatMenu adds a single category menu with command items directly listed.
func buildFlatMenu(category string, apps []DiscoveredApp, osKey string, menusNode *yaml.Node) {
	var menuItems []yamlItem
	for _, a := range apps {
		item := yamlItem{
			Type:  "command",
			Label: a.Name,
			Exec:  &yamlExec{},
		}
		setExecOS(item.Exec, osKey, a.Exec)
		menuItems = append(menuItems, item)
	}
	menuItems = append(menuItems, yamlItem{Type: "back", Label: "Back"})

	menu := yamlMenu{
		Title: category,
		Items: menuItems,
	}

	var menuNode yaml.Node
	if err := menuNode.Encode(menu); err != nil {
		return
	}
	keyNode := yaml.Node{Kind: yaml.ScalarNode, Value: sanitizeID(category)}
	menusNode.Content = append(menusNode.Content, &keyNode, &menuNode)
}

// buildMultiSourceMenus adds a category menu that contains submenu items per source,
// and the individual source sub-menus with the actual commands.
// For example, category "Games" with sources "steam" and "xbox" produces:
//
//	games:       submenu -> games_steam, submenu -> games_xbox, Back
//	games_steam: command items..., Back
//	games_xbox:  command items..., Back
func buildMultiSourceMenus(category string, sourceGroups map[string][]DiscoveredApp, osKey string, menusNode *yaml.Node) {
	// Sort source names for deterministic output
	var sourceNames []string
	for src := range sourceGroups {
		sourceNames = append(sourceNames, src)
	}
	sort.Strings(sourceNames)

	// Build the parent category menu with submenu entries per source
	var catItems []yamlItem
	for _, src := range sourceNames {
		subID := sanitizeID(category + "_" + src)
		catItems = append(catItems, yamlItem{
			Type:   "submenu",
			Label:  titleCase(src),
			Target: subID,
		})
	}
	catItems = append(catItems, yamlItem{Type: "back", Label: "Back"})

	catMenu := yamlMenu{
		Title: category,
		Items: catItems,
	}

	var catNode yaml.Node
	if err := catNode.Encode(catMenu); err != nil {
		return
	}
	catKey := yaml.Node{Kind: yaml.ScalarNode, Value: sanitizeID(category)}
	menusNode.Content = append(menusNode.Content, &catKey, &catNode)

	// Build individual source sub-menus
	for _, src := range sourceNames {
		apps := sourceGroups[src]
		subID := sanitizeID(category + "_" + src)

		var subItems []yamlItem
		for _, a := range apps {
			item := yamlItem{
				Type:  "command",
				Label: a.Name,
				Exec:  &yamlExec{},
			}
			setExecOS(item.Exec, osKey, a.Exec)
			subItems = append(subItems, item)
		}
		subItems = append(subItems, yamlItem{Type: "back", Label: "Back"})

		subMenu := yamlMenu{
			Title: titleCase(src),
			Items: subItems,
		}

		var subNode yaml.Node
		if err := subNode.Encode(subMenu); err != nil {
			continue
		}
		subKey := yaml.Node{Kind: yaml.ScalarNode, Value: subID}
		menusNode.Content = append(menusNode.Content, &subKey, &subNode)
	}
}

// setExecOS sets the appropriate OS field on a yamlExec struct.
func setExecOS(e *yamlExec, osKey, cmd string) {
	switch osKey {
	case "windows":
		e.Windows = cmd
	case "linux":
		e.Linux = cmd
	case "mac":
		e.Mac = cmd
	}
}

// titleCase capitalises the first letter of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// sanitizeID converts a display name to a YAML-safe menu ID.
// e.g. "System Tools" -> "system_tools"
func sanitizeID(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, s)
	// Collapse multiple underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	return s
}
