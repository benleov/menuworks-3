package discover

import (
	"bytes"
	"strings"
	"testing"
)

// --- Mock Source for testing ---

type mockSource struct {
	name      string
	category  string
	available bool
	apps      []DiscoveredApp
	err       error
}

func (m *mockSource) Name() string     { return m.name }
func (m *mockSource) Category() string { return m.category }
func (m *mockSource) Available() bool  { return m.available }
func (m *mockSource) Discover() ([]DiscoveredApp, error) {
	return m.apps, m.err
}

// --- Registry Tests ---

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.Sources()) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(r.Sources()))
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	s := &mockSource{name: "test", category: "Test", available: true}
	r.Register(s)

	sources := r.Sources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name() != "test" {
		t.Fatalf("expected source name 'test', got '%s'", sources[0].Name())
	}
}

func TestRegistryRegisterMultiple(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "a", category: "A", available: true})
	r.Register(&mockSource{name: "b", category: "B", available: false})
	r.Register(&mockSource{name: "c", category: "C", available: true})

	if len(r.Sources()) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(r.Sources()))
	}
}

func TestRegistryAvailableSources(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "a", available: true})
	r.Register(&mockSource{name: "b", available: false})
	r.Register(&mockSource{name: "c", available: true})

	avail := r.AvailableSources()
	if len(avail) != 2 {
		t.Fatalf("expected 2 available sources, got %d", len(avail))
	}
	names := []string{avail[0].Name(), avail[1].Name()}
	if names[0] != "a" || names[1] != "c" {
		t.Fatalf("unexpected available sources: %v", names)
	}
}

func TestRegistryAvailableSourcesEmpty(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "a", available: false})

	avail := r.AvailableSources()
	if len(avail) != 0 {
		t.Fatalf("expected 0 available sources, got %d", len(avail))
	}
}

func TestRegistrySourceByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "steam"})
	r.Register(&mockSource{name: "startmenu"})

	s := r.SourceByName("steam")
	if s == nil {
		t.Fatal("SourceByName returned nil for 'steam'")
	}
	if s.Name() != "steam" {
		t.Fatalf("expected 'steam', got '%s'", s.Name())
	}
}

func TestRegistrySourceByNameCaseInsensitive(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "Steam"})

	s := r.SourceByName("STEAM")
	if s == nil {
		t.Fatal("SourceByName should be case-insensitive")
	}
}

func TestRegistrySourceByNameNotFound(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "steam"})

	s := r.SourceByName("nonexistent")
	if s != nil {
		t.Fatal("SourceByName should return nil for unknown source")
	}
}

func TestSourcesCopyIsolation(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "a"})
	sources := r.Sources()
	sources[0] = nil // mutate the copy
	original := r.Sources()
	if original[0] == nil {
		t.Fatal("Sources() should return a copy, not the internal slice")
	}
}

// --- DiscoverAll Tests ---

func TestDiscoverAllNoFilter(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "src1",
		available: true,
		apps: []DiscoveredApp{
			{Name: "App1", Exec: "app1.exe", Source: "src1", Category: "Cat1"},
		},
	})
	r.Register(&mockSource{
		name:      "src2",
		available: true,
		apps: []DiscoveredApp{
			{Name: "App2", Exec: "app2.exe", Source: "src2", Category: "Cat2"},
		},
	})

	results, err := r.DiscoverAll(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestDiscoverAllWithFilter(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "src1",
		available: true,
		apps:      []DiscoveredApp{{Name: "App1", Exec: "app1"}},
	})
	r.Register(&mockSource{
		name:      "src2",
		available: true,
		apps:      []DiscoveredApp{{Name: "App2", Exec: "app2"}},
	})

	results, err := r.DiscoverAll([]string{"src1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Source != "src1" {
		t.Fatalf("expected source 'src1', got '%s'", results[0].Source)
	}
}

