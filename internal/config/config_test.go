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
	if cfg.Build.Source != "." || cfg.Build.Output != "_site" || cfg.Site.Language != "en" || cfg.Theme != "" || cfg.Preset != "simple" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Primary.Style != "button" || !cfg.Footer.GeneratedWith || cfg.FeedEnabled() || cfg.SitemapEnabled() {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadBlogPresetEnablesPublicationDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	if err := os.WriteFile(path, []byte("preset: blog\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsBlog() || !cfg.FeedEnabled() || !cfg.SitemapEnabled() {
		t.Fatalf("blog defaults were not enabled: %+v", cfg)
	}
}

func TestLoadAllowsBlogPublicationDefaultsToBeDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	if err := os.WriteFile(path, []byte("preset: blog\nfeed: false\nsitemap: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FeedEnabled() || cfg.SitemapEnabled() {
		t.Fatalf("explicit publication settings were ignored: %+v", cfg)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	data := `site:
  title: Test
  url: https://example.com
  favicon: favicon.svg
build:
  source: docs
theme: https://github.com/example/rootmark-theme
navigation:
  - label: Docs
    url: /docs/
primary:
  label: Open app
  url: https://example.com/app
  style: link
footer:
  generated_with: false
  developed_by:
    label: Example
    url: https://example.com
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.Title != "Test" || cfg.Site.URL != "https://example.com" || cfg.Site.Favicon != "favicon.svg" {
		t.Fatalf("unexpected site config: %+v", cfg.Site)
	}
	if cfg.Build.Source != "docs" || cfg.Build.Output != "_site" || cfg.Theme != "https://github.com/example/rootmark-theme" {
		t.Fatalf("unexpected build config: %+v", cfg)
	}
	if len(cfg.Navigation) != 1 || cfg.Primary.Style != "link" || cfg.Footer.GeneratedWith || cfg.Footer.DevelopedBy.Label != "Example" {
		t.Fatalf("unexpected chrome config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPrimaryStyle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	if err := os.WriteFile(path, []byte("primary:\n  style: loud\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid primary style to fail")
	}
}

func TestLoadRejectsUnknownPreset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rootmark.yml")
	if err := os.WriteFile(path, []byte("preset: magazine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown preset to fail")
	}
}
