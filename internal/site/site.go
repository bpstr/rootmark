package site

import (
	"bytes"
	"encoding/xml"
	"fmt"
	htmlpkg "html"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Template    string   `yaml:"template"`
	Date        string   `yaml:"date"`
	Updated     string   `yaml:"updated"`
	Draft       bool     `yaml:"draft"`
	Image       string   `yaml:"image"`
	Author      string   `yaml:"author"`
	Tags        []string `yaml:"tags"`
	Canonical   string   `yaml:"canonical"`
}

type contentPage struct {
	Meta         frontMatter
	RelativePath string
	Title        string
	Description  string
	Content      template.HTML
	Route        string
	Destination  string
	Date         time.Time
	Updated      time.Time
	HasDate      bool
	HasUpdated   bool
	IsPost       bool
	IsBlogIndex  bool
}

type templateData struct {
	Site        siteData
	Page        pageData
	Posts       []postData
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
	RSS         string
	Atom        string
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
	Title          string
	Description    string
	Content        template.HTML
	URL            string
	Canonical      string
	Source         string
	Date           string
	DateISO        string
	Updated        string
	UpdatedISO     string
	Image          string
	ImageCanonical string
	Author         string
	Tags           []string
	IsPost         bool
	IsBlogIndex    bool
	Previous       *postData
	Next           *postData
}

type postData struct {
	Title       string
	Description string
	URL         string
	Canonical   string
	Date        string
	DateISO     string
	Updated     string
	UpdatedISO  string
	Image       string
	Author      string
	Tags        []string
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
	if (cfg.FeedEnabled() || cfg.SitemapEnabled()) && strings.TrimSpace(cfg.Site.URL) == "" {
		return Result{}, fmt.Errorf("site.url is required when feed or sitemap generation is enabled")
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

	pages, err := discoverContent(cfg, source, output)
	if err != nil {
		return Result{}, fmt.Errorf("build site: %w", err)
	}
	if len(pages) == 0 && !cfg.IsBlog() {
		return Result{}, fmt.Errorf("no Markdown content found in %s", cfg.Build.Source)
	}

	posts := collectPosts(cfg, pages)
	if cfg.IsBlog() {
		pages = prepareBlogPages(cfg, pages, posts, output)
	}

	for _, page := range pages {
		if err := renderPage(cfg, activeTheme, page, posts, output); err != nil {
			return Result{}, fmt.Errorf("build site: %w", err)
		}
	}

	if cfg.FeedEnabled() {
		if err := writeFeeds(cfg, posts, output); err != nil {
			return Result{}, fmt.Errorf("generate feeds: %w", err)
		}
	}
	if cfg.SitemapEnabled() {
		if err := writeSitemap(cfg, pages, output); err != nil {
			return Result{}, fmt.Errorf("generate sitemap: %w", err)
		}
	}

	return Result{Pages: len(pages), Output: cfg.Build.Output}, nil
}

func discoverContent(cfg config.Config, source, output string) ([]*contentPage, error) {
	var pages []*contentPage
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
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
			page, err := loadContentPage(cfg, path, rel, output)
			if err != nil {
				return err
			}
			if page.Meta.Draft {
				return nil
			}
			pages = append(pages, page)
			return nil
		}

		return copyStaticFile(path, rel, output)
	})
	return pages, err
}

func loadContentPage(cfg config.Config, sourcePath, relativePath, output string) (*contentPage, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relativePath, err)
	}

	meta, body, err := parseFrontMatter(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", relativePath, err)
	}

	var rendered bytes.Buffer
	if err := markdown.Convert(body, &rendered); err != nil {
		return nil, fmt.Errorf("render %s: %w", relativePath, err)
	}

	title := meta.Title
	if title == "" {
		title = inferTitle(body, relativePath, cfg.Site.Title)
	}
	description := strings.TrimSpace(meta.Description)
	if description == "" {
		description = inferDescription(body)
	}

	date, hasDate, err := parseDate(meta.Date)
	if err != nil {
		return nil, fmt.Errorf("parse %s date: %w", relativePath, err)
	}
	updated, hasUpdated, err := parseDate(meta.Updated)
	if err != nil {
		return nil, fmt.Errorf("parse %s updated: %w", relativePath, err)
	}

	return &contentPage{
		Meta:         meta,
		RelativePath: filepath.ToSlash(relativePath),
		Title:        title,
		Description:  description,
		Content:      template.HTML(rendered.String()), // goldmark omits raw HTML by default.
		Route:        routePath(relativePath),
		Destination:  outputPath(output, relativePath),
		Date:         date,
		Updated:      updated,
		HasDate:      hasDate,
		HasUpdated:   hasUpdated,
	}, nil
}

