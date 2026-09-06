package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpstr/rootmark/internal/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v3"
)

type Result struct {
	Pages  int
	Output string
}

type frontMatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type pageData struct {
	SiteTitle   string
	Title       string
	Description string
	Language    string
	HTML        template.HTML
}

var markdown = goldmark.New(
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="{{.Language}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{if .Title}}{{.Title}}{{if .SiteTitle}} · {{.SiteTitle}}{{end}}{{else}}{{.SiteTitle}}{{end}}</title>
  {{- if .Description}}
  <meta name="description" content="{{.Description}}">
  {{- end}}
  <style>
    :root { color-scheme: light dark; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; line-height: 1.65; }
    main { width: min(100% - 2rem, 48rem); margin: 4rem auto; }
    h1, h2, h3 { line-height: 1.2; }
    img { max-width: 100%; height: auto; }
    pre { overflow-x: auto; padding: 1rem; border-radius: .5rem; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    a { color: inherit; text-underline-offset: .18em; }
  </style>
</head>
<body>
  <main>{{.HTML}}</main>
</body>
</html>
`))

func Build(cfg config.Config) (Result, error) {
	source, err := filepath.Abs(cfg.Build.Source)
	if err != nil {
		return Result{}, fmt.Errorf("resolve source: %w", err)
	}
	output, err := filepath.Abs(cfg.Build.Output)
	if err != nil {
		return Result{}, fmt.Errorf("resolve output: %w", err)
	}

	info, err := os.Stat(source)
	if err != nil {
		return Result{}, fmt.Errorf("source directory: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("source %q is not a directory", cfg.Build.Source)
	}
	if source == output {
		return Result{}, fmt.Errorf("source and output directories must differ")
	}

	if err := os.RemoveAll(output); err != nil {
		return Result{}, fmt.Errorf("clean output: %w", err)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output: %w", err)
	}

	pages := 0
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == source {
				return nil
			}
			if sameOrInside(path, output) || strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.EqualFold(filepath.Ext(path), ".md") || strings.EqualFold(entry.Name(), "README.md") {
			return nil
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if err := renderFile(cfg, path, rel, output); err != nil {
			return err
		}
		pages++
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("build site: %w", err)
	}

	if pages == 0 {
		return Result{}, fmt.Errorf("no Markdown content found in %s", cfg.Build.Source)
	}

	return Result{Pages: pages, Output: cfg.Build.Output}, nil
}

func renderFile(cfg config.Config, sourcePath, relativePath, output string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", relativePath, err)
	}

	meta, body, err := parseFrontMatter(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", relativePath, err)
	}

	var rendered bytes.Buffer
	if err := markdown.Convert(body, &rendered); err != nil {
		return fmt.Errorf("render %s: %w", relativePath, err)
	}

	title := meta.Title
	if title == "" {
		title = inferTitle(body, relativePath, cfg.Site.Title)
	}

	page := pageData{
		SiteTitle:   cfg.Site.Title,
		Title:       title,
		Description: meta.Description,
		Language:    cfg.Site.Language,
		HTML:        template.HTML(rendered.String()), // goldmark omits raw HTML by default.
	}

	var document bytes.Buffer
	if err := pageTemplate.Execute(&document, page); err != nil {
		return fmt.Errorf("render page template for %s: %w", relativePath, err)
	}

	destination := outputPath(output, relativePath)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", relativePath, err)
	}
	if err := os.WriteFile(destination, document.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relativePath, err)
	}
	return nil
}

func parseFrontMatter(data []byte) (frontMatter, []byte, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return frontMatter{}, []byte(text), nil
	}

	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return frontMatter{}, nil, fmt.Errorf("frontmatter is missing closing ---")
	}

	var meta frontMatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
		return frontMatter{}, nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	return meta, []byte(rest[end+5:]), nil
}

func inferTitle(body []byte, relativePath, siteTitle string) string {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	base := strings.TrimSuffix(filepath.Base(relativePath), filepath.Ext(relativePath))
	if strings.EqualFold(base, "index") {
		if siteTitle != "" {
			return siteTitle
		}
		return "Home"
	}

	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	words := strings.Fields(base)
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func outputPath(output, relativePath string) string {
	rel := filepath.Clean(relativePath)
	ext := filepath.Ext(rel)
	withoutExt := strings.TrimSuffix(rel, ext)

	if strings.EqualFold(filepath.Base(withoutExt), "index") {
		dir := filepath.Dir(withoutExt)
		if dir == "." {
			return filepath.Join(output, "index.html")
		}
		return filepath.Join(output, dir, "index.html")
	}

	return filepath.Join(output, withoutExt, "index.html")
}

func sameOrInside(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
