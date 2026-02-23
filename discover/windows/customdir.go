//go:build windows

package windows

import (
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// archDirNames is the set of directory names considered architecture-specific.
var archDirNames = map[string]bool{
	"x64": true, "x86": true, "arm": true, "arm64": true,
	"amd64": true, "win32": true, "win64": true, "i386": true, "i686": true,
}

// isArchDirName returns true when a directory name (lowercased) is an arch specifier.
func isArchDirName(name string) bool { return archDirNames[name] }

// archPriority assigns a preference rank to arch directory names; lower = preferred.
var archPriority = map[string]int{
	"x64": 0, "amd64": 0,
	"win64": 1,
	"x86": 2, "win32": 2, "i386": 2, "i686": 2,
	"arm64": 3,
	"arm":   4,
}

// collapseArchDirs merges groups[dir] entries where all sibling directories
// sharing the same grandparent are arch-named. Only the best-arch representative
// exe is kept; the others are discarded. The surviving exe is stored under the
// grandparent key so the display name can omit the arch path component.
func collapseArchDirs(groups map[string][]string, rootAbs string) map[string][]string {
	// Build a map of grandparent → child dirs (that have exes).
	byGrandparent := make(map[string][]string)
	for dir := range groups {
		if dir == rootAbs {
			continue
		}
		parent := filepath.Dir(dir)
		byGrandparent[parent] = append(byGrandparent[parent], dir)
	}

	result := make(map[string][]string, len(groups))
	for k, v := range groups {
		result[k] = v
	}

	for grandparent, childDirs := range byGrandparent {
		if len(childDirs) < 2 {
			continue
		}
		// Only merge when ALL child dirs with exes are arch-named.
		allArch := true
		for _, d := range childDirs {
			if !isArchDirName(strings.ToLower(filepath.Base(d))) {
				allArch = false
				break
			}
		}
		if !allArch {
			continue
		}

		// Pick the exe from the most-preferred architecture.
		best := ""
		bestPriority := math.MaxInt32
		for _, d := range childDirs {
			exe := pickMainExe(groups[d])
			dirName := strings.ToLower(filepath.Base(d))
			priority, ok := archPriority[dirName]
			if !ok {
				priority = 5
			}
			if priority < bestPriority {
				bestPriority = priority
				best = exe
			}
		}

		// Replace the individual arch groups with a single entry under grandparent.
		for _, d := range childDirs {
			delete(result, d)
		}
		if best != "" {
			result[grandparent] = []string{best}
		}
	}

	return result
}

// cleanRelPath removes any arch-named directory components from a relative path,
// keeping the filename intact. This produces clean display names when an arch
// directory was collapsed, e.g. "WinDirStat\x64\windirstat" → "WinDirStat\windirstat".
func cleanRelPath(rel string) string {
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) <= 1 {
		return rel
	}
	cleaned := make([]string, 0, len(parts))
	for i, p := range parts {
		if i == len(parts)-1 || !isArchDirName(strings.ToLower(p)) {
			cleaned = append(cleaned, p)
		}
	}
	return filepath.Join(cleaned...)
}

// CustomDirSource discovers .exe files in a user-specified directory (recursively).
// Each CustomDirSource has a MenuName that becomes both the Category and the
// submenu display label in the generated config.
//
// Discovery strategy:
//   - .exe files found directly in Dir (root level) are all kept — each is
//     assumed to be a distinct standalone tool.
//   - .exe files found inside a subdirectory are grouped by that subdirectory.
//     When a subdirectory contains more than one candidate, pickMainExe selects
//     a single representative using an arch-suffix heuristic, so that variant
//     binaries (e.g. tcpview64.exe, tcpvcon.exe) do not create duplicate entries.
//
// The optional Exclude list contains glob patterns matched against the exe
// filename (case-insensitive). Any matching exe is skipped before grouping.
type CustomDirSource struct {
	Dir      string
	MenuName string
	Exclude  []string // glob patterns matched against filename, e.g. "*64*"
}

// Name returns a unique identifier for this source derived from the menu name.
func (s *CustomDirSource) Name() string {
	lower := strings.ToLower(s.MenuName)
	return "customdir:" + strings.ReplaceAll(lower, " ", "-")
}

// Category returns the display name for the menu section.
func (s *CustomDirSource) Category() string { return s.MenuName }

// Available reports whether the configured directory exists.
func (s *CustomDirSource) Available() bool {
	_, err := os.Stat(s.Dir)
	return err == nil
}

