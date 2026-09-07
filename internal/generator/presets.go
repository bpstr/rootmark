package generator

import (
	"fmt"
	htmlpkg "html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bpstr/rootmark/internal/config"
	"gopkg.in/yaml.v3"
)

type generatedPage struct {
	OutputPath string
	Route      string
	Title      string
	Source     string
	Depth      int
}

type portfolioFrontMatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Date        string   `yaml:"date"`
	Image       string   `yaml:"image"`
	Tags        []string `yaml:"tags"`
	Project     *bool    `yaml:"project"`
}

type portfolioProject struct {
	Page        generatedPage
	Description string
	Date        time.Time
	HasDate     bool
	Image       string
	Tags        []string
}

type tocHeading struct {
	Level int
	ID    string
	Title string
}

var (
	mainOpenPattern     = regexp.MustCompile(`(?is)<main\b[^>]*>`)
	h1Pattern           = regexp.MustCompile(`(?is)<h1\b[^>]*>(.*?)</h1>`)
	headingPattern      = regexp.MustCompile(`(?is)<h([2-4])\b[^>]*\bid=["']([^"']+)["'][^>]*>(.*?)</h[2-4]>`)
	tagPattern          = regexp.MustCompile(`(?is)<[^>]+>`)
	ogTitlePattern      = regexp.MustCompile(`(?is)<meta\s+property=["']og:title["']\s+content=["']([^"']*)["'][^>]*>`)
	descriptionPattern  = regexp.MustCompile(`(?is)<meta\s+name=["']description["']\s+content=["']([^"']*)["'][^>]*>`)
)

func enhancePreset(cfg config.Config, output string) error {
	switch {
	case cfg.IsDocs():
		return enhanceDocs(cfg, output)
	case cfg.IsPortfolio():
		return enhancePortfolio(cfg, output)
	default:
		return nil
	}
}

func enhanceDocs(cfg config.Config, output string) error {
	pages, err := discoverGeneratedPages(cfg, output)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("docs preset requires at least one Markdown page")
	}

	for i, page := range pages {
		contents, err := os.ReadFile(page.OutputPath)
		if err != nil {
			return fmt.Errorf("read generated docs page %s: %w", page.Route, err)
		}
		document := string(contents)
		open := mainOpenPattern.FindStringIndex(document)
		close := strings.LastIndex(strings.ToLower(document), "</main>")
		if open == nil || close < open[1] {
			return fmt.Errorf("docs preset requires the active theme to contain a main element")
		}

		body := document[open[1]:close]
		root := filepath.Dir(page.OutputPath)
		nav := docsNavigationHTML(root, pages, i)
		toc := docsTOCHTML(extractTOC(body))
		pager := docsPagerHTML(root, pages, i)
		edit := docsEditHTML(cfg, page.Source)

		wrapped := `<div class="rootmark-docs-layout">` + nav + `<article class="rootmark-docs-content">` + body + pager + edit + `</article>` + toc + `</div>`
		document = document[:open[1]] + wrapped + document[close:]
		document = injectBeforeClosingTag(document, "head", docsPresetStyle)
		if err := os.WriteFile(page.OutputPath, []byte(document), 0o644); err != nil {
			return fmt.Errorf("write enhanced docs page %s: %w", page.Route, err)
		}
	}
	return nil
}

func enhancePortfolio(cfg config.Config, output string) error {
	pages, err := discoverGeneratedPages(cfg, output)
	if err != nil {
		return err
	}
	if len(pages) == 0 || pages[0].Route != "/" {
		return fmt.Errorf("portfolio preset requires index.md as the landing page")
	}

	projects, err := discoverPortfolioProjects(cfg, output, pages)
	if err != nil {
		return err
	}

	indexPath := pages[0].OutputPath
	contents, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read portfolio index: %w", err)
	}
	indexDocument := string(contents)
	cards := portfolioCardsHTML(output, indexPath, projects)
	indexDocument, err = insertBeforeMainClose(indexDocument, cards)
	if err != nil {
		return err
	}
	indexDocument = injectBeforeClosingTag(indexDocument, "head", portfolioPresetStyle)
	if err := os.WriteFile(indexPath, []byte(indexDocument), 0o644); err != nil {
		return fmt.Errorf("write portfolio index: %w", err)
	}

	for _, project := range projects {
		contents, err := os.ReadFile(project.Page.OutputPath)
		if err != nil {
			return fmt.Errorf("read portfolio project %s: %w", project.Page.Route, err)
		}
		document := string(contents)
		meta := portfolioProjectMetaHTML(output, project)
		document, err = insertAfterMainOpen(document, meta)
		if err != nil {
			return err
		}
		document = injectBeforeClosingTag(document, "head", portfolioPresetStyle)
		if err := os.WriteFile(project.Page.OutputPath, []byte(document), 0o644); err != nil {
			return fmt.Errorf("write portfolio project %s: %w", project.Page.Route, err)
		}
	}
	return nil
}

