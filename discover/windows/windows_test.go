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

func TestFindMainExecutable(t *testing.T) {
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

	// Returns single best exe (first non-filtered alphabetically as fallback)
	exe := findMainExecutable(tmpDir, "testdir")
	if exe == "" {
		t.Fatal("expected an executable to be found")
	}
	base := filepath.Base(exe)
	// mainprogram.exe comes before myapp.exe alphabetically
	if base != "mainprogram.exe" {
		t.Errorf("expected mainprogram.exe as fallback, got %s", base)
	}
}

func TestFindMainExecutablePrefersDirNameMatch(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "other.exe"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "myapp.exe"), []byte{}, 0644)

	exe := findMainExecutable(tmpDir, "MyApp")
	if exe == "" {
		t.Fatal("expected an executable to be found")
	}
	if filepath.Base(exe) != "myapp.exe" {
		t.Errorf("expected myapp.exe (matches dir name), got %s", filepath.Base(exe))
	}
}

func TestFindMainExecutableEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	exe := findMainExecutable(tmpDir, "empty")
	if exe != "" {
		t.Errorf("expected empty result for empty dir, got %s", exe)
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

// --- Xbox Source Tests ---

func TestCleanPackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Microsoft.MinecraftUWP", "Minecraft"},
		{"BethesdaSoftworks.Starfield", "Starfield"},
		{"343Industries.HaloInfinite", "Halo Infinite"},
		{"EA.DeadSpaceRemake", "Dead Space Remake"},
		{"Microsoft.SeaOfThievesW10", "Sea Of Thieves"},
		{"Ubisoft.FarCry6Beta", "Far Cry6"},
		{"Simple", "Simple"},
		{"Publisher.GameWindows", "Game"},
		{"Publisher.GamePC", "Game"},
		{"Publisher.GamePreview", "Game"},
	}

	for _, tc := range tests {
		got := cleanPackageName(tc.input)
		if got != tc.expected {
			t.Errorf("cleanPackageName(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HaloInfinite", "Halo Infinite"},
		{"DeadSpaceRemake", "Dead Space Remake"},
		{"Minecraft", "Minecraft"},
		{"XMLParser", "XML Parser"},
		{"", ""},
		{"lowercase", "lowercase"},
		{"A", "A"},
		{"AB", "AB"},
		{"ABc", "A Bc"},
	}

	for _, tc := range tests {
		got := splitCamelCase(tc.input)
		if got != tc.expected {
			t.Errorf("splitCamelCase(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestBuildAUMID(t *testing.T) {
	tests := []struct {
		pfn      string
		appID    string
		expected string
	}{
		{"Microsoft.MinecraftUWP_8wekyb3d8bbwe", "App", "Microsoft.MinecraftUWP_8wekyb3d8bbwe!App"},
		{"Bethesda.Starfield_3275kfvn8vcwc", "Game", "Bethesda.Starfield_3275kfvn8vcwc!Game"},
		{"Publisher.Game_abc123", "App", "Publisher.Game_abc123!App"},
	}

	for _, tc := range tests {
		got := buildAUMID(tc.pfn, tc.appID)
		if got != tc.expected {
			t.Errorf("buildAUMID(%q, %q) = %q, expected %q", tc.pfn, tc.appID, got, tc.expected)
		}
	}
}

func TestParseAppxJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		count    int
		wantErr  bool
	}{
		{
			name:  "array of packages",
			input: `[{"Name":"Microsoft.MinecraftUWP","PackageFamilyName":"Microsoft.MinecraftUWP_8wekyb3d8bbwe"},{"Name":"Bethesda.Starfield","PackageFamilyName":"Bethesda.Starfield_3275kfvn8vcwc"}]`,
			count: 2,
		},
		{
			name:  "single object (PowerShell quirk)",
			input: `{"Name":"Microsoft.MinecraftUWP","PackageFamilyName":"Microsoft.MinecraftUWP_8wekyb3d8bbwe"}`,
			count: 1,
		},
		{
			name:  "empty array",
			input: `[]`,
			count: 0,
		},
		{
			name:  "empty string",
			input: ``,
			count: 0,
		},
		{
			name:  "null",
			input: `null`,
			count: 0,
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json`,
			count:   0,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkgs, err := parseAppxJSON([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(pkgs) != tc.count {
				t.Fatalf("expected %d packages, got %d", tc.count, len(pkgs))
			}
		})
	}
}

func TestParseAppxJSONFields(t *testing.T) {
	input := `{"Name":"Microsoft.MinecraftUWP","PackageFamilyName":"Microsoft.MinecraftUWP_8wekyb3d8bbwe","DisplayName":"Minecraft"}`
	pkgs, err := parseAppxJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "Microsoft.MinecraftUWP" {
		t.Errorf("expected Name 'Microsoft.MinecraftUWP', got %q", pkgs[0].Name)
	}
	if pkgs[0].PackageFamilyName != "Microsoft.MinecraftUWP_8wekyb3d8bbwe" {
		t.Errorf("expected PackageFamilyName, got %q", pkgs[0].PackageFamilyName)
	}
	if pkgs[0].DisplayName != "Minecraft" {
		t.Errorf("expected DisplayName 'Minecraft', got %q", pkgs[0].DisplayName)
	}
}

func TestIsGamePackage(t *testing.T) {
	tests := []struct {
		name    string
		pkg     appxPackage
		isGame  bool
	}{
		{"regular game", appxPackage{Name: "Bethesda.Starfield", PackageFamilyName: "Bethesda.Starfield_abc"}, true},
		{"minecraft", appxPackage{Name: "Microsoft.MinecraftUWP", PackageFamilyName: "Microsoft.MinecraftUWP_abc"}, true},
		{"gaming services", appxPackage{Name: "Microsoft.GamingServices", PackageFamilyName: "Microsoft.GamingServices_abc"}, false},
		{"xbox app", appxPackage{Name: "Microsoft.XboxApp", PackageFamilyName: "Microsoft.XboxApp_abc"}, false},
		{"game bar", appxPackage{Name: "Microsoft.XboxGameBar", PackageFamilyName: "Microsoft.XboxGameBar_abc"}, false},
		{"gaming overlay", appxPackage{Name: "Microsoft.XboxGamingOverlay", PackageFamilyName: "Microsoft.XboxGamingOverlay_abc"}, false},
		{"identity provider", appxPackage{Name: "Microsoft.XboxIdentityProvider", PackageFamilyName: "Microsoft.XboxIdentityProvider_abc"}, false},
		{"speech to text", appxPackage{Name: "Microsoft.XboxSpeechToText", PackageFamilyName: "Microsoft.XboxSpeechToText_abc"}, false},
		{"empty name", appxPackage{Name: "", PackageFamilyName: "abc"}, false},
		{"empty pfn", appxPackage{Name: "Game", PackageFamilyName: ""}, false},
		{"both empty", appxPackage{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isGamePackage(tc.pkg)
			if got != tc.isGame {
				t.Errorf("isGamePackage(%+v) = %v, expected %v", tc.pkg, got, tc.isGame)
			}
		})
	}
}

func TestXboxSourceMetadata(t *testing.T) {
	s := &XboxSource{}
	if s.Name() != "xbox" {
		t.Errorf("expected Name 'xbox', got %q", s.Name())
	}
	if s.Category() != "Games" {
		t.Errorf("expected Category 'Games', got %q", s.Category())
	}
}

func TestXboxDiscoverWithMockPowerShell(t *testing.T) {
	// Override runPowerShellCommand to simulate PowerShell output
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	// DisplayName provided for Starfield; empty for Minecraft (falls back to cleanPackageName)
	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[{"Name":"Bethesda.Starfield","PackageFamilyName":"Bethesda.Starfield_3275kfvn8vcwc","DisplayName":"Starfield"},{"Name":"Microsoft.MinecraftUWP","PackageFamilyName":"Microsoft.MinecraftUWP_8wekyb3d8bbwe","DisplayName":""},{"Name":"Microsoft.GamingServices","PackageFamilyName":"Microsoft.GamingServices_abc","DisplayName":""}]`), nil
	}

	s := &XboxSource{}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// GamingServices should be filtered out
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps (GamingServices filtered), got %d", len(apps))
	}

	// Check names — Starfield uses DisplayName, Minecraft falls back to cleanPackageName
	names := map[string]bool{}
	for _, a := range apps {
		names[a.Name] = true
	}
	if !names["Starfield"] {
		t.Error("expected 'Starfield' in results")
	}
	if !names["Minecraft"] {
		t.Error("expected 'Minecraft' in results (cleaned from MinecraftUWP)")
	}

	// Check exec format
	for _, a := range apps {
		if !strings.HasPrefix(a.Exec, `start "" "shell:AppsFolder\`) {
			t.Errorf(`expected exec to start with 'start "" "shell:AppsFolder\', got %q`, a.Exec)
		}
		if !strings.Contains(a.Exec, "!App") {
			t.Errorf("expected exec to contain '!App', got %q", a.Exec)
		}
		if a.Source != "xbox" {
			t.Errorf("expected source 'xbox', got %q", a.Source)
		}
		if a.Category != "Games" {
			t.Errorf("expected category 'Games', got %q", a.Category)
		}
	}
}

