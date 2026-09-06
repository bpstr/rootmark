package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpstr/rootmark/internal/config"
)

func TestBuildRendersMarkdownRoutesAndSiteChrome(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(source, "favicon.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Repository readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Site.Title = "Example"
	cfg.Site.URL = "https://example.com/docs/"
	cfg.Site.Author = "Example Author"
	cfg.Site.Favicon = "/favicon.svg"
	cfg.Navigation = []config.Link{{Label: "Guide", URL: "/guide/start/"}}
	cfg.Primary = config.CTA{Label: "GitHub", URL: "https://github.com/example", Style: "button"}
	cfg.Footer.Text = "Example footer"
	cfg.Footer.DevelopedBy = config.Link{Label: "Example", URL: "https://example.com"}
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
	indexHTML := string(index)
	for _, expected := range []string{
		"<h1 id=\"hello\">Hello</h1>",
		"<link rel=\"icon\" href=\"favicon.svg\">",
		"<meta name=\"author\" content=\"Example Author\">",
		"href=\"guide/start/\">Guide</a>",
		"Generated with <a href=\"https://github.com/bpstr/rootmark\">Rootmark</a>",
	} {
		if !strings.Contains(indexHTML, expected) {
			t.Fatalf("index missing %q: %s", expected, indexHTML)
		}
	}

	guide, err := os.ReadFile(filepath.Join(output, "guide", "start", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	guideHTML := string(guide)
	if !strings.Contains(guideHTML, "<title>Start here · Example</title>") || !strings.Contains(guideHTML, "name=\"description\" content=\"First steps\"") {
		t.Fatalf("frontmatter was not rendered: %s", guideHTML)
	}
	if !strings.Contains(guideHTML, "href=\"../../favicon.svg\"") || !strings.Contains(guideHTML, "href=\"../../guide/start/\">Guide</a>") {
		t.Fatalf("nested site URLs were not resolved: %s", guideHTML)
	}
	if !strings.Contains(guideHTML, "href=\"https://example.com/docs/guide/start/\"") {
		t.Fatalf("canonical URL was not rendered: %s", guideHTML)
	}

	asset, err := os.ReadFile(filepath.Join(output, "favicon.svg"))
	if err != nil || string(asset) != "<svg></svg>" {
		t.Fatalf("static asset was not copied: %v %q", err, asset)
	}
	if _, err := os.Stat(filepath.Join(output, "README.md")); !os.IsNotExist(err) {
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

func TestRoutePath(t *testing.T) {
	cases := map[string]string{
		"index.md":       "/",
		"about.md":       "/about/",
		"docs/index.md":  "/docs/",
		"docs/start.md":  "/docs/start/",
		"deep/a/file.md": "/deep/a/file/",
	}
	for input, expected := range cases {
		if actual := routePath(input); actual != expected {
			t.Fatalf("routePath(%q) = %q, expected %q", input, actual, expected)
		}
	}
}
