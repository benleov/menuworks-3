//go:build linux

package linux

import (
	"os/exec"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// SnapSource discovers applications installed via Snap.
type SnapSource struct{}

func (s *SnapSource) Name() string     { return "snap" }
func (s *SnapSource) Category() string { return "Applications" }

func (s *SnapSource) Available() bool {
	_, err := exec.LookPath("snap")
	return err == nil
}

func (s *SnapSource) Discover() ([]discover.DiscoveredApp, error) {
	return discoverSnap()
}

func discoverSnap() ([]discover.DiscoveredApp, error) {
	out, err := exec.Command("snap", "list").Output()
	if err != nil {
		return nil, err
	}

	return parseSnapOutput(string(out))
}

// parseSnapOutput parses the output of `snap list`.
func parseSnapOutput(output string) ([]discover.DiscoveredApp, error) {
	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip header line
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: Name  Version  Rev  Tracking  Publisher  Notes
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]

		// Filter out system snaps
		if isSystemSnap(name) {
			continue
		}

		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true

		apps = append(apps, discover.DiscoveredApp{
			Name:     name,
			Exec:     "snap run " + name,
			Source:   "Snap",
			Category: "Applications",
		})
	}

	return apps, nil
}

// isSystemSnap returns true if the snap is a core/system component.
func isSystemSnap(name string) bool {
	systemSnaps := map[string]bool{
		"bare":             true,
		"core":             true,
		"core18":           true,
		"core20":           true,
		"core22":           true,
		"core24":           true,
		"gnome-3-28-1804":  true,
		"gnome-3-34-1804":  true,
		"gnome-3-38-2004":  true,
		"gnome-42-2204":    true,
		"gnome-46-2404":    true,
		"gtk-common-themes": true,
		"snapd":            true,
		"snapd-desktop-integration": true,
	}
	return systemSnaps[name]
}