func discoverGeneratedPages(cfg config.Config, output string) ([]generatedPage, error) {
	var pages []generatedPage
	err := filepath.WalkDir(output, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "_rootmark" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(entry.Name(), "index.html") {
			return nil
		}

		route, err := routeFromOutput(output, path)
		if err != nil {
			return err
		}
		source := sourceForRoute(cfg, route)
		if source == "" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		title := generatedHTMLTitle(string(contents))
		if title == "" {
			title = sourceTitle(source)
		}
		pages = append(pages, generatedPage{
			OutputPath: path,
			Route:      route,
			Title:      title,
			Source:     source,
			Depth:      routeDepth(route),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].Route == "/" {
			return true
		}
		if pages[j].Route == "/" {
			return false
		}
		return pages[i].Route < pages[j].Route
	})
	return pages, nil
}

func discoverPortfolioProjects(cfg config.Config, output string, pages []generatedPage) ([]portfolioProject, error) {
	projects := make([]portfolioProject, 0, len(pages))
	for _, page := range pages {
		if page.Route == "/" {
			continue
		}
		meta, err := readPortfolioFrontMatter(cfg, page.Source)
		if err != nil {
			return nil, err
		}
		if meta.Project != nil && !*meta.Project {
			continue
		}

		contents, err := os.ReadFile(page.OutputPath)
		if err != nil {
			return nil, err
		}
		description := strings.TrimSpace(meta.Description)
		if description == "" {
			description = generatedHTMLDescription(string(contents))
		}
		if meta.Title != "" {
			page.Title = meta.Title
		}

		project := portfolioProject{
			Page:        page,
			Description: description,
			Image:       strings.TrimSpace(meta.Image),
			Tags:        meta.Tags,
		}
		if strings.TrimSpace(meta.Date) != "" {
			parsed, err := parsePresetDate(meta.Date)
			if err != nil {
				return nil, fmt.Errorf("parse portfolio date in %s: %w", page.Source, err)
			}
			project.Date = parsed
			project.HasDate = true
		}
		projects = append(projects, project)
	}

	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].HasDate != projects[j].HasDate {
			return projects[i].HasDate
		}
		if projects[i].HasDate && !projects[i].Date.Equal(projects[j].Date) {
			return projects[i].Date.After(projects[j].Date)
		}
		return strings.ToLower(projects[i].Page.Title) < strings.ToLower(projects[j].Page.Title)
	})
	return projects, nil
}

func readPortfolioFrontMatter(cfg config.Config, source string) (portfolioFrontMatter, error) {
	path := filepath.Join(cfg.Build.Source, filepath.FromSlash(source))
	data, err := os.ReadFile(path)
	if err != nil {
		return portfolioFrontMatter{}, fmt.Errorf("read portfolio source %s: %w", source, err)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return portfolioFrontMatter{}, nil
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return portfolioFrontMatter{}, nil
	}
	var meta portfolioFrontMatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
		return portfolioFrontMatter{}, fmt.Errorf("parse portfolio frontmatter in %s: %w", source, err)
	}
	return meta, nil
}

func docsNavigationHTML(fromDir string, pages []generatedPage, active int) string {
	var out strings.Builder
	out.WriteString(`<nav class="rootmark-docs-nav" aria-label="Documentation"><strong>Documentation</strong>`)
	for i, page := range pages {
		href := relativePageHref(fromDir, page.OutputPath)
		class := "rootmark-docs-link"
		if i == active {
			class += " active"
		}
		out.WriteString(`<a class="` + class + `" style="--depth:` + strconv.Itoa(page.Depth) + `" href="` + htmlpkg.EscapeString(href) + `">` + htmlpkg.EscapeString(page.Title) + `</a>`)
	}
	out.WriteString(`</nav>`)
	return out.String()
}

