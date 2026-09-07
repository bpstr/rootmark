package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpstr/rootmark/internal/config"
)

func TestDocsPresetAddsNavigationTOCAndEditLinks(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "_site")
	if err := os.MkdirAll(filepath.Join(root, "guide"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"index.md":         "# Root docs\n\nWelcome.\n",
		"guide/index.md":   "# Guide\n\nGuide overview.\n",
		"guide/install.md": "# Install\n\n## Linux\n\nInstall on Linux.\n\n### Package\n\nUse the package.\n",
		"reference.md":     "# Reference\n\nAPI reference.\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Default()
	cfg.Preset = "docs"
	cfg.Site.Title = "Root docs"
	cfg.Site.Repository = "https://github.com/example/project"
	cfg.Build.Source = root
	cfg.Build.Output = output

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}

	install, err := os.ReadFile(filepath.Join(output, "guide", "install", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(install)
	for _, expected := range []string{
		`class="rootmark-docs-layout"`,
		`class="rootmark-docs-link active"`,
		`href="#linux"`,
		`href="#package"`,
		`Edit this page`,
		`https://github.com/example/project/edit/main/guide/install.md`,
		`<small>Previous</small>Guide`,
		`<small>Next</small>Reference`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("docs page missing %q: %s", expected, html)
		}
	}
}

func TestPortfolioPresetBuildsProjectGridAndProjectMetadata(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "_site")
	files := map[string]string{
		"index.md": "# Work\n\nSelected projects.\n",
		"alpha.md": `---
title: Alpha
description: First project.
date: 2026-05-01
image: alpha.svg
tags: [go, cli]
---

# Alpha

Project Alpha.
`,
		"beta.md": `---
title: Beta
description: Second project.
date: 2026-06-01
tags: [web]
---

# Beta

Project Beta.
`,
		"about.md": `---
title: About
project: false
---

# About

About me.
`,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "alpha.svg"), []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Preset = "portfolio"
	cfg.Site.Title = "Work"
	cfg.Build.Source = root
	cfg.Build.Output = output

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}

	index, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(index)
	betaAt := strings.Index(html, ">Beta<")
	alphaAt := strings.Index(html, ">Alpha<")
	if betaAt == -1 || alphaAt == -1 || betaAt >= alphaAt {
		t.Fatalf("portfolio projects are not newest-first: %s", html)
	}
	for _, expected := range []string{
		`class="rootmark-portfolio-grid"`,
		`First project.`,
		`alpha.svg`,
		`>go<`,
		`>cli<`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("portfolio index missing %q: %s", expected, html)
		}
	}
	if strings.Contains(html, `>About<`) {
		t.Fatalf("project:false page appeared in portfolio grid: %s", html)
	}

	alpha, err := os.ReadFile(filepath.Join(output, "alpha", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	alphaHTML := string(alpha)
	if !strings.Contains(alphaHTML, `class="rootmark-project-hero"`) || !strings.Contains(alphaHTML, `>2026<`) || !strings.Contains(alphaHTML, `>go<`) {
		t.Fatalf("portfolio project metadata missing: %s", alphaHTML)
	}
}

func TestPortfolioPresetRequiresIndex(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.md"), []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Preset = "portfolio"
	cfg.Build.Source = root
	cfg.Build.Output = filepath.Join(root, "_site")
	if _, err := Build(cfg); err == nil || !strings.Contains(err.Error(), "requires index.md") {
		t.Fatalf("expected portfolio index validation error, got %v", err)
	}
}
