//go:build windows

package windows

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"unicode"

	"github.com/benworks/menuworks/discover"
)

// XboxSource discovers games installed via the Xbox app / Microsoft Store.
// It uses PowerShell's Get-AppxPackage to enumerate packages and filters
// to games using the GamingServices package repository.
type XboxSource struct{}

func (s *XboxSource) Name() string     { return "xbox" }
func (s *XboxSource) Category() string { return "Games" }

// Available checks whether PowerShell and Get-AppxPackage are present.
// Returns false gracefully if PowerShell is not installed or the cmdlet
// is unavailable, allowing discovery to continue with other sources.
func (s *XboxSource) Available() bool {
	return isPowerShellAvailable()
}

// Discover enumerates Xbox/Microsoft Store games via PowerShell.
// Returns nil, error if PowerShell invocation fails (non-fatal in the pipeline).
func (s *XboxSource) Discover() ([]discover.DiscoveredApp, error) {
	data, err := runPowerShellCommand(xboxDiscoveryScript)
	if err != nil {
		return nil, fmt.Errorf("xbox: powershell command failed: %w", err)
	}

	pkgs, err := parseAppxJSON(data)
	if err != nil {
		return nil, fmt.Errorf("xbox: failed to parse package data: %w", err)
	}

	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, pkg := range pkgs {
		if !isGamePackage(pkg) {
			continue
		}

		// Prefer the display name from the AppxManifest over the coded package name
		name := pkg.DisplayName
		if name == "" {
			name = cleanPackageName(pkg.Name)
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		appID := pkg.AppID
		if appID == "" {
			appID = "App"
		}
		aumid := buildAUMID(pkg.PackageFamilyName, appID)
		apps = append(apps, discover.DiscoveredApp{
			Name:     name,
			Exec:     fmt.Sprintf("explorer.exe shell:AppsFolder\\%s", aumid),
			Source:   "xbox",
			Category: "Games",
		})
	}

	return apps, nil
}

// xboxDiscoveryScript is the PowerShell command that enumerates Store gaming packages.
// It reads game package names from GamingServices\GameConfig (which lists all games
// registered with Xbox/Gaming Services), cross-references them with AppxPackage,
// and reads the display name from each package's AppxManifest.xml.
const xboxDiscoveryScript = `$ErrorActionPreference = 'SilentlyContinue'
$gc = Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\GamingServices\GameConfig' 2>$null | Select-Object -ExpandProperty PSChildName 2>$null
if (-not $gc) { '[]'; exit 0 }
$gameNames = @{}
foreach ($entry in $gc) {
    $base = ($entry -split '_')[0]
    if ($base) { $gameNames[$base] = $true }
}
if ($gameNames.Count -eq 0) { '[]'; exit 0 }
$results = @()
Get-AppxPackage | Where-Object { -not $_.IsFramework -and $gameNames.ContainsKey($_.Name) } | ForEach-Object {
    $dn = ''
    $aid = 'App'
    $mp = Join-Path $_.InstallLocation 'AppxManifest.xml'
    if (Test-Path $mp) {
        try {
            [xml]$m = Get-Content $mp
            $d = $m.Package.Properties.DisplayName
            if ($d -and $d -notmatch '^ms-resource:') { $dn = $d }
            $a = $m.Package.Applications.Application
            if ($a -is [array]) { $aid = $a[0].Id } elseif ($a) { $aid = $a.Id }
        } catch {}
    }
    $results += [PSCustomObject]@{ Name = $_.Name; PackageFamilyName = $_.PackageFamilyName; DisplayName = $dn; AppId = $aid }
}
if ($results.Count -eq 0) { '[]' } else { $results | ConvertTo-Json -Compress }`

// appxPackage represents the relevant fields from Get-AppxPackage JSON output.
type appxPackage struct {
	Name              string `json:"Name"`
	PackageFamilyName string `json:"PackageFamilyName"`
	DisplayName       string `json:"DisplayName"`
	AppID             string `json:"AppId"`
}

// isPowerShellAvailable checks if powershell.exe can be found and the
// Get-AppxPackage cmdlet exists.
func isPowerShellAvailable() bool {
	path, err := exec.LookPath("powershell.exe")
	if err != nil || path == "" {
		return false
	}
	// Verify Get-AppxPackage cmdlet is available
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		"if (Get-Command Get-AppxPackage -ErrorAction SilentlyContinue) { 'ok' } else { exit 1 }")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "ok"
}

// runPowerShellCommand executes a PowerShell script and returns stdout bytes.
// This is a package-level var so tests can override it.
var runPowerShellCommand = runPowerShellCommandImpl

func runPowerShellCommandImpl(script string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Output()
}

// parseAppxJSON parses the JSON output from Get-AppxPackage.
// Handles both array and single-object responses (PowerShell returns a bare
// object instead of an array when there is exactly one result).
func parseAppxJSON(data []byte) ([]appxPackage, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil, nil
	}

	// Try array first
	var pkgs []appxPackage
	if err := json.Unmarshal([]byte(trimmed), &pkgs); err == nil {
		return pkgs, nil
	}

	// PowerShell emits a bare object when there is exactly one result
	var single appxPackage
	if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if single.Name == "" {
		return nil, nil
	}
	return []appxPackage{single}, nil
}

// buildAUMID constructs an Application User Model ID for launching a Store app.
// Format: "{PackageFamilyName}!{AppID}"
func buildAUMID(packageFamilyName, appID string) string {
	return packageFamilyName + "!" + appID
}

// cleanPackageName converts a raw AppX package name to a human-readable display name.
// Examples:
//
//	"Microsoft.MinecraftUWP"      -> "Minecraft"
//	"BethesdaSoftworks.Starfield" -> "Starfield"
//	"343Industries.HaloInfinite"  -> "Halo Infinite"
func cleanPackageName(name string) string {
	// Strip publisher prefix (everything before and including the last dot)
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}

	// Remove common suffixes
	for _, suffix := range []string{"UWP", "W10", "Win10", "PC", "Windows", "Beta", "Preview"} {
		name = strings.TrimSuffix(name, suffix)
	}
	name = strings.TrimRight(name, "_- ")

	// Insert spaces before uppercase letters in camelCase/PascalCase
	name = splitCamelCase(name)

	return strings.TrimSpace(name)
}

// splitCamelCase inserts spaces before runs of uppercase letters forming new words.
// "HaloInfinite" -> "Halo Infinite", "MinecraftDungeons" -> "Minecraft Dungeons"
func splitCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			// Insert space if previous char is lowercase, or if this uppercase
			// is followed by a lowercase (handles "XMLParser" -> "XML Parser")
			if unicode.IsLower(prev) {
				b.WriteRune(' ')
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isGamePackage returns true if the package looks like a game rather than
// a system component or utility. This is a secondary filter after the
// GamingServices cross-reference in the PowerShell script.
func isGamePackage(pkg appxPackage) bool {
	if pkg.Name == "" || pkg.PackageFamilyName == "" {
		return false
	}

	lower := strings.ToLower(pkg.Name)

	// Filter known non-game packages that may appear in GamingServices
	nonGamePrefixes := []string{
		"microsoft.gamingservices",
		"microsoft.gamingapp",
		"microsoft.xboxapp",
		"microsoft.xboxgamebar",
		"microsoft.xboxidentityprovider",
		"microsoft.xboxspeechtotext",
		"microsoft.xboxgamecallableui",
		"microsoft.xbox.tcui",
		"microsoft.xboxgamingoverlay",
	}
	for _, prefix := range nonGamePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}

	return true
}
