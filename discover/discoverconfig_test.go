package discover

import (
	"testing"
)

func TestParseDiscoverConfig_Basic(t *testing.T) {
	yaml := `
title: "My Config"
discover:
  dirs:
    - dir: "F:\\Utilities"
      name: "Utilities"
    - dir: "C:\\Tools"
      name: "My Tools"
      exclude:
        - "*64*"
        - "setup*"
`
	cfg, err := ParseDiscoverConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d", len(cfg.Dirs))
	}
	if cfg.Dirs[0].Dir != `F:\Utilities` || cfg.Dirs[0].Name != "Utilities" {
		t.Errorf("unexpected first entry: %+v", cfg.Dirs[0])
	}
	if len(cfg.Dirs[0].Exclude) != 0 {
		t.Errorf("expected no exclude patterns on first entry, got %v", cfg.Dirs[0].Exclude)
	}
	if cfg.Dirs[1].Dir != `C:\Tools` || cfg.Dirs[1].Name != "My Tools" {
		t.Errorf("unexpected second entry: %+v", cfg.Dirs[1])
	}
	if len(cfg.Dirs[1].Exclude) != 2 || cfg.Dirs[1].Exclude[0] != "*64*" || cfg.Dirs[1].Exclude[1] != "setup*" {
		t.Errorf("unexpected exclude patterns on second entry: %v", cfg.Dirs[1].Exclude)
	}
}

func TestParseDiscoverConfig_MissingBlock(t *testing.T) {
	yaml := `
title: "My Config"
theme: "dark"
`
	cfg, err := ParseDiscoverConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Dirs) != 0 {
		t.Fatalf("expected 0 dirs, got %d", len(cfg.Dirs))
	}
}

func TestParseDiscoverConfig_EmptyDirs(t *testing.T) {
	yaml := `
discover:
  dirs: []
`
	cfg, err := ParseDiscoverConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Dirs) != 0 {
		t.Fatalf("expected 0 dirs, got %d", len(cfg.Dirs))
	}
}

func TestParseDiscoverConfig_InvalidYAML(t *testing.T) {
	_, err := ParseDiscoverConfig([]byte(": : {bad yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseDiscoverConfig_IgnoresOtherKeys(t *testing.T) {
	yaml := `
title: "Test"
menus:
  foo:
    title: "Foo"
    items: []
discover:
  dirs:
    - dir: "C:\\Apps"
      name: "Apps"
`
	cfg, err := ParseDiscoverConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d", len(cfg.Dirs))
	}
}