func collectPosts(cfg config.Config, pages []*contentPage) []*contentPage {
	if !cfg.IsBlog() && !cfg.FeedEnabled() {
		return nil
	}

	posts := make([]*contentPage, 0)
	for _, page := range pages {
		if !page.HasDate || page.Route == "/" || page.Route == "/404.html" {
			continue
		}
		if cfg.IsBlog() {
			page.IsPost = true
		}
		posts = append(posts, page)
	}
	sort.SliceStable(posts, func(i, j int) bool {
		if posts[i].Date.Equal(posts[j].Date) {
			return posts[i].RelativePath < posts[j].RelativePath
		}
		return posts[i].Date.After(posts[j].Date)
	})
	return posts
}

func prepareBlogPages(cfg config.Config, pages, posts []*contentPage, output string) []*contentPage {
	var index *contentPage
	var notFound *contentPage
	for _, page := range pages {
		switch page.Route {
		case "/":
			index = page
		case "/404.html":
			notFound = page
		}
	}

	if index == nil {
		title := strings.TrimSpace(cfg.Site.Title)
		if title == "" {
			title = "Blog"
		}
		var intro strings.Builder
		intro.WriteString("<h1>")
		intro.WriteString(htmlpkg.EscapeString(title))
		intro.WriteString("</h1>\n")
		if strings.TrimSpace(cfg.Site.Description) != "" {
			intro.WriteString("<p>")
			intro.WriteString(htmlpkg.EscapeString(cfg.Site.Description))
			intro.WriteString("</p>\n")
		}
		index = &contentPage{
			Title:        title,
			Description:  cfg.Site.Description,
			Content:      template.HTML(intro.String()),
			Route:        "/",
			Destination:  filepath.Join(output, "index.html"),
			RelativePath: "",
		}
		pages = append(pages, index)
	}
	index.IsBlogIndex = true
	index.Content = template.HTML(string(index.Content) + blogListingHTML(posts))

	if notFound == nil {
		notFound = &contentPage{
			Title:       "Not found",
			Description: "The requested page could not be found.",
			Content: template.HTML(`<h1>404</h1>
<p>The page you are looking for does not exist.</p>
<p><a href="./">Return home</a></p>`),
			Route:        "/404.html",
			Destination:  filepath.Join(output, "404.html"),
			RelativePath: "",
		}
		pages = append(pages, notFound)
	}

	return pages
}

func blogListingHTML(posts []*contentPage) string {
	var out strings.Builder
	out.WriteString(`<section class="post-list" aria-label="Posts">`)
	out.WriteString("\n<h2>Posts</h2>\n")
	if len(posts) == 0 {
		out.WriteString("<p>No posts yet.</p>\n</section>\n")
		return out.String()
	}
	for _, post := range posts {
		out.WriteString(`<article class="post-entry">`)
		out.WriteString("\n<time class=\"post-date\" datetime=\"")
		out.WriteString(htmlpkg.EscapeString(post.Date.Format("2006-01-02")))
		out.WriteString("\">")
		out.WriteString(htmlpkg.EscapeString(post.Date.Format("January 2, 2006")))
		out.WriteString("</time>\n<h3><a href=\"")
		out.WriteString(htmlpkg.EscapeString(strings.TrimLeft(post.Route, "/")))
		out.WriteString("\">")
		out.WriteString(htmlpkg.EscapeString(post.Title))
		out.WriteString("</a></h3>\n")
		if post.Description != "" {
			out.WriteString("<p>")
			out.WriteString(htmlpkg.EscapeString(post.Description))
			out.WriteString("</p>\n")
		}
		out.WriteString("</article>\n")
	}
	out.WriteString("</section>\n")
	return out.String()
}

