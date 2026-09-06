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

func TestBlogPresetBuildsListingFeedsSitemapAndNavigation(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "_site")

	files := map[string]string{
		"older.md": `---
title: Older post
description: The older post.
date: 2026-09-01
tags: [go, static]
---

# Older post

Old content.
`,
		"newer.md": `---
title: Newer post
description: The newer post.
date: 2026-09-05
updated: 2026-09-06
image: cover.png
author: Guest Author
tags: [release]
---

# Newer post

New content.
`,
		"about.md": `---
title: About
---

# About

A normal undated page.
`,
		"draft.md": `---
title: Draft
date: 2026-09-07
draft: true
---

# Draft
`,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "cover.png"), []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Preset = "blog"
	cfg.Site.Title = "Example Blog"
	cfg.Site.Description = "Notes from Example."
	cfg.Site.URL = "https://example.com/blog/"
	cfg.Site.Author = "Example Author"
	cfg.Build.Source = root
	cfg.Build.Output = output

	result, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Pages != 5 {
		t.Fatalf("expected 5 rendered pages including generated index and 404, got %d", result.Pages)
	}

	index := mustRead(t, filepath.Join(output, "index.html"))
	newerAt := strings.Index(index, "Newer post")
	olderAt := strings.Index(index, "Older post")
	if newerAt == -1 || olderAt == -1 || newerAt >= olderAt {
		t.Fatalf("blog index is not newest-first: %s", index)
	}
	if strings.Contains(index, "Draft") || !strings.Contains(index, `href="newer/"`) || !strings.Contains(index, `href="rss.xml"`) {
		t.Fatalf("blog index publication metadata is wrong: %s", index)
	}

	newer := mustRead(t, filepath.Join(output, "newer", "index.html"))
	for _, expected := range []string{
		`property="og:type" content="article"`,
		`property="og:image" content="https://example.com/blog/cover.png"`,
		`property="article:published_time" content="2026-09-05"`,
		`property="article:modified_time" content="2026-09-06T00:00:00Z"`,
		`Guest Author`,
		`<small>Previous</small>Older post`,
	} {
		if !strings.Contains(newer, expected) {
			t.Fatalf("newer post missing %q: %s", expected, newer)
		}
	}
	older := mustRead(t, filepath.Join(output, "older", "index.html"))
	if !strings.Contains(older, `<small>Next</small>Newer post`) {
		t.Fatalf("older post is missing next navigation: %s", older)
	}

	rss := mustRead(t, filepath.Join(output, "rss.xml"))
	if !strings.Contains(rss, "<title>Example Blog</title>") || !strings.Contains(rss, "https://example.com/blog/newer/") || strings.Contains(rss, "About") || strings.Contains(rss, "Draft") {
		t.Fatalf("unexpected RSS feed: %s", rss)
	}
	if strings.Index(rss, "Newer post") >= strings.Index(rss, "Older post") {
		t.Fatalf("RSS feed is not newest-first: %s", rss)
	}

	atom := mustRead(t, filepath.Join(output, "atom.xml"))
	if !strings.Contains(atom, `xmlns="http://www.w3.org/2005/Atom"`) || !strings.Contains(atom, "Newer post") {
		t.Fatalf("unexpected Atom feed: %s", atom)
	}

	sitemap := mustRead(t, filepath.Join(output, "sitemap.xml"))
	for _, expected := range []string{
		"https://example.com/blog/",
		"https://example.com/blog/about/",
		"https://example.com/blog/newer/",
		"https://example.com/blog/older/",
	} {
		if !strings.Contains(sitemap, expected) {
			t.Fatalf("sitemap missing %q: %s", expected, sitemap)
		}
	}
	if strings.Contains(sitemap, "404.html") || strings.Contains(sitemap, "draft") {
		t.Fatalf("sitemap contains excluded pages: %s", sitemap)
	}

	if _, err := os.Stat(filepath.Join(output, "404.html")); err != nil {
		t.Fatalf("generated 404 page is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "draft", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("draft should not be rendered")
	}
}

func TestBlogPresetUsesExistingIndexAsIntroduction(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("# Journal\n\nA custom introduction."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "entry.md"), []byte("---\ndate: 2026-09-01\n---\n\n# Entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Preset = "blog"
	cfg.Site.Title = "Journal"
	cfg.Site.URL = "https://example.com/"
	cfg.Build.Source = root
	cfg.Build.Output = filepath.Join(root, "_site")

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	index := mustRead(t, filepath.Join(root, "_site", "index.html"))
	if !strings.Contains(index, "A custom introduction.") || !strings.Contains(index, "Entry") {
		t.Fatalf("custom index was not combined with the post listing: %s", index)
	}
}

func TestPublicationOutputsRequireSiteURL(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "post.md"), []byte("---\ndate: 2026-09-01\n---\n# Post"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Preset = "blog"
	cfg.Build.Source = root
	cfg.Build.Output = filepath.Join(root, "_site")
	if _, err := Build(cfg); err == nil || !strings.Contains(err.Error(), "site.url") {
		t.Fatalf("expected site.url validation error, got %v", err)
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
		"404.md":         "/404.html",
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

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
