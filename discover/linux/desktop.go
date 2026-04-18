//go:build linux

package linux

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// DesktopSource discovers applications from XDG .desktop files.
type DesktopSource struct{}

func (s *DesktopSource) Name() string     { return "desktop" }
func (s *DesktopSource) Category() string { return "Applications" }

func (s *DesktopSource) Available() bool {
	for _, dir := range desktopDirs() {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

func (s *DesktopSource) Discover() ([]discover.DiscoveredApp, error) {
	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, dir := range desktopDirs() {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}

			app, err := parseDesktopFile(filepath.Join(dir, entry.Name()))
			if err != nil || app == nil {
				continue
			}

			key := strings.ToLower(app.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			apps = append(apps, *app)
		}
	}

	return apps, nil
}

// desktopDirs returns the XDG application directories to scan.
func desktopDirs() []string {
	dirs := []string{"/usr/share/applications", "/usr/local/share/applications"}

	// User-local .desktop files
	home := os.Getenv("HOME")
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}

	// XDG_DATA_DIRS override
	if xdgDirs := os.Getenv("XDG_DATA_DIRS"); xdgDirs != "" {
		for _, d := range strings.Split(xdgDirs, ":") {
			appDir := filepath.Join(d, "applications")
			dirs = append(dirs, appDir)
		}
	}

	return dirs
}

// parseDesktopFile reads an XDG .desktop file and returns a DiscoveredApp.
// Returns nil if the entry should be skipped (NoDisplay, Hidden, not Application type).
func parseDesktopFile(path string) (*discover.DiscoveredApp, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseDesktopReader(bufio.NewScanner(f))
}

// parseDesktopReader parses .desktop content from a scanner.
// Exported for testing.
func parseDesktopReader(scanner *bufio.Scanner) (*discover.DiscoveredApp, error) {
	var name, execCmd string
	var noDisplay, hidden, terminal bool
	entryType := ""
	inDesktopEntry := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Track section headers
		if strings.HasPrefix(line, "[") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}

		// Only process [Desktop Entry] section
		if !inDesktopEntry {
			continue
		}

		key, value := parseDesktopLine(line)
		switch key {
		case "Name":
			if name == "" { // take first Name (not localized variants)
				name = value
			}
		case "Exec":
			execCmd = value
		case "Type":
			entryType = value
		case "NoDisplay":
			noDisplay = strings.ToLower(value) == "true"
		case "Hidden":
			hidden = strings.ToLower(value) == "true"
		case "Terminal":
			terminal = strings.ToLower(value) == "true"
		}
	}

	// Filter out non-application entries
	if entryType != "Application" {
		return nil, nil
	}

	// Skip hidden/NoDisplay entries
	if noDisplay || hidden {
		return nil, nil
	}

	// Skip terminal apps (they don't work well from a TUI)
	if terminal {
		return nil, nil
	}

	if name == "" || execCmd == "" {
		return nil, nil
	}

	// Clean up Exec line: remove field codes (%f, %F, %u, %U, etc.)
	execCmd = cleanExecLine(execCmd)

	return &discover.DiscoveredApp{
		Name:     name,
		Exec:     execCmd,
		Source:   "Desktop",
		Category: "Applications",
	}, nil
}

// parseDesktopLine splits a desktop file line into key=value.
func parseDesktopLine(line string) (string, string) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", ""
	}
	// Handle localized keys like Name[fr] — skip them
	key := line[:idx]
	if strings.ContainsRune(key, '[') {
		return "", ""
	}
	return key, line[idx+1:]
}

// cleanExecLine removes XDG field codes from an Exec value.
func cleanExecLine(exec string) string {
	// Remove %f, %F, %u, %U, %d, %D, %n, %N, %i, %c, %k, %v, %m
	fields := strings.Fields(exec)
	var cleaned []string
	for _, f := range fields {
		if len(f) == 2 && f[0] == '%' {
			continue
		}
		cleaned = append(cleaned, f)
	}
	return strings.Join(cleaned, " ")
}