func docsTOCHTML(headings []tocHeading) string {
	if len(headings) == 0 {
		return `<aside class="rootmark-docs-toc" aria-label="On this page"></aside>`
	}
	var out strings.Builder
	out.WriteString(`<aside class="rootmark-docs-toc" aria-label="On this page"><strong>On this page</strong>`)
	for _, heading := range headings {
		out.WriteString(`<a style="--depth:` + strconv.Itoa(heading.Level-2) + `" href="#` + htmlpkg.EscapeString(heading.ID) + `">` + htmlpkg.EscapeString(heading.Title) + `</a>`)
	}
	out.WriteString(`</aside>`)
	return out.String()
}

func docsPagerHTML(fromDir string, pages []generatedPage, index int) string {
	if len(pages) < 2 {
		return ""
	}
	var out strings.Builder
	out.WriteString(`<nav class="rootmark-docs-pager" aria-label="Documentation navigation">`)
	if index > 0 {
		previous := pages[index-1]
		out.WriteString(`<a href="` + htmlpkg.EscapeString(relativePageHref(fromDir, previous.OutputPath)) + `"><small>Previous</small>` + htmlpkg.EscapeString(previous.Title) + `</a>`)
	} else {
		out.WriteString(`<span></span>`)
	}
	if index+1 < len(pages) {
		next := pages[index+1]
		out.WriteString(`<a href="` + htmlpkg.EscapeString(relativePageHref(fromDir, next.OutputPath)) + `"><small>Next</small>` + htmlpkg.EscapeString(next.Title) + `</a>`)
	}
	out.WriteString(`</nav>`)
	return out.String()
}

func docsEditHTML(cfg config.Config, source string) string {
	edit := githubEditURL(cfg.Site.Repository, repositorySourcePath(cfg, source))
	if edit == "" {
		return ""
	}
	return `<p class="rootmark-docs-edit"><a href="` + htmlpkg.EscapeString(edit) + `">Edit this page</a></p>`
}

func extractTOC(contents string) []tocHeading {
	matches := headingPattern.FindAllStringSubmatch(contents, -1)
	headings := make([]tocHeading, 0, len(matches))
	for _, match := range matches {
		level, _ := strconv.Atoi(match[1])
		title := cleanHTMLText(match[3])
		if title == "" {
			continue
		}
		headings = append(headings, tocHeading{Level: level, ID: match[2], Title: title})
	}
	return headings
}

func portfolioCardsHTML(output, indexPath string, projects []portfolioProject) string {
	var out strings.Builder
	out.WriteString(`<section class="rootmark-portfolio" aria-label="Projects"><div class="rootmark-portfolio-heading"><h2>Selected work</h2><span>` + strconv.Itoa(len(projects)) + ` projects</span></div><div class="rootmark-portfolio-grid">`)
	for _, project := range projects {
		href := relativePageHref(filepath.Dir(indexPath), project.Page.OutputPath)
		out.WriteString(`<article class="rootmark-portfolio-card"><a class="rootmark-portfolio-card-link" href="` + htmlpkg.EscapeString(href) + `">`)
		if project.Image != "" {
			out.WriteString(`<img src="` + htmlpkg.EscapeString(relativeAssetHref(output, indexPath, project.Image)) + `" alt="">`)
		}
		out.WriteString(`<div class="rootmark-portfolio-card-body">`)
		if project.HasDate {
			out.WriteString(`<time datetime="` + project.Date.Format("2006-01-02") + `">` + project.Date.Format("2006") + `</time>`)
		}
		out.WriteString(`<h3>` + htmlpkg.EscapeString(project.Page.Title) + `</h3>`)
		if project.Description != "" {
			out.WriteString(`<p>` + htmlpkg.EscapeString(project.Description) + `</p>`)
		}
		if len(project.Tags) > 0 {
			out.WriteString(`<div class="rootmark-portfolio-tags">`)
			for _, tag := range project.Tags {
				out.WriteString(`<span>` + htmlpkg.EscapeString(tag) + `</span>`)
			}
			out.WriteString(`</div>`)
		}
		out.WriteString(`</div></a></article>`)
	}
	out.WriteString(`</div></section>`)
	return out.String()
}

func portfolioProjectMetaHTML(output string, project portfolioProject) string {
	var out strings.Builder
	if project.Image != "" {
		out.WriteString(`<img class="rootmark-project-hero" src="` + htmlpkg.EscapeString(relativeAssetHref(output, project.Page.OutputPath, project.Image)) + `" alt="">`)
	}
	if project.HasDate || len(project.Tags) > 0 {
		out.WriteString(`<div class="rootmark-project-meta">`)
		if project.HasDate {
			out.WriteString(`<time datetime="` + project.Date.Format("2006-01-02") + `">` + project.Date.Format("2006") + `</time>`)
		}
		for _, tag := range project.Tags {
			out.WriteString(`<span>` + htmlpkg.EscapeString(tag) + `</span>`)
		}
		out.WriteString(`</div>`)
	}
	return out.String()
}

