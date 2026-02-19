package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var defaultConfigYAML string

// MenuItem represents a single item in a menu
type MenuItem struct {
	Type       string      `yaml:"type"`   // command, submenu, back, separator
	Label      string      `yaml:"label"`
	Hotkey     string      `yaml:"hotkey,omitempty"`
	Target     string      `yaml:"target,omitempty"`     // for submenu type
	Exec       ExecConfig  `yaml:"exec,omitempty"`       // for command type
	ShowOutput *bool       `yaml:"showOutput,omitempty"` // for command type (default: true)
}

// ExecConfig holds command execution details with OS-specific variants
type ExecConfig struct {
	Windows string `yaml:"windows,omitempty"`
	Linux   string `yaml:"linux,omitempty"`
	Mac     string `yaml:"mac,omitempty"`
	WorkDir string `yaml:"workdir,omitempty"`
}

// CommandForOS returns the command for the given OS, or empty string if not defined
func (ec ExecConfig) CommandForOS(osType string) string {
	switch osType {
	case "windows":
		return ec.Windows
	case "linux":
		return ec.Linux
	case "darwin":
		return ec.Mac
	default:
		return ""
	}
}

// Menu represents a menu with a title and list of items
type Menu struct {
	Title string      `yaml:"title"`
	Items []MenuItem  `yaml:"items"`
}

// ThemeColors defines the color scheme for the UI
type ThemeColors struct {
	Background  string `yaml:"background"`
	Text        string `yaml:"text"`
	Border      string `yaml:"border"`
	HighlightBg string `yaml:"highlight_bg"`
	HighlightFg string `yaml:"highlight_fg"`
	Hotkey      string `yaml:"hotkey"`
	Shadow      string `yaml:"shadow"`
	Disabled    string `yaml:"disabled"`
}

// Config is the root configuration structure
type Config struct {
	Title  string               `yaml:"title"`
	Items  []MenuItem           `yaml:"items"`
	Menus  map[string]Menu      `yaml:"menus"`
	Theme  string               `yaml:"theme,omitempty"`
	Themes map[string]ThemeColors `yaml:"themes,omitempty"`
}

// Load reads the config file from disk, or writes embedded default if missing
// Returns (config, wasCreated, error) where wasCreated indicates if config was just created on first run
func Load(filePath string) (*Config, bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, write embedded default
			if writeErr := WriteDefault(filePath); writeErr != nil {
				return nil, false, fmt.Errorf("failed to write default config: %w", writeErr)
			}
			// Parse the embedded default and signal that it was created
			cfg, parseErr := parseYAML([]byte(defaultConfigYAML))
			return cfg, true, parseErr
		}
		return nil, false, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, err := parseYAML(data)
	return cfg, false, err
}

// parseYAML unmarshals YAML bytes into Config struct
func parseYAML(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return &cfg, nil
}

// WriteDefault writes the embedded default config to filePath
func WriteDefault(filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filePath, []byte(defaultConfigYAML), 0644); err != nil {
		return err
	}
	return nil
}

// Validate checks for invalid targets and item types
func Validate(cfg *Config) []string {
	var errs []string

	// Check root items for valid types and targets
	for i, item := range cfg.Items {
		if err := validateItem(item, i, cfg); err != nil {
			errs = append(errs, err...)
		}
	}

	// Check submenu items
	if cfg.Menus != nil {
		for menuName, menu := range cfg.Menus {
			for i, item := range menu.Items {
				if err := validateItem(item, i, cfg); err != nil {
					// Prefix with menu name for context
					var prefixed []string
					for _, e := range err {
						prefixed = append(prefixed, fmt.Sprintf("%s: %s", menuName, e))
					}
					errs = append(errs, prefixed...)
				}
			}
		}
	}

	return errs
}

