//go:build linux

package linux

import (
	"bufio"
	"strings"
	"testing"
)

// --- Desktop File Parsing Tests ---

func TestParseDesktopLine(t *testing.T) {
	tests := []struct {
		line  string
		key   string
		value string
	}{
		{"Name=Firefox", "Name", "Firefox"},
		{"Exec=firefox %u", "Exec", "firefox %u"},
		{"Type=Application", "Type", "Application"},
		{"NoDisplay=true", "NoDisplay", "true"},
		{"Name[fr]=Navigateur Web", "", ""}, // localized, should be skipped
		{"# comment", "", ""},
		{"", "", ""},
		{"Icon=firefox", "Icon", "firefox"},
		{"Terminal=false", "Terminal", "false"},
	}

	for _, tc := range tests {
		k, v := parseDesktopLine(tc.line)
		if k != tc.key || v != tc.value {
			t.Errorf("parseDesktopLine(%q) = (%q, %q), expected (%q, %q)", tc.line, k, v, tc.key, tc.value)
		}
	}
}

func TestCleanExecLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"firefox %u", "firefox"},
		{"gimp-2.10 %U", "gimp-2.10"},
		{"libreoffice --writer %F", "libreoffice --writer"},
		{"/usr/bin/nautilus --new-window %U", "/usr/bin/nautilus --new-window"},
		{"vlc", "vlc"},
		{"baobab %U", "baobab"},
	}

	for _, tc := range tests {
		got := cleanExecLine(tc.input)
		if got != tc.expected {
			t.Errorf("cleanExecLine(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseDesktopReader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string
		wantExec string
		wantNil  bool
	}{
		{
			name: "basic application",
			content: `[Desktop Entry]
Name=Calculator
Exec=gnome-calculator
Type=Application
Icon=gnome-calculator`,
			wantName: "Calculator",
			wantExec: "gnome-calculator",
		},
		{
			name: "with field codes",
			content: `[Desktop Entry]
Name=Files
Exec=nautilus --new-window %U
Type=Application`,
			wantName: "Files",
			wantExec: "nautilus --new-window",
		},
		{
			name: "NoDisplay entry",
			content: `[Desktop Entry]
Name=Hidden App
Exec=hiddenapp
Type=Application
NoDisplay=true`,
			wantNil: true,
		},
		{
			name: "Hidden entry",
			content: `[Desktop Entry]
Name=Hidden App
Exec=hiddenapp
Type=Application
Hidden=true`,
			wantNil: true,
		},
		{
			name: "terminal app",
			content: `[Desktop Entry]
Name=htop
Exec=htop
Type=Application
Terminal=true`,
			wantNil: true,
		},
		{
			name: "not an application",
			content: `[Desktop Entry]
Name=My Link
URL=https://example.com
Type=Link`,
			wantNil: true,
		},
		{
			name: "missing exec",
			content: `[Desktop Entry]
Name=NoExec App
Type=Application`,
			wantNil: true,
		},
		{
			name: "localized names ignored",
			content: `[Desktop Entry]
Name=Calculator
Name[fr]=Calculatrice
Exec=gnome-calculator
Type=Application`,
			wantName: "Calculator",
			wantExec: "gnome-calculator",
		},
		{
			name: "multiple sections",
			content: `[Desktop Entry]
Name=My App
Exec=myapp
Type=Application

[Desktop Action New]
Name=New Window
Exec=myapp --new`,
			wantName: "My App",
			wantExec: "myapp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tc.content))
			app, err := parseDesktopReader(scanner)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if app != nil {
					t.Errorf("expected nil, got %+v", app)
				}
				return
			}
			if app == nil {
				t.Fatal("expected app, got nil")
			}
			if app.Name != tc.wantName {
				t.Errorf("name = %q, want %q", app.Name, tc.wantName)
			}
			if app.Exec != tc.wantExec {
				t.Errorf("exec = %q, want %q", app.Exec, tc.wantExec)
			}
			if app.Source != "Desktop" {
				t.Errorf("source = %q, want %q", app.Source, "Desktop")
			}
			if app.Category != "Applications" {
				t.Errorf("category = %q, want %q", app.Category, "Applications")
			}
		})
	}
}

// --- Steam VDF Parsing Tests ---

