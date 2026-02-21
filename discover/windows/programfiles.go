//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// ProgramFilesSource discovers applications from Program Files directories.
type ProgramFilesSource struct{}

func (s *ProgramFilesSource) Name() string     { return "programfiles" }
func (s *ProgramFilesSource) Category() string { return "Applications" }

func (s *ProgramFilesSource) Available() bool {
	for _, dir := range programFilesDirs() {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

func (s *ProgramFilesSource) Discover() ([]discover.DiscoveredApp, error) {
	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, baseDir := range programFilesDirs() {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			subDir := filepath.Join(baseDir, entry.Name())
			exe := findMainExecutable(subDir, entry.Name())
			if exe == "" {
				continue
			}

			key := strings.ToLower(exe)
			if seen[key] {
				continue
			}
			seen[key] = true

			name := cleanAppName(entry.Name(), filepath.Base(exe))

			apps = append(apps, discover.DiscoveredApp{
				Name:     name,
				Exec:     exe,
				Source:   "programfiles",
				Category: "Applications",
			})
		}
	}

	return apps, nil
}

// programFilesDirs returns Program Files directories to scan.
func programFilesDirs() []string {
	var dirs []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		dirs = append(dirs, pf)
	}
	if pfx86 := os.Getenv("ProgramFiles(x86)"); pfx86 != "" {
		dirs = append(dirs, pfx86)
	}
	return dirs
}

// findMainExecutable finds the single best .exe in a directory.
// Prefers an exe whose name matches the directory name; otherwise picks the first non-filtered one.
func findMainExecutable(dir string, dirName string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	normDir := strings.ToLower(strings.ReplaceAll(dirName, " ", ""))
	var fallback string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			continue
		}
		if isFilteredExecutable(name) {
			continue
		}

		// Prefer exe whose base name (without .exe) matches the directory name
		normExe := strings.ToLower(strings.ReplaceAll(strings.TrimSuffix(name, filepath.Ext(name)), " ", ""))
		if normExe == normDir {
			return filepath.Join(dir, name)
		}

		// Keep first valid exe as fallback
		if fallback == "" {
			fallback = filepath.Join(dir, name)
		}
	}

	return fallback
}

// isFilteredExecutable returns true if the exe name suggests it should be excluded.
func isFilteredExecutable(name string) bool {
	lower := strings.ToLower(name)
	filterWords := []string{
		"unins", "uninst", "uninstall", "remove",
		"update", "updater",
		"setup", "install", "installer",
		"helper", "crash", "reporter", "diagnostic",
		"daemon", "service", "svc",
		"cli", "cmd",
	}
	for _, w := range filterWords {
		if strings.Contains(lower, w) {
			return true
		}
	}

	// Filter common helper executables by exact name
	exactFilter := map[string]bool{
		"wow_helper.exe": true,
		"dxsetup.exe":    true,
		"vcredist.exe":   true,
	}
	return exactFilter[lower]
}

// cleanAppName produces a display name from the directory and exe name.
// Uses the directory name since it's usually the proper application name.
func cleanAppName(dirName, exeName string) string {
	// If the exe name (without .exe) is a simplified version of the dir name, use dir name
	return dirName
}