// validateItem checks a single menu item
func validateItem(item MenuItem, index int, cfg *Config) []string {
	var errs []string

	switch item.Type {
	case "command":
		if item.Label == "" {
			errs = append(errs, fmt.Sprintf("item %d: command missing label", index))
		}
		if item.Exec.Windows == "" && item.Exec.Linux == "" && item.Exec.Mac == "" {
			errs = append(errs, fmt.Sprintf("item %d: command missing exec variant (windows, linux, or mac)", index))
		}
	case "submenu":
		if item.Label == "" {
			errs = append(errs, fmt.Sprintf("item %d: submenu missing label", index))
		}
		if item.Target == "" {
			errs = append(errs, fmt.Sprintf("item %d: submenu missing target", index))
		} else if cfg.Menus == nil {
			errs = append(errs, fmt.Sprintf("item %d: submenu target '%s' not found (no menus defined)", index, item.Target))
		} else if _, exists := cfg.Menus[item.Target]; !exists {
			// Target doesn't exist - don't error here, it will be marked disabled at runtime
		}
	case "back":
		if item.Label == "" {
			errs = append(errs, fmt.Sprintf("item %d: back missing label", index))
		}
	case "separator":
		if item.Label != "" || item.Hotkey != "" {
			errs = append(errs, fmt.Sprintf("item %d: separator must not have label or hotkey", index))
		}
	default:
		errs = append(errs, fmt.Sprintf("item %d: unknown type '%s'", index, item.Type))
	}

	return errs
}

// GetDefaultConfig returns the embedded default config as a string
func GetDefaultConfig() string {
	return defaultConfigYAML
}

// ParseColorName converts a color name string to tcell.Color
// Returns the color and true if valid, otherwise returns a default color and false
func ParseColorName(name string) (tcell.Color, bool) {
	if name == "" {
		return tcell.ColorDefault, false
	}
	
	// Normalize the color name (lowercase, trim spaces)
	name = strings.ToLower(strings.TrimSpace(name))
	
	// Map of valid color names to tcell colors
	colorMap := map[string]tcell.Color{
		"black":   tcell.ColorBlack,
		"maroon":  tcell.ColorMaroon,
		"green":   tcell.ColorGreen,
		"olive":   tcell.ColorOlive,
		"navy":    tcell.ColorNavy,
		"purple":  tcell.ColorPurple,
		"teal":    tcell.ColorTeal,
		"silver":  tcell.ColorSilver,
		"gray":    tcell.ColorGray,
		"grey":    tcell.ColorGray,
		"red":     tcell.ColorRed,
		"lime":    tcell.ColorLime,
		"yellow":  tcell.ColorYellow,
		"blue":    tcell.ColorBlue,
		"fuchsia": tcell.ColorFuchsia,
		"aqua":    tcell.ColorAqua,
		"cyan":    tcell.ColorAqua,
		"white":   tcell.ColorWhite,
	}
	
	if color, ok := colorMap[name]; ok {
		return color, true
	}
	
	return tcell.ColorDefault, false
}

// ValidateTheme validates that the selected theme exists and has valid colors
// Returns a list of warning messages (not fatal errors)
func ValidateTheme(cfg *Config) []string {
	var warnings []string
	
	// If no theme is specified, that's fine (use defaults)
	if cfg.Theme == "" {
		return warnings
	}
	
	// Check if themes map exists
	if cfg.Themes == nil || len(cfg.Themes) == 0 {
		warnings = append(warnings, fmt.Sprintf("theme: selected theme '%s' but no themes defined", cfg.Theme))
		return warnings
	}
	
	// Check if selected theme exists
	theme, exists := cfg.Themes[cfg.Theme]
	if !exists {
		warnings = append(warnings, fmt.Sprintf("theme: selected theme '%s' not found in themes", cfg.Theme))
		return warnings
	}
	
	// Validate each color in the theme
	colorFields := map[string]string{
		"background":   theme.Background,
		"text":         theme.Text,
		"border":       theme.Border,
		"highlight_bg": theme.HighlightBg,
		"highlight_fg": theme.HighlightFg,
		"hotkey":       theme.Hotkey,
		"shadow":       theme.Shadow,
		"disabled":     theme.Disabled,
	}
	
	for fieldName, colorName := range colorFields {
		if colorName == "" {
			warnings = append(warnings, fmt.Sprintf("theme '%s': %s color not specified", cfg.Theme, fieldName))
			continue
		}
		if _, valid := ParseColorName(colorName); !valid {
			warnings = append(warnings, fmt.Sprintf("theme '%s': invalid color name '%s' for %s", cfg.Theme, colorName, fieldName))
		}
	}
	
	return warnings
}

// GetThemeColors returns the ThemeColors for the selected theme, or nil if none/invalid
func GetThemeColors(cfg *Config) *ThemeColors {
	if cfg.Theme == "" || cfg.Themes == nil {
		return nil
	}
	
	theme, exists := cfg.Themes[cfg.Theme]
	if !exists {
		return nil
	}
	
	return &theme
}
