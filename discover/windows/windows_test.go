//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benworks/menuworks/discover"
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
	// AppId varies per game (Game, Minecraft, etc.) — uses manifest values
	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[{"Name":"Bethesda.Starfield","PackageFamilyName":"Bethesda.Starfield_3275kfvn8vcwc","DisplayName":"Starfield","AppId":"Game"},{"Name":"Microsoft.MinecraftUWP","PackageFamilyName":"Microsoft.MinecraftUWP_8wekyb3d8bbwe","DisplayName":"","AppId":"Minecraft"},{"Name":"Microsoft.GamingServices","PackageFamilyName":"Microsoft.GamingServices_abc","DisplayName":"","AppId":"App"}]`), nil
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

	// Check exec format — uses explorer.exe with correct AppId from manifest
	for _, a := range apps {
		if !strings.HasPrefix(a.Exec, "explorer.exe shell:AppsFolder\\") {
			t.Errorf("expected exec to start with 'explorer.exe shell:AppsFolder\\', got %q", a.Exec)
		}
		if !strings.Contains(a.Exec, "!") {
			t.Errorf("expected exec to contain '!' (AUMID separator), got %q", a.Exec)
		}
		if a.Source != "xbox" {
			t.Errorf("expected source 'xbox', got %q", a.Source)
		}
		if a.Category != "Games" {
			t.Errorf("expected category 'Games', got %q", a.Category)
		}
	}

	// Verify specific AppIds are used
	execMap := map[string]string{}
	for _, a := range apps {
		execMap[a.Name] = a.Exec
	}
	if !strings.HasSuffix(execMap["Starfield"], "!Game") {
		t.Errorf("Starfield should use !Game AppId, got %q", execMap["Starfield"])
	}
	if !strings.HasSuffix(execMap["Minecraft"], "!Minecraft") {
		t.Errorf("Minecraft should use !Minecraft AppId, got %q", execMap["Minecraft"])
	}
}

func TestXboxDiscoverPrefersDisplayName(t *testing.T) {
	origRunner := runPowerShellCommand
	defer func() { runPowerShellCommand = origRunner }()

	// Codenames that would produce bad cleaned names — DisplayName saves us
	runPowerShellCommand = func(script string) ([]byte, error) {
		return []byte(`[{"Name":"Microsoft.Limitless","PackageFamilyName":"Microsoft.Limitless_8wekyb3d8bbwe","DisplayName":"Microsoft Flight Simulator 2024","AppId":"App"},{"Name":"SEGAofAmericaInc.D0cb6b3aet","PackageFamilyName":"SEGAofAmericaInc.D0cb6b3aet_s751p9cej88mt","DisplayName":"Persona 4 Golden","AppId":"Game"}]`), nil
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
		return []byte(`[{"Name":"Publisher.MyGame","PackageFamilyName":"Publisher.MyGame_abc","DisplayName":"My Game","AppId":"Game"},{"Name":"Publisher.MyGame","PackageFamilyName":"Publisher.MyGame_def","DisplayName":"My Game","AppId":"Game"}]`), nil
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

// --- CustomDirSource Tests ---

func TestCustomDirSource_Available(t *testing.T) {
	dir := t.TempDir()
	s := &CustomDirSource{Dir: dir, MenuName: "Test"}
	if !s.Available() {
		t.Error("expected Available() == true for existing directory")
	}

	s2 := &CustomDirSource{Dir: filepath.Join(dir, "nonexistent"), MenuName: "Test"}
	if s2.Available() {
		t.Error("expected Available() == false for missing directory")
	}
}

func TestCustomDirSource_Name(t *testing.T) {
	s := &CustomDirSource{Dir: "C:\\Test", MenuName: "My Tools"}
	if s.Name() != "customdir:my-tools" {
		t.Errorf("unexpected Name(): %q", s.Name())
	}
}

func TestCustomDirSource_Category(t *testing.T) {
	s := &CustomDirSource{Dir: "C:\\Test", MenuName: "Utilities"}
	if s.Category() != "Utilities" {
		t.Errorf("unexpected Category(): %q", s.Category())
	}
}

// Root-level files are all kept, even those whose names look like arch variants.
// (Each file in the root is assumed to be a distinct standalone tool.)
func TestCustomDirSource_Discover_RootLevelKeepsAll(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"putty.exe", "puttygen.exe", "pagent.exe", "WinSCP.exe", "rufus-4.12.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// These should be filtered by the existing isFilteredExecutable filter.
	for _, name := range []string{"uninstall.exe", "setup.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Utilities"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	want := map[string]bool{"putty": true, "puttygen": true, "pagent": true, "WinSCP": true, "rufus-4.12": true}
	if len(apps) != len(want) {
		t.Fatalf("expected %d apps, got %d: %v", len(want), len(apps), appNames(apps))
	}
	for _, a := range apps {
		if !want[a.Name] {
			t.Errorf("unexpected app name: %q", a.Name)
		}
	}
}

// Files inside a subdirectory are deduplicated: only one representative is kept.
// The display name is the relative path from the scan root (subdir\binary).
func TestCustomDirSource_Discover_SubdirDeduplication(t *testing.T) {
	dir := t.TempDir()

	// Simulate a TCPView-style directory: main exe + arch/console variants.
	tcpDir := filepath.Join(dir, "TCPView")
	if err := os.Mkdir(tcpDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tcpview.exe", "tcpview64.exe", "tcpview64a.exe", "tcpvcon.exe", "tcpvcon64.exe", "tcpvcon64a.exe"} {
		if err := os.WriteFile(filepath.Join(tcpDir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Utilities"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(apps) != 1 {
		t.Fatalf("expected 1 app from subdirectory (arch variants deduplicated), got %d: %v", len(apps), appNames(apps))
	}
	// Name should be relative path: TCPView\tcpview
	wantName := filepath.Join("TCPView", "tcpview")
	if apps[0].Name != wantName {
		t.Errorf("expected name %q, got %q", wantName, apps[0].Name)
	}
}

// Root-level exes and subdirectory exes coexist correctly.
// Subdirectory exe names include the relative path prefix.
func TestCustomDirSource_Discover_MixedRootAndSubdir(t *testing.T) {
	dir := t.TempDir()

	// Root-level tools (all kept, name = just binary)
	for _, name := range []string{"toplevel.exe", "another.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Subdirectory with variants (only main kept, name = subdir\binary)
	sub := filepath.Join(dir, "toolbox")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"toolbox.exe", "toolbox64.exe", "toolboxcmd.exe"} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Tools"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// 2 root-level + 1 from subdirectory = 3 total
	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d: %v", len(apps), appNames(apps))
	}

	names := make(map[string]bool)
	for _, a := range apps {
		names[a.Name] = true
	}
	wantSubdir := filepath.Join("toolbox", "toolbox")
	if !names["toplevel"] || !names["another"] || !names[wantSubdir] {
		t.Errorf("unexpected names: %v", appNames(apps))
	}
}

func TestCustomDirSource_Discover_SpaceInPath(t *testing.T) {
	dir := t.TempDir()
	spaceDir := filepath.Join(dir, "my tools")
	if err := os.Mkdir(spaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spaceDir, "neat.exe"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Tools"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	if !strings.HasPrefix(apps[0].Exec, `"`) || !strings.HasSuffix(apps[0].Exec, `"`) {
		t.Errorf("expected quoted exec path for path with spaces, got %q", apps[0].Exec)
	}
}

func TestCustomDirSource_Exclude(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"rufus-4.12.exe", "rufus-3.99.exe", "putty.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Utilities", Exclude: []string{"rufus*"}}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(apps) != 1 || apps[0].Name != "putty" {
		t.Fatalf("expected only putty after Exclude filter, got %v", appNames(apps))
	}
}

// --- isArchDirName / collapseArchDirs / cleanRelPath Tests ---

func TestIsArchDirName(t *testing.T) {
	archNames := []string{"x64", "x86", "arm", "arm64", "amd64", "win32", "win64", "i386", "i686"}
	for _, n := range archNames {
		if !isArchDirName(n) {
			t.Errorf("expected isArchDirName(%q) == true", n)
		}
	}
	notArch := []string{"TCPView", "toolbox", "bin", "lib", "app", "WinDirStat"}
	for _, n := range notArch {
		if isArchDirName(strings.ToLower(n)) {
			t.Errorf("expected isArchDirName(%q) == false", strings.ToLower(n))
		}
	}
}

func TestCleanRelPath(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		input string
		want  string
	}{
		{"putty", "putty"},
		{"TCPView" + sep + "tcpview", "TCPView" + sep + "tcpview"},
		{"WinDirStat" + sep + "x64" + sep + "windirstat", "WinDirStat" + sep + "windirstat"},
		{"WinDirStat" + sep + "arm64" + sep + "windirstat", "WinDirStat" + sep + "windirstat"},
		{"app" + sep + "x86" + sep + "sub" + sep + "tool", "app" + sep + "sub" + sep + "tool"},
	}
	for _, tt := range tests {
		got := cleanRelPath(tt.input)
		if got != tt.want {
			t.Errorf("cleanRelPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// WinDirStat-style: multiple arch-named subdirs, only the best arch is kept.
func TestCustomDirSource_Discover_ArchDirCollapse(t *testing.T) {
	dir := t.TempDir()

	wdDir := filepath.Join(dir, "WinDirStat")
	if err := os.Mkdir(wdDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"arm", "x64", "x86", "arm64"} {
		archDir := filepath.Join(wdDir, arch)
		if err := os.Mkdir(archDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(archDir, "windirstat.exe"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Utilities"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(apps) != 1 {
		t.Fatalf("expected 1 app (arch dirs collapsed), got %d: %v", len(apps), appNames(apps))
	}
	// Should prefer x64 and strip the arch component from the name.
	wantName := filepath.Join("WinDirStat", "windirstat")
	if apps[0].Name != wantName {
		t.Errorf("expected name %q, got %q", wantName, apps[0].Name)
	}
	// Exec should point to the x64 binary.
	if !strings.Contains(apps[0].Exec, "x64") {
		t.Errorf("expected exec to point to x64 binary, got %q", apps[0].Exec)
	}
}

// Mixed: arch-dir app sits alongside a normal-subdir app and root-level tools.
func TestCustomDirSource_Discover_ArchDirMixed(t *testing.T) {
	dir := t.TempDir()

	// Root-level standalone tool
	if err := os.WriteFile(filepath.Join(dir, "putty.exe"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	// Normal subdirectory (should NOT be collapsed)
	tcpDir := filepath.Join(dir, "TCPView")
	if err := os.Mkdir(tcpDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tcpDir, "tcpview.exe"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	// Arch-subdirectory app
	wdDir := filepath.Join(dir, "WinDirStat")
	if err := os.Mkdir(wdDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"x64", "x86"} {
		archDir := filepath.Join(wdDir, arch)
		if err := os.Mkdir(archDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(archDir, "windirstat.exe"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := &CustomDirSource{Dir: dir, MenuName: "Utilities"}
	apps, err := s.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// putty + TCPView\tcpview + WinDirStat\windirstat = 3
	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d: %v", len(apps), appNames(apps))
	}
	names := make(map[string]bool)
	for _, a := range apps {
		names[a.Name] = true
	}
	wantTCP := filepath.Join("TCPView", "tcpview")
	wantWDS := filepath.Join("WinDirStat", "windirstat")
	if !names["putty"] || !names[wantTCP] || !names[wantWDS] {
		t.Errorf("unexpected app names: %v", appNames(apps))
	}
}

func TestIsArchVariant(t *testing.T) {
	tests := []struct {
		base     string
		expected bool
	}{
		{"tcpview64", true},
		{"tcpview64a", true},
		{"tcpvcon", true},
		{"app_x64", true},
		{"app_x86", true},
		{"app_64", true},
		{"app_32", true},
		{"appcmd", true},
		{"appcli", true},
		{"tcpview", false},
		{"putty", false},
		{"rufus-4.12", false}, // version numbers should not be flagged
		{"WinSCP", false},
	}
	for _, tt := range tests {
		got := isArchVariant(strings.ToLower(tt.base))
		if got != tt.expected {
			t.Errorf("isArchVariant(%q) = %v, want %v", tt.base, got, tt.expected)
		}
	}
}

func TestPickMainExe(t *testing.T) {
	tests := []struct {
		name string
		input []string
		want string // just the filename, not full path
	}{
		{
			name:  "picks non-variant over variant",
			input: []string{`C:\t\tcpview64.exe`, `C:\t\tcpview.exe`, `C:\t\tcpvcon.exe`},
			want:  "tcpview.exe",
		},
		{
			name:  "picks shortest when multiple non-variants",
			input: []string{`C:\t\myapp.exe`, `C:\t\myapp-extra.exe`},
			want:  "myapp.exe",
		},
		{
			name:  "falls back to shortest when all are variants",
			input: []string{`C:\t\app64.exe`, `C:\t\app_x64.exe`},
			want:  "app64.exe",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filepath.Base(pickMainExe(tt.input))
			if got != tt.want {
				t.Errorf("pickMainExe = %q, want %q", got, tt.want)
			}
		})
	}
}

// appNames returns the Name field of each DiscoveredApp for use in test error messages.
func appNames(apps []discover.DiscoveredApp) []string {
	names := make([]string, len(apps))
	for i, a := range apps {
		names[i] = a.Name
	}
	return names
}
