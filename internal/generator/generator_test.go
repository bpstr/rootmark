package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpstr/rootmark/internal/config"
)

func TestBuildGeneratesRobotsForPublishedSite(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "_site")
	if err := os.WriteFile(filepath.Join(root, "post.md"), []byte("---\ndate: 2026-09-01\n---\n\n# Post\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Preset = "blog"
	cfg.Site.Title = "Example"
	cfg.Site.URL = "https://example.com/blog/"
	cfg.Build.Source = root
	cfg.Build.Output = output

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	robots, err := os.ReadFile(filepath.Join(output, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(robots)
	if !strings.Contains(text, "User-agent: *") || !strings.Contains(text, "Allow: /") || !strings.Contains(text, "Sitemap: https://example.com/blog/sitemap.xml") {
		t.Fatalf("unexpected robots.txt: %s", text)
	}
}

func TestBuildPreservesCustomRobots(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "_site")
	if err := os.WriteFile(filepath.Join(root, "post.md"), []byte("---\ndate: 2026-09-01\n---\n\n# Post\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "robots.txt"), []byte("User-agent: *\nDisallow: /private/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Preset = "blog"
	cfg.Site.URL = "https://example.com/"
	cfg.Build.Source = root
	cfg.Build.Output = output

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	robots, err := os.ReadFile(filepath.Join(output, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(robots) != "User-agent: *\nDisallow: /private/\n" {
		t.Fatalf("custom robots.txt was overwritten: %s", robots)
	}
}
