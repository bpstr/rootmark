package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenConfigIsMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Build.Source != "." || cfg.Build.Output != "_site" || cfg.Site.Language != "en" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	if err := os.WriteFile(path, []byte("site:\n  title: Test\nbuild:\n  source: docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.Title != "Test" || cfg.Build.Source != "docs" || cfg.Build.Output != "_site" || cfg.Site.Language != "en" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