func TestDiscoverAllUnknownSource(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{name: "src1", available: true})

	_, err := r.DiscoverAll([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
	if !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDiscoverAllSkipsUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "unavail",
		available: false,
		apps:      []DiscoveredApp{{Name: "Hidden"}},
	})

	results, err := r.DiscoverAll(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for unavailable source, got %d", len(results))
	}
}

func TestDiscoverAllPropagatesSourceError(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "failing",
		available: true,
		err:       errTest,
	})

	results, err := r.DiscoverAll(nil)
	if err != nil {
		t.Fatalf("DiscoverAll should not return error for source-level errors: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected error in result")
	}
}

var errTest = &testError{msg: "test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// --- CollectApps Tests ---

func TestCollectAppsSorted(t *testing.T) {
	results := []DiscoverResult{
		{Source: "s1", Apps: []DiscoveredApp{
			{Name: "Zebra", Category: "B"},
			{Name: "Alpha", Category: "B"},
		}},
		{Source: "s2", Apps: []DiscoveredApp{
			{Name: "Gamma", Category: "A"},
		}},
	}

	apps := CollectApps(results)
	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(apps))
	}
	// Sorted: A/Gamma, B/Alpha, B/Zebra
	if apps[0].Name != "Gamma" || apps[0].Category != "A" {
		t.Fatalf("expected first app 'Gamma' in category 'A', got '%s' in '%s'", apps[0].Name, apps[0].Category)
	}
	if apps[1].Name != "Alpha" || apps[2].Name != "Zebra" {
		t.Fatalf("expected sorted order Alpha, Zebra in category B")
	}
}

func TestCollectAppsSkipsErrors(t *testing.T) {
	results := []DiscoverResult{
		{Source: "s1", Apps: []DiscoveredApp{{Name: "Good"}}, Err: nil},
		{Source: "s2", Apps: nil, Err: errTest},
	}

	apps := CollectApps(results)
	if len(apps) != 1 {
		t.Fatalf("expected 1 app (skipping errored source), got %d", len(apps))
	}
}

func TestCollectAppsEmpty(t *testing.T) {
	apps := CollectApps(nil)
	if len(apps) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(apps))
	}
}

// --- GroupByCategory Tests ---

func TestGroupByCategory(t *testing.T) {
	apps := []DiscoveredApp{
		{Name: "A", Category: "Games"},
		{Name: "B", Category: "Apps"},
		{Name: "C", Category: "Games"},
	}

	groups := GroupByCategory(apps)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["Games"]) != 2 {
		t.Fatalf("expected 2 items in Games, got %d", len(groups["Games"]))
	}
	if len(groups["Apps"]) != 1 {
		t.Fatalf("expected 1 item in Apps, got %d", len(groups["Apps"]))
	}
}

func TestGroupByCategoryEmpty(t *testing.T) {
	groups := GroupByCategory(nil)
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}

// --- DeduplicateApps Tests ---

func TestDeduplicateApps(t *testing.T) {
	apps := []DiscoveredApp{
		{Name: "App1", Exec: "app1.exe"},
		{Name: "App1 Duplicate", Exec: "APP1.EXE"}, // same exec, different case
		{Name: "App2", Exec: "app2.exe"},
	}

	deduped := DeduplicateApps(apps)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 apps after dedup, got %d", len(deduped))
	}
	if deduped[0].Name != "App1" {
		t.Fatalf("expected first occurrence to be kept, got '%s'", deduped[0].Name)
	}
}

func TestDeduplicateAppsNoDuplicates(t *testing.T) {
	apps := []DiscoveredApp{
		{Name: "A", Exec: "a.exe"},
		{Name: "B", Exec: "b.exe"},
	}

	deduped := DeduplicateApps(apps)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(deduped))
	}
}

func TestDeduplicateAppsEmpty(t *testing.T) {
	deduped := DeduplicateApps(nil)
	if len(deduped) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(deduped))
	}
}

// --- Writer Tests ---

