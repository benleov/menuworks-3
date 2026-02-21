package discover

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

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
		var menuItems []yamlItem
		for _, a := range catApps {
			item := yamlItem{
				Type:  "command",
				Label: a.Name,
				Exec:  &yamlExec{},
			}
			switch osKey {
			case "windows":
				item.Exec.Windows = a.Exec
			case "linux":
				item.Exec.Linux = a.Exec
			case "mac":
				item.Exec.Mac = a.Exec
			}
			menuItems = append(menuItems, item)
		}
		menuItems = append(menuItems, yamlItem{Type: "back", Label: "Back"})

		menu := yamlMenu{
			Title: name,
			Items: menuItems,
		}

		// Marshal the menu value to a node
		var menuNode yaml.Node
		if err := menuNode.Encode(menu); err != nil {
			continue
		}

		// Add key and value nodes
		keyNode := yaml.Node{Kind: yaml.ScalarNode, Value: sanitizeID(name)}
		menusNode.Content = append(menusNode.Content, &keyNode, &menuNode)
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