func TestXboxDiscoverPrefersDisplayName(t *testing.T) {
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	// Codenames that would produce bad cleaned names — DisplayName saves us
	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[{"Name":"Microsoft.Limitless","PackageFamilyName":"Microsoft.Limitless_8wekyb3d8bbwe","DisplayName":"Microsoft Flight Simulator 2024"},{"Name":"SEGAofAmericaInc.D0cb6b3aet","PackageFamilyName":"SEGAofAmericaInc.D0cb6b3aet_s751p9cej88mt","DisplayName":"Persona 4 Golden"}]`), nil
	}

	s := &XboxSource{}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}

	names := map[string]bool{}
	for _, a := range apps {
		names[a.Name] = true
	}
	if !names["Microsoft Flight Simulator 2024"] {
		t.Error("expected 'Microsoft Flight Simulator 2024', not the codename 'Limitless'")
	}
	if !names["Persona 4 Golden"] {
		t.Error("expected 'Persona 4 Golden', not the hash 'D0cb6b3aet'")
	}
}

func TestXboxDiscoverEmptyResult(t *testing.T) {
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[]`), nil
	}

	s := &XboxSource{}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(apps))
	}
}

func TestXboxDiscoverPowerShellError(t *testing.T) {
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	runPowerShellCommand = func(script string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	s := &XboxSource{}
	apps, err := s.Discover()
	if err == nil {
		t.Fatal("expected error when PowerShell fails")
	}
	if apps != nil {
		t.Fatalf("expected nil apps on error, got %d", len(apps))
	}
}

func TestXboxDiscoverDeduplicates(t *testing.T) {
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	// Same game name appearing twice (different package versions)
	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[{"Name":"Publisher.MyGame","PackageFamilyName":"Publisher.MyGame_abc","DisplayName":"My Game"},{"Name":"Publisher.MyGame","PackageFamilyName":"Publisher.MyGame_def","DisplayName":"My Game"}]`), nil
	}

	s := &XboxSource{}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app after internal dedup, got %d", len(apps))
	}
}
