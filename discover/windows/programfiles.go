//go:build windows

package windows

import (
	"fmt"
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
			exes := findExecutables(subDir)

			for _, exe := range exes {
				key := strings.ToLower(exe)
				if seen[key] {
					continue
				}
				seen[key] = true

				name := cleanAppName(entry.Name(), filepath.Base(exe))
				exec := fmt.Sprintf("start \"\" \"%s\"", exe)

				apps = append(apps, discover.DiscoveredApp{
					Name:     name,
					Exec:     exec,
					Source:   "programfiles",
					Category: "Applications",
				})
			}
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

// findExecutables finds .exe files in a directory (one level deep only).
func findExecutables(dir string) []string {
	var exes []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return exes
	}

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
		exes = append(exes, filepath.Join(dir, name))
	}

	return exes
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
