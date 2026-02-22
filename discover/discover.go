// Package discover provides application discovery for automatic config generation.
//
// This package is intentionally isolated from the rest of MenuWorks (config, menu,
// ui, exec). It discovers installed applications and generates YAML config directly.
package discover

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Source discovers applications from a specific location on the system.
type Source interface {
	// Name returns the identifier for this source (e.g. "steam", "startmenu").
	Name() string

	// Category returns the menu category for discovered apps (e.g. "Games", "Applications").
	Category() string

	// Available reports whether this source is present on the current system.
	Available() bool

	// Discover scans for installed applications and returns them.
	Discover() ([]DiscoveredApp, error)
}

// DiscoveredApp represents a single application found by a Source.
type DiscoveredApp struct {
	Name     string // display name (used as menu label)
	Exec     string // command to launch the application (platform-specific)
	Source   string // source that found it (e.g. "steam")
	Category string // grouping category (e.g. "Games")
}

// Registry holds all known discovery sources and orchestrates scanning.
type Registry struct {
	mu      sync.Mutex
	sources []Source
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a Source to the registry.
func (r *Registry) Register(s Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = append(r.sources, s)
}

// Sources returns all registered sources.
func (r *Registry) Sources() []Source {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Source, len(r.sources))
	copy(out, r.sources)
	return out
}

// AvailableSources returns only sources that report Available() == true.
func (r *Registry) AvailableSources() []Source {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Source
	for _, s := range r.sources {
		if s.Available() {
			out = append(out, s)
		}
	}
	return out
}

// SourceByName returns the source with the given name, or nil.
func (r *Registry) SourceByName(name string) Source {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sources {
		if strings.EqualFold(s.Name(), name) {
			return s
		}
	}
	return nil
}

// DiscoverResult holds results from a single source.
type DiscoverResult struct {
	Source string
	Apps   []DiscoveredApp
	Err    error
}

// DiscoverAll runs discovery on all available sources (or the filtered set).
// If sourceNames is non-empty, only sources whose names match (case-insensitive) are used.
func (r *Registry) DiscoverAll(sourceNames []string) ([]DiscoverResult, error) {
	sources := r.AvailableSources()

	// Filter by requested names if specified
	if len(sourceNames) > 0 {
		nameSet := make(map[string]bool)
		for _, n := range sourceNames {
			nameSet[strings.ToLower(n)] = true
		}
		var filtered []Source
		for _, s := range sources {
			if nameSet[strings.ToLower(s.Name())] {
				filtered = append(filtered, s)
			}
		}
		// Check for unknown source names
		for _, n := range sourceNames {
			found := false
			for _, s := range r.Sources() {
				if strings.EqualFold(s.Name(), n) {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown source: %s", n)
			}
		}
		sources = filtered
	}

	var results []DiscoverResult
	for _, s := range sources {
		apps, err := s.Discover()
		results = append(results, DiscoverResult{
			Source: s.Name(),
			Apps:   apps,
			Err:    err,
		})
	}
	return results, nil
}

// CollectApps gathers all successfully discovered apps from results, sorted by category then name.
func CollectApps(results []DiscoverResult) []DiscoveredApp {
	var apps []DiscoveredApp
	for _, r := range results {
		if r.Err == nil {
			apps = append(apps, r.Apps...)
		}
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Category != apps[j].Category {
			return apps[i].Category < apps[j].Category
		}
		return apps[i].Name < apps[j].Name
	})
	return apps
}

// GroupByCategory groups apps by their category name.
func GroupByCategory(apps []DiscoveredApp) map[string][]DiscoveredApp {
	groups := make(map[string][]DiscoveredApp)
	for _, a := range apps {
		groups[a.Category] = append(groups[a.Category], a)
	}
	return groups
}

// GroupBySource groups apps by their source name.
func GroupBySource(apps []DiscoveredApp) map[string][]DiscoveredApp {
	groups := make(map[string][]DiscoveredApp)
	for _, a := range apps {
		groups[a.Source] = append(groups[a.Source], a)
	}
	return groups
}

// DeduplicateApps removes duplicate apps, keeping the first occurrence.
// Deduplicates by exec command (case-insensitive) and by normalized name within the same category.
func DeduplicateApps(apps []DiscoveredApp) []DiscoveredApp {
	seenExec := make(map[string]bool)
	seenName := make(map[string]bool) // key = "category|normalizedName"
	var out []DiscoveredApp
	for _, a := range apps {
		execKey := strings.ToLower(a.Exec)
		if seenExec[execKey] {
			continue
		}
		nameKey := strings.ToLower(a.Category) + "|" + strings.ToLower(a.Name)
		if seenName[nameKey] {
			continue
		}
		seenExec[execKey] = true
		seenName[nameKey] = true
		out = append(out, a)
	}
	return out
}
