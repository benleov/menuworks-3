package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

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

// ExecConfig holds command execution details
type ExecConfig struct {
	Command string `yaml:"command"`
	WorkDir string `yaml:"workdir,omitempty"`
}

// Menu represents a menu with a title and list of items
type Menu struct {
	Title string      `yaml:"title"`
	Items []MenuItem  `yaml:"items"`
}

// Config is the root configuration structure
type Config struct {
	Title string          `yaml:"title"`
	Items []MenuItem      `yaml:"items"`
	Menus map[string]Menu `yaml:"menus"`
}

// Load reads the config file from disk, or writes embedded default if missing
func Load(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, write embedded default
			if writeErr := WriteDefault(filePath); writeErr != nil {
				return nil, fmt.Errorf("failed to write default config: %w", writeErr)
			}
			// Parse the embedded default
			return parseYAML([]byte(defaultConfigYAML))
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return parseYAML(data)
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
		if item.Exec.Command == "" {
			errs = append(errs, fmt.Sprintf("item %d: command missing exec.command", index))
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