func TestRenderConfigBasic(t *testing.T) {
	// Override OS for deterministic output
	origOS := writerOS
	writerOS = "windows"
	defer func() { writerOS = origOS }()

	apps := []DiscoveredApp{
		{Name: "Notepad++", Exec: `start "" "C:\Program Files\Notepad++\notepad++.exe"`, Category: "Applications"},
		{Name: "Half-Life 2", Exec: "start steam://rungameid/220", Category: "Games"},
	}

	var buf bytes.Buffer
	err := RenderConfig(apps, &buf)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	output := buf.String()

	// Verify structure
	if !strings.Contains(output, `title: "MenuWorks 3.X"`) {
		t.Error("missing title")
	}
	if !strings.Contains(output, `theme: "dark"`) {
		t.Error("missing theme")
	}
	if !strings.Contains(output, `target: "applications"`) {
		t.Error("missing applications submenu")
	}
	if !strings.Contains(output, `target: "games"`) {
		t.Error("missing games submenu")
	}
	if !strings.Contains(output, `label: "Notepad++"`) {
		t.Error("missing Notepad++ label")
	}
	if !strings.Contains(output, `label: "Half-Life 2"`) {
		t.Error("missing Half-Life 2 label")
	}
	if !strings.Contains(output, `windows:`) {
		t.Error("missing windows exec key")
	}
	if !strings.Contains(output, `label: "Quit"`) {
		t.Error("missing Quit item")
	}
	if !strings.Contains(output, `label: "Back"`) {
		t.Error("missing Back item")
	}
}

func TestRenderConfigMultipleCategories(t *testing.T) {
	origOS := writerOS
	writerOS = "windows"
	defer func() { writerOS = origOS }()

	apps := []DiscoveredApp{
		{Name: "Game1", Exec: "game1", Category: "Games"},
		{Name: "App1", Exec: "app1", Category: "Applications"},
		{Name: "Game2", Exec: "game2", Category: "Games"},
	}

	var buf bytes.Buffer
	err := RenderConfig(apps, &buf)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	output := buf.String()

	// Applications should come before Games (alphabetical)
	appIdx := strings.Index(output, `"Applications"`)
	gameIdx := strings.Index(output, `"Games"`)
	if appIdx < 0 || gameIdx < 0 {
		t.Fatal("missing category references")
	}
	if appIdx > gameIdx {
		t.Error("categories should be sorted alphabetically")
	}
}

func TestRenderConfigEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := RenderConfig(nil, &buf)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `title: "MenuWorks 3.X"`) {
		t.Error("empty config should still have a title")
	}
	if !strings.Contains(output, `label: "Quit"`) {
		t.Error("empty config should still have Quit")
	}
}

func TestRenderConfigEscaping(t *testing.T) {
	origOS := writerOS
	writerOS = "windows"
	defer func() { writerOS = origOS }()

	apps := []DiscoveredApp{
		{Name: `App "with" quotes`, Exec: `start "" "C:\path\to\app.exe"`, Category: "Test"},
	}

	var buf bytes.Buffer
	err := RenderConfig(apps, &buf)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	output := buf.String()
	// Quotes and backslashes should be escaped
	if !strings.Contains(output, `App \"with\" quotes`) {
		t.Errorf("expected escaped quotes in output, got: %s", output)
	}
}

