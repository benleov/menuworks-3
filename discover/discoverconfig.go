// Package discover provides application discovery for automatic config generation.
package discover

import "gopkg.in/yaml.v3"

// DirEntry specifies a single directory to scan for executable files, along with
// the display name to use as the menu section label.
//
// Exclude is an optional list of glob patterns matched against the exe filename
// (not the full path). Any exe whose name matches at least one pattern is skipped.
// Example patterns: "*64*", "setup*", "*helper.exe"
type DirEntry struct {
	Dir     string   `yaml:"dir"`
	Name    string   `yaml:"name"`
	Exclude []string `yaml:"exclude,omitempty"`
}

// DiscoverConfig holds the optional discovery configuration block from a base YAML file.
// It is read from the top-level "discover:" key and is silently ignored by the TUI at runtime.
type DiscoverConfig struct {
	Dirs []DirEntry `yaml:"dirs"`
}

// ParseDiscoverConfig extracts the "discover:" block from a YAML config file.
// It returns an empty DiscoverConfig (not an error) if the block is absent.
// All other YAML keys in the document are ignored.
func ParseDiscoverConfig(data []byte) (*DiscoverConfig, error) {
	var wrapper struct {
		Discover DiscoverConfig `yaml:"discover"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Discover, nil
}
