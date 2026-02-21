//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Start Menu Filter Tests ---

func TestIsFilteredShortcut(t *testing.T) {
	tests := []struct {
		name     string
		filtered bool
	}{
		{"Notepad++", false},
		{"Firefox", false},
		{"Uninstall MyApp", true},
		{"Update Manager", true},
		{"MyApp Updater", true},
		{"README", true},
		{"Help", true},
		{"Documentation", true},
		{"License Agreement", true},
		{"Release Notes", true},
		{"Repair Tool", true},
		{"Website", true},
		{"My Game", false},
		{"Remove MyApp", true},
		{"Troubleshoot", true},
	}

	for _, tc := range tests {
		got := isFilteredShortcut(tc.name)
		if got != tc.filtered {
			t.Errorf("isFilteredShortcut(%q) = %v, expected %v", tc.name, got, tc.filtered)
		}
	}
}

// --- Steam VDF Parsing Tests ---

func TestParseVDFLine(t *testing.T) {
	tests := []struct {
		line     string
		key      string
		value    string
	}{
		{`		"appid"		"220"`, "appid", "220"},
		{`		"name"		"Half-Life 2"`, "name", "Half-Life 2"},
		{`		"Universe"		"1"`, "universe", "1"},
		{`	{`, "", ""},
		{`	}`, "", ""},
		{`"path"		"C:\\Steam"`, "path", `C:\\Steam`},
		{``, "", ""},
		{``, "", ""},
	}

	for _, tc := range tests {
		k, v := parseVDFLine(tc.line)
		if k != tc.key || v != tc.value {
			t.Errorf("parseVDFLine(%q) = (%q, %q), expected (%q, %q)", tc.line, k, v, tc.key, tc.value)
		}
	}
}

func TestIsSteamTool(t *testing.T) {
	tests := []struct {
		name     string
		isTool   bool
	}{
		{"Half-Life 2", false},
		{"Portal", false},
		{"Microsoft Visual C++ 2015-2019 Redistributable", true},
		{"Proton Experimental", true},
		{"Steamworks Common Redistributables", true},
		{"DirectX Runtime", true},
		{"Steam Linux Runtime - Sniper", true},
		{"Counter-Strike 2", false},
	}

	for _, tc := range tests {
		got := isSteamTool(tc.name)
		if got != tc.isTool {
			t.Errorf("isSteamTool(%q) = %v, expected %v", tc.name, got, tc.isTool)
		}
	}
}