// --- sanitizeID Tests ---

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Games", "games"},
		{"System Tools", "system_tools"},
		{"Applications", "applications"},
		{"My Apps!!", "my_apps"},
		{"  spaces  ", "spaces"},
		{"123-Numbers", "123_numbers"},
		{"A--B  C", "a_b_c"},
	}

	for _, tc := range tests {
		got := sanitizeID(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeID(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

// --- escapeYAMLString Tests ---

func TestEscapeYAMLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`simple`, `simple`},
		{`with "quotes"`, `with \"quotes\"`},
		{`back\slash`, `back\\slash`},
		{`C:\Program Files\app.exe`, `C:\\Program Files\\app.exe`},
		{`"quoted" and \slashed\`, `\"quoted\" and \\slashed\\`},
	}

	for _, tc := range tests {
		got := escapeYAMLString(tc.input)
		if got != tc.expected {
			t.Errorf("escapeYAMLString(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

// --- Integration Test: End-to-End ---

func TestEndToEndDiscoverAndRender(t *testing.T) {
	origOS := writerOS
	writerOS = "windows"
	defer func() { writerOS = origOS }()

	r := NewRegistry()
	r.Register(&mockSource{
		name:      "steam",
		category:  "Games",
		available: true,
		apps: []DiscoveredApp{
			{Name: "Portal 2", Exec: "start steam://rungameid/620", Source: "steam", Category: "Games"},
			{Name: "Half-Life 2", Exec: "start steam://rungameid/220", Source: "steam", Category: "Games"},
		},
	})
	r.Register(&mockSource{
		name:      "startmenu",
		category:  "Applications",
		available: true,
		apps: []DiscoveredApp{
			{Name: "Notepad++", Exec: `start "" "C:\notepad++.exe"`, Source: "startmenu", Category: "Applications"},
			{Name: "Firefox", Exec: `start "" "C:\firefox.exe"`, Source: "startmenu", Category: "Applications"},
		},
	})

	results, err := r.DiscoverAll(nil)
	if err != nil {
		t.Fatalf("DiscoverAll failed: %v", err)
	}

	apps := CollectApps(results)
	apps = DeduplicateApps(apps)

	if len(apps) != 4 {
		t.Fatalf("expected 4 apps, got %d", len(apps))
	}

	var buf bytes.Buffer
	err = RenderConfig(apps, &buf)
	if err != nil {
		t.Fatalf("RenderConfig failed: %v", err)
	}

	output := buf.String()

	// Verify the generated YAML has proper structure
	expectedStrings := []string{
		`title: "MenuWorks 3.X"`,
		`target: "applications"`,
		`target: "games"`,
		`label: "Firefox"`,
		`label: "Notepad++"`,
		`label: "Half-Life 2"`,
		`label: "Portal 2"`,
		`label: "Quit"`,
		`label: "Back"`,
		`type: submenu`,
		`type: command`,
		`type: separator`,
		`type: back`,
	}
	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("expected output to contain %q", s)
		}
	}
}

func TestEndToEndWithSourceFilter(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "steam",
		category:  "Games",
		available: true,
		apps:      []DiscoveredApp{{Name: "Game", Exec: "game", Category: "Games"}},
	})
	r.Register(&mockSource{
		name:      "startmenu",
		category:  "Applications",
		available: true,
		apps:      []DiscoveredApp{{Name: "App", Exec: "app", Category: "Applications"}},
	})

	results, err := r.DiscoverAll([]string{"steam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apps := CollectApps(results)
	if len(apps) != 1 {
		t.Fatalf("expected 1 app with filter, got %d", len(apps))
	}
	if apps[0].Name != "Game" {
		t.Fatalf("expected 'Game', got '%s'", apps[0].Name)
	}
}

func TestEndToEndDeduplication(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSource{
		name:      "src1",
		category:  "Apps",
		available: true,
		apps: []DiscoveredApp{
			{Name: "App via src1", Exec: "same.exe", Category: "Apps"},
		},
	})
	r.Register(&mockSource{
		name:      "src2",
		category:  "Apps",
		available: true,
		apps: []DiscoveredApp{
			{Name: "App via src2", Exec: "SAME.EXE", Category: "Apps"}, // same exec, different case
		},
	})

	results, _ := r.DiscoverAll(nil)
	apps := CollectApps(results)
	apps = DeduplicateApps(apps)

	if len(apps) != 1 {
		t.Fatalf("expected 1 app after dedup, got %d", len(apps))
	}
}
