package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpstr/rootmark/internal/config"
	"github.com/bpstr/rootmark/internal/theme"
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
	Template    string `yaml:"template"`
}

type templateData struct {
	Site        siteData
	Page        pageData
	Root        string
	ThemeAssets string
}

type siteData struct {
	Title       string
	Description string
	Language    string
	URL         string
	Author      string
	Favicon     string
	Logo        string
	Repository  string
	Home        string
	Navigation  []linkData
	Primary     ctaData
	Footer      footerData
}

type linkData struct {
	Label string
	URL   string
}

type ctaData struct {
	Label string
	URL   string
	Style string
}

type footerData struct {
	Text          string
	Links         []linkData
	GeneratedWith bool
	DevelopedBy   linkData
}

type pageData struct {
	Title       string
	Description string
	Content     template.HTML
	URL         string
	Canonical   string
	Source      string
}

var markdown = goldmark.New(
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

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

	activeTheme, err := theme.Load(cfg.Theme)
	if err != nil {
		return Result{}, err
	}
	defer activeTheme.Close()

	if err := os.RemoveAll(output); err != nil {
		return Result{}, fmt.Errorf("clean output: %w", err)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output: %w", err)
	}
	if err := activeTheme.CopyAssets(output); err != nil {
		return Result{}, fmt.Errorf("copy theme assets: %w", err)
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

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(entry.Name(), ".") || isRepositoryFile(rel) {
			return nil
		}

		if strings.EqualFold(filepath.Ext(path), ".md") {
			if err := renderFile(cfg, activeTheme, path, rel, output); err != nil {
				return err
			}
			pages++
			return nil
		}

		return copyStaticFile(path, rel, output)
	})
	if err != nil {
		return Result{}, fmt.Errorf("build site: %w", err)
	}

	if pages == 0 {
		return Result{}, fmt.Errorf("no Markdown content found in %s", cfg.Build.Source)
	}

	return Result{Pages: pages, Output: cfg.Build.Output}, nil
}

func renderFile(cfg config.Config, activeTheme *theme.Theme, sourcePath, relativePath, output string) error {
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

	destination := outputPath(output, relativePath)
	root, err := filepath.Rel(filepath.Dir(destination), output)
	if err != nil {
		return fmt.Errorf("resolve site root for %s: %w", relativePath, err)
	}
	root = filepath.ToSlash(root)
	route := routePath(relativePath)

	page := templateData{
		Site: siteData{
			Title:       cfg.Site.Title,
			Description: cfg.Site.Description,
			Language:    cfg.Site.Language,
			URL:         cfg.Site.URL,
			Author:      cfg.Site.Author,
			Favicon:     relativeSiteURL(root, cfg.Site.Favicon),
			Logo:        relativeSiteURL(root, cfg.Site.Logo),
			Repository:  cfg.Site.Repository,
			Home:        homeURL(root),
			Navigation:  resolveLinks(root, cfg.Navigation),
			Primary: ctaData{
				Label: cfg.Primary.Label,
				URL:   relativeSiteURL(root, cfg.Primary.URL),
				Style: cfg.Primary.Style,
			},
			Footer: footerData{
				Text:          cfg.Footer.Text,
				Links:         resolveLinks(root, cfg.Footer.Links),
				GeneratedWith: cfg.Footer.GeneratedWith,
				DevelopedBy: linkData{
					Label: cfg.Footer.DevelopedBy.Label,
					URL:   relativeSiteURL(root, cfg.Footer.DevelopedBy.URL),
				},
			},
		},
		Page: pageData{
			Title:       title,
			Description: meta.Description,
			Content:     template.HTML(rendered.String()), // goldmark omits raw HTML by default.
			URL:         route,
			Canonical:   canonicalURL(cfg.Site.URL, route),
			Source:      filepath.ToSlash(relativePath),
		},
		Root:        root,
		ThemeAssets: relativeSiteURL(root, "/_rootmark/"),
	}

	var document bytes.Buffer
	if err := activeTheme.Execute(&document, meta.Template, page); err != nil {
		return fmt.Errorf("render page template for %s: %w", relativePath, err)
	}

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

func routePath(relativePath string) string {
	rel := filepath.ToSlash(filepath.Clean(relativePath))
	withoutExt := strings.TrimSuffix(rel, filepath.Ext(rel))
	if strings.EqualFold(filepath.Base(withoutExt), "index") {
		dir := filepath.ToSlash(filepath.Dir(withoutExt))
		if dir == "." {
			return "/"
		}
		return "/" + strings.Trim(dir, "/") + "/"
	}
	return "/" + strings.Trim(withoutExt, "/") + "/"
}

func copyStaticFile(sourcePath, relativePath, output string) error {
	destination := filepath.Join(output, relativePath)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", relativePath, err)
	}
	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read asset %s: %w", relativePath, err)
	}
	if err := os.WriteFile(destination, contents, 0o644); err != nil {
		return fmt.Errorf("write asset %s: %w", relativePath, err)
	}
	return nil
}

func isRepositoryFile(relativePath string) bool {
	if filepath.Dir(relativePath) != "." {
		return false
	}
	switch strings.ToLower(filepath.Base(relativePath)) {
	case "readme.md", "license", "license.md", "license.txt", "changelog.md", "contributing.md":
		return true
	default:
		return false
	}
}

func resolveLinks(root string, links []config.Link) []linkData {
	resolved := make([]linkData, 0, len(links))
	for _, link := range links {
		if strings.TrimSpace(link.Label) == "" || strings.TrimSpace(link.URL) == "" {
			continue
		}
		resolved = append(resolved, linkData{Label: link.Label, URL: relativeSiteURL(root, link.URL)})
	}
	return resolved
}

func relativeSiteURL(root, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") {
		return target
	}
	if target == "/" {
		return homeURL(root)
	}
	parsed, err := url.Parse(target)
	if err == nil && parsed.IsAbs() {
		return target
	}

	target = strings.TrimLeft(target, "/")
	if root == "." || root == "" {
		return target
	}
	return strings.TrimRight(root, "/") + "/" + target
}

func homeURL(root string) string {
	if root == "." || root == "" {
		return "./"
	}
	return strings.TrimRight(root, "/") + "/"
}

func canonicalURL(siteURL, route string) string {
	if strings.TrimSpace(siteURL) == "" {
		return ""
	}
	return strings.TrimRight(siteURL, "/") + "/" + strings.TrimLeft(route, "/")
}

func sameOrInside(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
