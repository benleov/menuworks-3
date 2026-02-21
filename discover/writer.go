package discover

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"
)

// writerOS is the OS identifier to use for exec commands in generated YAML.
// It can be overridden in tests.
var writerOS = detectOS()

func detectOS() string {
	// Use build-time GOOS detection
	switch os := strings.ToLower(getGOOS()); os {
	case "darwin":
		return "mac"
	case "linux":
		return "linux"
	default:
		return "windows"
	}
}

// configTemplate is the Go template for generating config.yaml
const configTemplate = `title: "MenuWorks 3.X"
theme: "dark"
themes:
  dark:
    background: "blue"
    text: "silver"
    border: "aqua"
    highlight_bg: "navy"
    highlight_fg: "white"
    hotkey: "yellow"
    shadow: "gray"
    disabled: "gray"

items:
{{- range .Categories }}
  - type: submenu
    label: "{{ .Name }}"
    target: "{{ .ID }}"
{{- end }}
  - type: separator
  - type: back
    label: "Quit"

menus:
{{- range .Categories }}
  {{ .ID }}:
    title: "{{ .Name }}"
    items:
{{- range .Apps }}
      - type: command
        label: "{{ .Label }}"
        exec:
          {{ .OSKey }}: "{{ .Exec }}"
{{- end }}
      - type: back
        label: "Back"
{{ end -}}
`

// categoryData holds template data for a single category.
type categoryData struct {
	Name string
	ID   string
	Apps []appData
}

// appData holds template data for a single application.
type appData struct {
	Label string
	OSKey string
	Exec  string
}

// templateData holds the full template context.
type templateData struct {
	Categories []categoryData
}

// WriteConfig generates a MenuWorks config.yaml from discovered apps and writes it to the given path.
// If the file already exists, it returns an error (use WriteConfigMerge for merge behavior).
func WriteConfig(apps []DiscoveredApp, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	return RenderConfig(apps, f)
}

// WriteConfigStdout generates a MenuWorks config.yaml and writes it to stdout.
func WriteConfigStdout(apps []DiscoveredApp) error {
	return RenderConfig(apps, os.Stdout)
}

// RenderConfig generates the config YAML from apps and writes to w.
func RenderConfig(apps []DiscoveredApp, w io.Writer) error {
	data := buildTemplateData(apps)

	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

// buildTemplateData transforms discovered apps into template-ready data.
func buildTemplateData(apps []DiscoveredApp) templateData {
	groups := GroupByCategory(apps)

	// Sort category names for deterministic output
	var catNames []string
	for name := range groups {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	osKey := writerOS

	var categories []categoryData
	for _, name := range catNames {
		catApps := groups[name]
		id := sanitizeID(name)

		var items []appData
		for _, a := range catApps {
			items = append(items, appData{
				Label: escapeYAMLString(a.Name),
				OSKey: osKey,
				Exec:  escapeYAMLString(a.Exec),
			})
		}

		categories = append(categories, categoryData{
			Name: name,
			ID:   id,
			Apps: items,
		})
	}

	return templateData{Categories: categories}
}

// sanitizeID converts a display name to a YAML-safe menu ID.
// e.g. "System Tools" -> "system_tools"
func sanitizeID(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, s)
	// Collapse multiple underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	return s
}

// escapeYAMLString escapes characters that need escaping within a YAML double-quoted string.
func escapeYAMLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