// Discover walks Dir recursively, applies filtering, and returns discovered apps.
func (s *CustomDirSource) Discover() ([]discover.DiscoveredApp, error) {
	rootAbs, err := filepath.Abs(s.Dir)
	if err != nil {
		rootAbs = s.Dir
	}

	// Collect all candidate exe paths, grouped by their immediate parent directory.
	// key = absolute path of immediate parent dir.
	groups := make(map[string][]string)

	walkerr := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			return nil
		}
		if isFilteredExecutable(name) {
			return nil
		}
		if s.matchesExclude(name) {
			return nil
		}

		absPath, aerr := filepath.Abs(path)
		if aerr != nil {
			absPath = path
		}

		parent := filepath.Dir(absPath)
		groups[parent] = append(groups[parent], absPath)
		return nil
	})
	if walkerr != nil {
		return nil, walkerr
	}

	// Collapse groups where all sibling dirs under a common parent are arch-named
	// (e.g. WinDirStat/x64, WinDirStat/x86, WinDirStat/arm64, WinDirStat/arm).
	groups = collapseArchDirs(groups, rootAbs)

	// For each group decide which exe(s) to keep.
	seen := make(map[string]bool)
	var selected []string

	for parent, exes := range groups {
		if parent == rootAbs {
			// Root-level files: keep all — each is a distinct standalone tool.
			selected = append(selected, exes...)
		} else {
			// Subdirectory: pick one representative exe.
			selected = append(selected, pickMainExe(exes))
		}
	}

	// Sort for deterministic output order.
	sort.Strings(selected)

	var apps []discover.DiscoveredApp
	for _, absPath := range selected {
		key := strings.ToLower(absPath)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Build display name as relative path from scan root, stripping arch dir
		// components and .exe — e.g.:
		//   F:\Utils\WinDirStat\x64\windirstat.exe → "WinDirStat\windirstat"
		//   F:\Utils\putty.exe                    → "putty"
		rel, err := filepath.Rel(rootAbs, absPath)
		if err != nil {
			rel = filepath.Base(absPath)
		}
		displayName := strings.TrimSuffix(rel, ".exe")
		displayName = strings.TrimSuffix(displayName, ".EXE")
		displayName = cleanRelPath(displayName)

		execPath := absPath
		if strings.Contains(execPath, " ") {
			execPath = `"` + execPath + `"`
		}

		apps = append(apps, discover.DiscoveredApp{
			Name:     displayName,
			Exec:     execPath,
			Source:   s.MenuName,
			Category: s.MenuName,
		})
	}

	return apps, nil
}

// matchesExclude returns true if the filename matches any of the Exclude glob patterns.
func (s *CustomDirSource) matchesExclude(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range s.Exclude {
		matched, err := filepath.Match(strings.ToLower(pattern), lower)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// pickMainExe selects the single best representative exe from a group of
// candidates found in the same subdirectory.
//
// Strategy:
//  1. Filter out arch/console variants (names ending in 64, 64a, 32, x64, x86,
//     _x64, _x86, con, cmd, cli — case-insensitive, after stripping .exe).
//  2. From the survivors, return the one with the shortest filename.
//  3. If all candidates were filtered out (no non-variant remains), fall back
//     to the shortest name from the full original set.
func pickMainExe(exes []string) string {
	var candidates []string
	for _, exe := range exes {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(exe), ".exe"))
		if !isArchVariant(base) {
			candidates = append(candidates, exe)
		}
	}
	if len(candidates) == 0 {
		candidates = exes // all were variants; fall back to full set
	}
	// Return shortest filename (most likely the base/main exe).
	sort.Slice(candidates, func(i, j int) bool {
		ni := filepath.Base(candidates[i])
		nj := filepath.Base(candidates[j])
		if len(ni) != len(nj) {
			return len(ni) < len(nj)
		}
		return ni < nj // stable alphabetical tiebreak
	})
	return candidates[0]
}

// isArchVariant returns true when a base exe name (without .exe) looks like an
// architecture-specific variant or a console companion rather than a main binary.
func isArchVariant(base string) bool {
	// Numeric architecture suffixes and console companion suffixes to skip.
	variantSuffixes := []string{
		"64", "32", "64a", "86",
		"x64", "x86",
		"_x64", "_x86", "_64", "_32",
		"con", "cmd", "cli",
	}
	for _, s := range variantSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	return false
}