func renderPage(cfg config.Config, activeTheme *theme.Theme, page *contentPage, posts []*contentPage, output string) error {
	root, err := filepath.Rel(filepath.Dir(page.Destination), output)
	if err != nil {
		return fmt.Errorf("resolve site root for %s: %w", page.RelativePath, err)
	}
	root = filepath.ToSlash(root)

	pageView := pageData{
		Title:          page.Title,
		Description:    page.Description,
		Content:        page.Content,
		URL:            page.Route,
		Canonical:      canonicalForPage(cfg.Site.URL, page),
		Source:         page.RelativePath,
		Image:          relativeSiteURL(root, page.Meta.Image),
		ImageCanonical: absoluteSiteURL(cfg.Site.URL, page.Meta.Image),
		Author:         pageAuthor(cfg, page),
		Tags:           page.Meta.Tags,
		IsPost:         page.IsPost,
		IsBlogIndex:    page.IsBlogIndex,
	}
	if page.HasDate {
		pageView.Date = page.Date.Format("January 2, 2006")
		pageView.DateISO = page.Date.Format("2006-01-02")
	}
	if page.HasUpdated {
		pageView.Updated = page.Updated.Format("January 2, 2006")
		pageView.UpdatedISO = page.Updated.Format(time.RFC3339)
	}
	if page.IsPost {
		for i, post := range posts {
			if post != page {
				continue
			}
			if i+1 < len(posts) {
				previous := makePostData(cfg, root, posts[i+1])
				pageView.Previous = &previous
			}
			if i > 0 {
				next := makePostData(cfg, root, posts[i-1])
				pageView.Next = &next
			}
			break
		}
	}

	postViews := make([]postData, 0, len(posts))
	for _, post := range posts {
		postViews = append(postViews, makePostData(cfg, root, post))
	}

	data := templateData{
		Site:        makeSiteData(cfg, root),
		Page:        pageView,
		Posts:       postViews,
		Root:        root,
		ThemeAssets: relativeSiteURL(root, "/_rootmark/"),
	}

	var document bytes.Buffer
	if err := activeTheme.Execute(&document, page.Meta.Template, data); err != nil {
		return fmt.Errorf("render page template for %s: %w", displaySource(page), err)
	}

	if err := os.MkdirAll(filepath.Dir(page.Destination), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", displaySource(page), err)
	}
	if err := os.WriteFile(page.Destination, document.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", displaySource(page), err)
	}
	return nil
}

func makeSiteData(cfg config.Config, root string) siteData {
	data := siteData{
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
	}
	if cfg.FeedEnabled() {
		data.RSS = relativeSiteURL(root, "/rss.xml")
		data.Atom = relativeSiteURL(root, "/atom.xml")
	}
	return data
}

func makePostData(cfg config.Config, root string, page *contentPage) postData {
	data := postData{
		Title:       page.Title,
		Description: page.Description,
		URL:         relativeSiteURL(root, page.Route),
		Canonical:   canonicalForPage(cfg.Site.URL, page),
		Image:       relativeSiteURL(root, page.Meta.Image),
		Author:      pageAuthor(cfg, page),
		Tags:        page.Meta.Tags,
	}
	if page.HasDate {
		data.Date = page.Date.Format("January 2, 2006")
		data.DateISO = page.Date.Format("2006-01-02")
	}
	if page.HasUpdated {
		data.Updated = page.Updated.Format("January 2, 2006")
		data.UpdatedISO = page.Updated.Format(time.RFC3339)
	}
	return data
}

func pageAuthor(cfg config.Config, page *contentPage) string {
	if strings.TrimSpace(page.Meta.Author) != "" {
		return page.Meta.Author
	}
	return cfg.Site.Author
}

func displaySource(page *contentPage) string {
	if page.RelativePath != "" {
		return page.RelativePath
	}
	return page.Route
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

func parseDate(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("%q must be YYYY-MM-DD or RFC3339", value)
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

func inferDescription(body []byte) string {
	lines := strings.Split(string(body), "\n")
	var paragraph []string
	inFence := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">") {
			continue
		}
		if line == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, line)
	}
	text := strings.Join(paragraph, " ")
	replacer := strings.NewReplacer("**", "", "__", "", "*", "", "_", "", "`", "")
	text = strings.TrimSpace(replacer.Replace(text))
	runes := []rune(text)
	if len(runes) > 220 {
		text = strings.TrimSpace(string(runes[:217])) + "..."
	}
	return text
}