func routeFromOutput(output, path string) (string, error) {
	rel, err := filepath.Rel(output, path)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "index.html" {
		return "/", nil
	}
	return "/" + strings.TrimSuffix(rel, "index.html"), nil
}

func sourceForRoute(cfg config.Config, route string) string {
	root, err := filepath.Abs(cfg.Build.Source)
	if err != nil {
		return ""
	}
	var candidates []string
	if route == "/" {
		candidates = []string{"index.md"}
	} else {
		path := strings.Trim(route, "/")
		candidates = []string{path + ".md", filepath.ToSlash(filepath.Join(path, "index.md"))}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate))); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func sourceTitle(source string) string {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	if strings.EqualFold(base, "index") {
		base = filepath.Base(filepath.Dir(source))
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

func generatedHTMLTitle(document string) string {
	if match := ogTitlePattern.FindStringSubmatch(document); len(match) > 1 {
		return htmlpkg.UnescapeString(strings.TrimSpace(match[1]))
	}
	if match := h1Pattern.FindStringSubmatch(document); len(match) > 1 {
		return cleanHTMLText(match[1])
	}
	return ""
}

func generatedHTMLDescription(document string) string {
	if match := descriptionPattern.FindStringSubmatch(document); len(match) > 1 {
		return htmlpkg.UnescapeString(strings.TrimSpace(match[1]))
	}
	return ""
}

func cleanHTMLText(value string) string {
	value = tagPattern.ReplaceAllString(value, "")
	return strings.TrimSpace(htmlpkg.UnescapeString(value))
}

func routeDepth(route string) int {
	trimmed := strings.Trim(route, "/")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "/")
}

func relativePageHref(fromDir, target string) string {
	targetDir := filepath.Dir(target)
	rel, err := filepath.Rel(fromDir, targetDir)
	if err != nil {
		return "./"
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "./"
	}
	return strings.TrimRight(rel, "/") + "/"
}

func relativeAssetHref(output, fromFile, asset string) string {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return ""
	}
	parsed, err := url.Parse(asset)
	if err == nil && (parsed.IsAbs() || strings.HasPrefix(asset, "//")) {
		return asset
	}
	target := filepath.Join(output, filepath.FromSlash(strings.TrimLeft(asset, "/")))
	rel, err := filepath.Rel(filepath.Dir(fromFile), target)
	if err != nil {
		return asset
	}
	return filepath.ToSlash(rel)
}

func repositorySourcePath(cfg config.Config, source string) string {
	buildSource := filepath.Clean(cfg.Build.Source)
	if filepath.IsAbs(buildSource) || buildSource == "." {
		return filepath.ToSlash(source)
	}
	return filepath.ToSlash(filepath.Join(buildSource, filepath.FromSlash(source)))
}

func githubEditURL(repository, source string) string {
	repository = strings.TrimSpace(strings.TrimSuffix(repository, "/"))
	if repository == "" || source == "" {
		return ""
	}
	parsed, err := url.Parse(repository)
	if err != nil || !strings.EqualFold(parsed.Host, "github.com") {
		return ""
	}
	repository = strings.TrimSuffix(repository, ".git")
	return repository + "/edit/main/" + escapePath(source)
}

