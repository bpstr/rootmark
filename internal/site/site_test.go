package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpstr/rootmark/internal/config"
)

func TestBuildRendersMarkdownRoutes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "content")
	output := filepath.Join(root, "site")

	if err := os.MkdirAll(filepath.Join(source, "guide"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "index.md"), []byte("# Hello\n\nHome page."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "guide", "start.md"), []byte("---\ntitle: Start here\ndescription: First steps\n---\n\n# Ignored as title\n\nGuide."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Repository readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Site.Title = "Example"
	cfg.Build.Source = source
	cfg.Build.Output = output

	result, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Pages != 2 {
		t.Fatalf("expected 2 pages, got %d", result.Pages)
	}

	index, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "<h1 id=\"hello\">Hello</h1>") {
		t.Fatalf("index was not rendered: %s", index)
	}

	guide, err := os.ReadFile(filepath.Join(output, "guide", "start", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), "<title>Start here · Example</title>") || !strings.Contains(string(guide), "name=\"description\" content=\"First steps\"") {
		t.Fatalf("frontmatter was not rendered: %s", guide)
	}

	if _, err := os.Stat(filepath.Join(output, "README", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("README.md should be ignored")
	}
}

func TestBuildRejectsMissingFrontMatterDelimiter(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("---\ntitle: Broken\n# Body"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Build.Source = root
	cfg.Build.Output = filepath.Join(root, "_site")

	_, err := Build(cfg)
	if err == nil || !strings.Contains(err.Error(), "missing closing") {
		t.Fatalf("expected frontmatter error, got %v", err)
	}
}