func outputPath(output, relativePath string) string {
	rel := filepath.Clean(relativePath)
	if strings.EqualFold(filepath.ToSlash(rel), "404.md") {
		return filepath.Join(output, "404.html")
	}
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
	if strings.EqualFold(rel, "404.md") {
		return "/404.html"
	}
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

var excludedContentFiles = map[string]struct{}{
	"readme.md":          {},
	"agents.md":          {},
	"claude.md":          {},
	"codex.md":           {},
	"gemini.md":          {},
	"skill.md":           {},
	"license":            {},
	"license.md":         {},
	"license.txt":        {},
	"changelog.md":       {},
	"contributing.md":    {},
	"security.md":        {},
	"support.md":         {},
	"code_of_conduct.md": {},
}

func isRepositoryFile(relativePath string) bool {
	_, excluded := excludedContentFiles[strings.ToLower(filepath.Base(relativePath))]
	return excluded
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

func canonicalForPage(siteURL string, page *contentPage) string {
	if strings.TrimSpace(page.Meta.Canonical) == "" {
		return canonicalURL(siteURL, page.Route)
	}
	return absoluteSiteURL(siteURL, page.Meta.Canonical)
}

func absoluteSiteURL(siteURL, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	parsed, err := url.Parse(target)
	if err == nil && parsed.IsAbs() {
		return target
	}
	if strings.TrimSpace(siteURL) == "" {
		return target
	}
	return canonicalURL(siteURL, target)
}

func sameOrInside(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language,omitempty"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        rssGUID  `xml:"guid"`
	Description string   `xml:"description,omitempty"`
	PubDate     string   `xml:"pubDate"`
	Author      string   `xml:"author,omitempty"`
	Categories  []string `xml:"category,omitempty"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

type atomDocument struct {
	XMLName xml.Name    `xml:"feed"`
	XMLNS   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Author  *atomAuthor `xml:"author,omitempty"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	ID         string         `xml:"id"`
	Link       atomLink       `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Summary    string         `xml:"summary,omitempty"`
	Author     *atomAuthor    `xml:"author,omitempty"`
	Categories []atomCategory `xml:"category,omitempty"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

func writeFeeds(cfg config.Config, posts []*contentPage, output string) error {
	home := canonicalURL(cfg.Site.URL, "/")
	updated := time.Unix(0, 0).UTC()
	rssItems := make([]rssItem, 0, len(posts))
	atomEntries := make([]atomEntry, 0, len(posts))

	for _, post := range posts {
		link := canonicalForPage(cfg.Site.URL, post)
		postUpdated := effectiveUpdated(post)
		if postUpdated.After(updated) {
			updated = postUpdated
		}
		author := pageAuthor(cfg, post)
		rssItems = append(rssItems, rssItem{
			Title:       post.Title,
			Link:        link,
			GUID:        rssGUID{Value: link, IsPermaLink: true},
			Description: post.Description,
			PubDate:     post.Date.Format(time.RFC1123Z),
			Author:      author,
			Categories:  post.Meta.Tags,
		})
		categories := make([]atomCategory, 0, len(post.Meta.Tags))
		for _, tag := range post.Meta.Tags {
			categories = append(categories, atomCategory{Term: tag})
		}
		var atomAuthorValue *atomAuthor
		if author != "" {
			atomAuthorValue = &atomAuthor{Name: author}
		}
		atomEntries = append(atomEntries, atomEntry{
			Title:      post.Title,
			ID:         link,
			Link:       atomLink{Href: link},
			Published:  post.Date.Format(time.RFC3339),
			Updated:    postUpdated.Format(time.RFC3339),
			Summary:    post.Description,
			Author:     atomAuthorValue,
			Categories: categories,
		})
	}

	rss := rssDocument{
		Version: "2.0",
		Channel: rssChannel{
			Title:         cfg.Site.Title,
			Link:          home,
			Description:   cfg.Site.Description,
			Language:      cfg.Site.Language,
			LastBuildDate: updated.Format(time.RFC1123Z),
			Items:         rssItems,
		},
	}
	if err := writeXML(filepath.Join(output, "rss.xml"), rss); err != nil {
		return err
	}

	var feedAuthor *atomAuthor
	if cfg.Site.Author != "" {
		feedAuthor = &atomAuthor{Name: cfg.Site.Author}
	}
	atom := atomDocument{
		XMLNS:   "http://www.w3.org/2005/Atom",
		Title:   cfg.Site.Title,
		ID:      home,
		Updated: updated.Format(time.RFC3339),
		Links: []atomLink{
			{Href: canonicalURL(cfg.Site.URL, "/atom.xml"), Rel: "self", Type: "application/atom+xml"},
			{Href: home, Rel: "alternate", Type: "text/html"},
		},
		Author:  feedAuthor,
		Entries: atomEntries,
	}
	return writeXML(filepath.Join(output, "atom.xml"), atom)
}

func effectiveUpdated(page *contentPage) time.Time {
	if page.HasUpdated {
		return page.Updated
	}
	if page.HasDate {
		return page.Date
	}
	return time.Unix(0, 0).UTC()
}

type sitemapDocument struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
	LastMod  string `xml:"lastmod,omitempty"`
}

func writeSitemap(cfg config.Config, pages []*contentPage, output string) error {
	entries := make([]sitemapURL, 0, len(pages))
	for _, page := range pages {
		if page.Route == "/404.html" {
			continue
		}
		entry := sitemapURL{Location: canonicalURL(cfg.Site.URL, page.Route)}
		if page.HasUpdated {
			entry.LastMod = page.Updated.Format("2006-01-02")
		} else if page.HasDate {
			entry.LastMod = page.Date.Format("2006-01-02")
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Location < entries[j].Location })
	return writeXML(filepath.Join(output, "sitemap.xml"), sitemapDocument{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  entries,
	})
}

func writeXML(path string, value any) error {
	data, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
