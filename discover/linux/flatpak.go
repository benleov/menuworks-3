//go:build linux

package linux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/benworks/menuworks/discover"
)

// FlatpakSource discovers applications installed via Flatpak.
type FlatpakSource struct{}

func (s *FlatpakSource) Name() string     { return "flatpak" }
func (s *FlatpakSource) Category() string { return "Applications" }

func (s *FlatpakSource) Available() bool {
	_, err := exec.LookPath("flatpak")
	return err == nil
}

func (s *FlatpakSource) Discover() ([]discover.DiscoveredApp, error) {
	return discoverFlatpak()
}

func discoverFlatpak() ([]discover.DiscoveredApp, error) {
	// List installed Flatpak apps (not runtimes)
	out, err := exec.Command("flatpak", "list", "--app", "--columns=application,name").Output()
	if err != nil {
		return nil, err
	}

	return parseFlatpakOutput(string(out))
}

// parseFlatpakOutput parses the output of `flatpak list --app --columns=application,name`.
func parseFlatpakOutput(output string) ([]discover.DiscoveredApp, error) {
	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "application.id\tDisplay Name"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}

		appID := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])

		if appID == "" || name == "" {
			continue
		}

		key := strings.ToLower(appID)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Check for .desktop file to get the actual Exec command
		execCmd := flatpakExecCommand(appID)

		apps = append(apps, discover.DiscoveredApp{
			Name:     name,
			Exec:     execCmd,
			Source:   "Flatpak",
			Category: "Applications",
		})
	}

	return apps, nil
}

// flatpakExecCommand returns the command to launch a Flatpak app.
func flatpakExecCommand(appID string) string {
	// Check if app has a .desktop file with an Exec line
	desktopPaths := []string{
		filepath.Join("/var/lib/flatpak/exports/share/applications", appID+".desktop"),
	}

	home := os.Getenv("HOME")
	if home != "" {
		desktopPaths = append(desktopPaths,
			filepath.Join(home, ".local/share/flatpak/exports/share/applications", appID+".desktop"),
		)
	}

	for _, path := range desktopPaths {
		if _, err := os.Stat(path); err == nil {
			// Use flatpak run which handles sandboxing
			return "flatpak run " + appID
		}
	}

	return "flatpak run " + appID
}
