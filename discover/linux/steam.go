//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// SteamSource discovers games from Steam on Linux.
type SteamSource struct{}

func (s *SteamSource) Name() string     { return "steam" }
func (s *SteamSource) Category() string { return "Games" }

func (s *SteamSource) Available() bool {
	_, err := os.Stat(defaultSteamPath())
	return err == nil
}

func (s *SteamSource) Discover() ([]discover.DiscoveredApp, error) {
	steamPath := defaultSteamPath()
	libraryFolders, err := parseLibraryFolders(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		libraryFolders = []string{filepath.Join(steamPath, "steamapps")}
	}

	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, libDir := range libraryFolders {
		manifests, _ := filepath.Glob(filepath.Join(libDir, "appmanifest_*.acf"))
		for _, manifest := range manifests {
			app, err := parseAppManifest(manifest)
			if err != nil {
				continue
			}
			if seen[app.Name] {
				continue
			}
			seen[app.Name] = true
			apps = append(apps, *app)
		}
	}

	return apps, nil
}

// defaultSteamPath returns the default Steam installation directory on Linux.
func defaultSteamPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}

	// Check common Linux Steam locations
	candidates := []string{
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".local", "share", "Steam"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return candidates[0] // default fallback
}

// parseLibraryFolders parses Steam's libraryfolders.vdf to find all library paths.
func parseLibraryFolders(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return extractLibraryPaths(string(data)), nil
}

// extractLibraryPaths extracts library folder paths from VDF content.
func extractLibraryPaths(content string) []string {
	var paths []string
	pathRegex := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	matches := pathRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			steamapps := filepath.Join(m[1], "steamapps")
			if _, err := os.Stat(steamapps); err == nil {
				paths = append(paths, steamapps)
			}
		}
	}
	return paths
}

// parseAppManifest reads a Steam app manifest (.acf) and returns a DiscoveredApp.
func parseAppManifest(path string) (*discover.DiscoveredApp, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var appID, name string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if k, v := parseVDFLine(line); k != "" {
			switch k {
			case "appid":
				appID = v
			case "name":
				name = v
			}
		}
	}

	if appID == "" || name == "" {
		return nil, fmt.Errorf("incomplete manifest: %s", path)
	}

	if isSteamTool(name) {
		return nil, fmt.Errorf("filtered tool: %s", name)
	}

	return &discover.DiscoveredApp{
		Name:     name,
		Exec:     fmt.Sprintf("steam steam://rungameid/%s", appID),
		Source:   "Steam",
		Category: "Games",
	}, nil
}

// parseVDFLine extracts a key-value pair from a VDF line like: "key" "value"
func parseVDFLine(line string) (string, string) {
	parts := strings.SplitN(line, "\"", 5)
	if len(parts) < 5 {
		return "", ""
	}
	return strings.ToLower(parts[1]), parts[3]
}

// isSteamTool returns true if the name looks like a Steam tool/redistributable.
func isSteamTool(name string) bool {
	lower := strings.ToLower(name)
	toolPatterns := []string{
		"redistribut", "redist", "directx", "vcredist",
		"proton", "steamworks", "steam linux runtime",
		"steam controller", "steamvr",
	}
	for _, p := range toolPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