func TestParseVDFLine(t *testing.T) {
	tests := []struct {
		line  string
		key   string
		value string
	}{
		{`		"appid"		"220"`, "appid", "220"},
		{`		"name"		"Half-Life 2"`, "name", "Half-Life 2"},
		{`		"Universe"		"1"`, "universe", "1"},
		{`	{`, "", ""},
		{`	}`, "", ""},
		{`"path"		"/home/user/.steam"`, "path", "/home/user/.steam"},
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
		name   string
		isTool bool
	}{
		{"Half-Life 2", false},
		{"Portal", false},
		{"Proton Experimental", true},
		{"Proton 8.0", true},
		{"Steam Linux Runtime 3.0 (sniper)", true},
		{"Steamworks Common Redistributables", true},
		{"DirectX Jun2010 Redist", true},
	}

	for _, tc := range tests {
		got := isSteamTool(tc.name)
		if got != tc.isTool {
			t.Errorf("isSteamTool(%q) = %v, expected %v", tc.name, got, tc.isTool)
		}
	}
}

// --- Flatpak Output Parsing Tests ---

func TestParseFlatpakOutput(t *testing.T) {
	output := "com.spotify.Client\tSpotify\n" +
		"org.mozilla.firefox\tFirefox\n" +
		"org.gimp.GIMP\tGNU Image Manipulation Program\n" +
		"\n"

	apps, err := parseFlatpakOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(apps))
	}

	expected := []struct {
		name string
		exec string
	}{
		{"Spotify", "flatpak run com.spotify.Client"},
		{"Firefox", "flatpak run org.mozilla.firefox"},
		{"GNU Image Manipulation Program", "flatpak run org.gimp.GIMP"},
	}

	for i, e := range expected {
		if apps[i].Name != e.name {
			t.Errorf("app[%d].Name = %q, want %q", i, apps[i].Name, e.name)
		}
		if apps[i].Exec != e.exec {
			t.Errorf("app[%d].Exec = %q, want %q", i, apps[i].Exec, e.exec)
		}
		if apps[i].Source != "Flatpak" {
			t.Errorf("app[%d].Source = %q, want %q", i, apps[i].Source, "Flatpak")
		}
	}
}

func TestParseFlatpakOutputEmpty(t *testing.T) {
	apps, err := parseFlatpakOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("expected 0 apps, got %d", len(apps))
	}
}

func TestParseFlatpakOutputDedupe(t *testing.T) {
	output := "com.spotify.Client\tSpotify\n" +
		"com.spotify.Client\tSpotify\n"

	apps, err := parseFlatpakOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("expected 1 app (deduped), got %d", len(apps))
	}
}

// --- Snap Output Parsing Tests ---

func TestParseSnapOutput(t *testing.T) {
	output := `Name      Version  Rev  Tracking       Publisher    Notes
firefox   128.0    123  latest/stable  mozilla      -
vlc       3.0.20   456  latest/stable  videolan     -
core22    20240111 789  latest/stable  canonical    base
snapd     2.63     101  latest/stable  canonical    snapd
`

	apps, err := parseSnapOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apps) != 2 {
		t.Fatalf("expected 2 apps (filtered system snaps), got %d: %+v", len(apps), apps)
	}

	if apps[0].Name != "firefox" {
		t.Errorf("app[0].Name = %q, want %q", apps[0].Name, "firefox")
	}
	if apps[0].Exec != "snap run firefox" {
		t.Errorf("app[0].Exec = %q, want %q", apps[0].Exec, "snap run firefox")
	}
	if apps[1].Name != "vlc" {
		t.Errorf("app[1].Name = %q, want %q", apps[1].Name, "vlc")
	}
}

func TestParseSnapOutputEmpty(t *testing.T) {
	output := `Name  Version  Rev  Tracking  Publisher  Notes
`
	apps, err := parseSnapOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("expected 0 apps, got %d", len(apps))
	}
}

func TestIsSystemSnap(t *testing.T) {
	tests := []struct {
		name     string
		isSystem bool
	}{
		{"firefox", false},
		{"vlc", false},
		{"core22", true},
		{"snapd", true},
		{"bare", true},
		{"gtk-common-themes", true},
		{"gnome-42-2204", true},
		{"my-custom-app", false},
	}

	for _, tc := range tests {
		got := isSystemSnap(tc.name)
		if got != tc.isSystem {
			t.Errorf("isSystemSnap(%q) = %v, expected %v", tc.name, got, tc.isSystem)
		}
	}
}