func escapePath(value string) string {
	parts := strings.Split(filepath.ToSlash(value), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func parsePresetDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("%q must be YYYY-MM-DD or RFC3339", value)
}

func insertBeforeMainClose(document, addition string) (string, error) {
	close := strings.LastIndex(strings.ToLower(document), "</main>")
	if close == -1 {
		return "", fmt.Errorf("preset requires the active theme to contain a main element")
	}
	return document[:close] + addition + document[close:], nil
}

func insertAfterMainOpen(document, addition string) (string, error) {
	open := mainOpenPattern.FindStringIndex(document)
	if open == nil {
		return "", fmt.Errorf("preset requires the active theme to contain a main element")
	}
	return document[:open[1]] + addition + document[open[1]:], nil
}

func injectBeforeClosingTag(document, tag, addition string) string {
	needle := "</" + strings.ToLower(tag) + ">"
	index := strings.LastIndex(strings.ToLower(document), needle)
	if index == -1 {
		return addition + document
	}
	return document[:index] + addition + document[index:]
}

const docsPresetStyle = `<style id="rootmark-docs-preset">
.rootmark-docs-layout{width:min(78rem,calc(100vw - 2rem));margin-left:50%;transform:translateX(-50%);display:grid;grid-template-columns:14rem minmax(0,1fr) 12rem;gap:2rem;align-items:start}.rootmark-docs-content{min-width:0}.rootmark-docs-nav,.rootmark-docs-toc{position:sticky;top:1rem;display:flex;flex-direction:column;gap:.2rem;font:500 .76rem/1.45 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.rootmark-docs-nav strong,.rootmark-docs-toc strong{margin-bottom:.55rem;color:var(--text,#181a1b);font-size:.72rem;text-transform:uppercase;letter-spacing:.08em}.rootmark-docs-link,.rootmark-docs-toc a{display:block;padding:.36rem .48rem .36rem calc(.48rem + var(--depth,0)*.72rem);border-radius:.35rem;color:var(--muted,#686d72);text-decoration:none}.rootmark-docs-link:hover,.rootmark-docs-toc a:hover,.rootmark-docs-link.active{color:var(--text,#181a1b);background:var(--soft,#efefeb)}.rootmark-docs-link.active{font-weight:700}.rootmark-docs-pager{display:grid;grid-template-columns:1fr 1fr;gap:1rem;margin-top:3rem;padding-top:1.25rem;border-top:1px solid var(--line,#ddd)}.rootmark-docs-pager a{display:block;padding:.75rem;border:1px solid var(--line,#ddd);border-radius:.5rem;text-decoration:none}.rootmark-docs-pager a:last-child{text-align:right}.rootmark-docs-pager small{display:block;color:var(--muted,#686d72);font:500 .7rem/1.4 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.rootmark-docs-edit{margin-top:1.2rem;color:var(--muted,#686d72);font:500 .72rem/1.4 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}@media(max-width:980px){.rootmark-docs-layout{grid-template-columns:12rem minmax(0,1fr)}.rootmark-docs-toc{display:none}}@media(max-width:720px){.rootmark-docs-layout{width:100%;margin-left:0;transform:none;display:block}.rootmark-docs-nav{position:static;margin-bottom:2rem;padding-bottom:1rem;border-bottom:1px solid var(--line,#ddd)}.rootmark-docs-pager{grid-template-columns:1fr}.rootmark-docs-pager a:last-child{text-align:left}}
</style>`

const portfolioPresetStyle = `<style id="rootmark-portfolio-preset">
.rootmark-portfolio{margin-top:3rem}.rootmark-portfolio-heading{display:flex;align-items:baseline;justify-content:space-between;gap:1rem;margin-bottom:1rem}.rootmark-portfolio-heading h2{margin:0}.rootmark-portfolio-heading span,.rootmark-project-meta{color:var(--muted,#686d72);font:500 .72rem/1.4 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.rootmark-portfolio-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.rootmark-portfolio-card{overflow:hidden;border:1px solid var(--line,#ddd);border-radius:.65rem;background:var(--panel,transparent)}.rootmark-portfolio-card-link{display:block;height:100%;text-decoration:none}.rootmark-portfolio-card img{display:block;width:100%;aspect-ratio:16/9;object-fit:cover;border-radius:0;border-bottom:1px solid var(--line,#ddd)}.rootmark-portfolio-card-body{padding:1rem}.rootmark-portfolio-card time{color:var(--muted,#686d72);font:500 .7rem/1.4 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.rootmark-portfolio-card h3{margin:.35rem 0 .45rem;font-size:1.2rem}.rootmark-portfolio-card p{margin:.35rem 0;color:var(--muted,#686d72)}.rootmark-portfolio-tags,.rootmark-project-meta{display:flex;gap:.4rem;flex-wrap:wrap}.rootmark-portfolio-tags{margin-top:.75rem}.rootmark-portfolio-tags span,.rootmark-project-meta span{padding:.16rem .38rem;border:1px solid var(--line,#ddd);border-radius:999px}.rootmark-project-hero{display:block;width:100%;margin:0 0 1.5rem;aspect-ratio:16/9;object-fit:cover}.rootmark-project-meta{margin-bottom:1.25rem}@media(max-width:680px){.rootmark-portfolio-grid{grid-template-columns:1fr}}
</style>`