func TestExtractLibraryPaths(t *testing.T) {
	// Create a temp directory structure to simulate Steam libraries
	tmpDir := t.TempDir()
	lib1 := filepath.Join(tmpDir, "lib1", "steamapps")
	lib2 := filepath.Join(tmpDir, "lib2", "steamapps")
	os.MkdirAll(lib1, 0755)
	os.MkdirAll(lib2, 0755)

	// Note: extracts paths that match "path" key and have existing steamapps dirs
	vdf := `"libraryfolders"
{
	"0"
	{
		"path"		"` + strings.ReplaceAll(filepath.Join(tmpDir, "lib1"), `\`, `\\`) + `"
		"label"		""
	}
	"1"
	{
		"path"		"` + strings.ReplaceAll(filepath.Join(tmpDir, "lib2"), `\`, `\\`) + `"
		"label"		""
	}
	"2"
	{
		"path"		"C:\\nonexistent\\path"
		"label"		""
	}
}`

	paths := extractLibraryPaths(vdf)

	// Should find 2 paths (lib1 and lib2 exist, nonexistent doesn't)
	if len(paths) != 2 {
		t.Fatalf("expected 2 library paths, got %d: %v", len(paths), paths)
	}

	// Verify both paths end with steamapps
	for _, p := range paths {
		if !strings.HasSuffix(p, "steamapps") {
			t.Errorf("expected path ending in 'steamapps', got: %s", p)
		}
	}
}

func TestParseAppManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a valid app manifest
	manifest := `"AppState"
{
	"appid"		"220"
	"Universe"		"1"
	"name"		"Half-Life 2"
	"StateFlags"		"4"
	"installdir"		"Half-Life 2"
}`
	manifestPath := filepath.Join(tmpDir, "appmanifest_220.acf")
	os.WriteFile(manifestPath, []byte(manifest), 0644)

	app, err := parseAppManifest(manifestPath)
	if err != nil {
		t.Fatalf("parseAppManifest failed: %v", err)
	}

	if app.Name != "Half-Life 2" {
		t.Errorf("expected name 'Half-Life 2', got '%s'", app.Name)
	}
	if app.Exec != "start steam://rungameid/220" {
		t.Errorf("expected exec 'start steam://rungameid/220', got '%s'", app.Exec)
	}
	if app.Source != "steam" {
		t.Errorf("expected source 'steam', got '%s'", app.Source)
	}
	if app.Category != "Games" {
		t.Errorf("expected category 'Games', got '%s'", app.Category)
	}
}

func TestParseAppManifestIncomplete(t *testing.T) {
	tmpDir := t.TempDir()

	// Manifest missing name
	manifest := `"AppState"
{
	"appid"		"999"
}`
	manifestPath := filepath.Join(tmpDir, "appmanifest_999.acf")
	os.WriteFile(manifestPath, []byte(manifest), 0644)

	_, err := parseAppManifest(manifestPath)
	if err == nil {
		t.Fatal("expected error for incomplete manifest")
	}
}

func TestParseAppManifestFiltersTool(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `"AppState"
{
	"appid"		"228980"
	"name"		"Steamworks Common Redistributables"
}`
	manifestPath := filepath.Join(tmpDir, "appmanifest_228980.acf")
	os.WriteFile(manifestPath, []byte(manifest), 0644)

	_, err := parseAppManifest(manifestPath)
	if err == nil {
		t.Fatal("expected tool to be filtered out")
	}
}

// --- Program Files Filter Tests ---

func TestIsFilteredExecutable(t *testing.T) {
	tests := []struct {
		name     string
		filtered bool
	}{
		{"notepad++.exe", false},
		{"firefox.exe", false},
		{"unins000.exe", true},
		{"uninstall.exe", true},
		{"updater.exe", true},
		{"setup.exe", true},
		{"installer.exe", true},
		{"CrashReporter.exe", true},
		{"helper.exe", true},
		{"myapp.exe", false},
		{"update.exe", true},
		{"diagnostic.exe", true},
	}

	for _, tc := range tests {
		got := isFilteredExecutable(tc.name)
		if got != tc.filtered {
			t.Errorf("isFilteredExecutable(%q) = %v, expected %v", tc.name, got, tc.filtered)
		}
	}
}

func TestFindExecutables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{
		"myapp.exe",
		"uninstall.exe",    // should be filtered
		"helper.exe",       // should be filtered
		"readme.txt",       // not an exe
		"mainprogram.exe",  // should be included
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(tmpDir, f), []byte{}, 0644)
	}

	// Create a subdirectory (should not be scanned - one level only)
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "nested.exe"), []byte{}, 0644)

	exes := findExecutables(tmpDir)

	// Should find myapp.exe and mainprogram.exe only
	if len(exes) != 2 {
		t.Fatalf("expected 2 executables, got %d: %v", len(exes), exes)
	}

	names := make(map[string]bool)
	for _, exe := range exes {
		names[filepath.Base(exe)] = true
	}
	if !names["myapp.exe"] {
		t.Error("expected myapp.exe to be found")
	}
	if !names["mainprogram.exe"] {
		t.Error("expected mainprogram.exe to be found")
	}
}

func TestCleanAppName(t *testing.T) {
	got := cleanAppName("Notepad++", "notepad++.exe")
	if got != "Notepad++" {
		t.Errorf("expected 'Notepad++', got '%s'", got)
	}
}

func TestStartMenuDirs(t *testing.T) {
	dirs := startMenuDirs()
	// Should have at least one directory on a Windows system
	if len(dirs) == 0 {
		t.Fatal("startMenuDirs returned no directories")
	}
	for _, d := range dirs {
		if !strings.Contains(d, "Start Menu") {
			t.Errorf("expected Start Menu path, got: %s", d)
		}
	}
}

func TestProgramFilesDirs(t *testing.T) {
	dirs := programFilesDirs()
	if len(dirs) == 0 {
		t.Fatal("programFilesDirs returned no directories")
	}
	for _, d := range dirs {
		if !strings.Contains(strings.ToLower(d), "program files") {
			t.Errorf("expected Program Files path, got: %s", d)
		}
	}
}
