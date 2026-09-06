package theme

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const formatVersion = 1

type Manifest struct {
	Name   string `yaml:"name"`
	Format int    `yaml:"format"`
}

type Theme struct {
	manifest  Manifest
	templates *template.Template
	assetsDir string
	tempDir   string
}

func Default() *Theme {
	t := template.Must(template.New("default.html").Parse(defaultTemplate))
	return &Theme{
		manifest:  Manifest{Name: "Rootmark", Format: formatVersion},
		templates: t,
	}
}

func Load(source string) (*Theme, error) {
	if strings.TrimSpace(source) == "" {
		return Default(), nil
	}

	repoURL, err := url.Parse(source)
	if err != nil || repoURL.Scheme != "https" || repoURL.Host == "" || repoURL.User != nil {
		return nil, fmt.Errorf("theme must be a public https repository URL")
	}
	if repoURL.Fragment != "" || repoURL.RawQuery != "" {
		return nil, fmt.Errorf("theme URL query strings and fragments are not supported")
	}

	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("fetch theme: git is required to load remote themes")
	}

	tempDir, err := os.MkdirTemp("", "rootmark-theme-*")
	if err != nil {
		return nil, fmt.Errorf("create theme directory: %w", err)
	}

	cmd := exec.Command("git", "-c", "credential.helper=", "clone", "--depth", "1", "--single-branch", source, tempDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("fetch theme: %s", message)
	}

	t, err := loadDirectory(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, err
	}
	t.tempDir = tempDir
	return t, nil
}

func loadDirectory(dir string) (*Theme, error) {
	manifestPath := filepath.Join(dir, "theme.yml")
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("theme manifest: %w", err)
	}
	if manifestInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("theme manifest cannot be a symlink")
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("theme manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse theme manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return nil, fmt.Errorf("theme manifest requires name")
	}
	if manifest.Format != formatVersion {
		return nil, fmt.Errorf("unsupported theme format %d, expected %d", manifest.Format, formatVersion)
	}

	templatesDir := filepath.Join(dir, "templates")
	parsed := template.New("root")
	foundTemplate := false
	err = filepath.WalkDir(templatesDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("theme templates cannot contain symlinks: %s", entry.Name())
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}

		rel, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if _, err := parsed.New(name).Parse(string(contents)); err != nil {
			return fmt.Errorf("parse template %s: %w", name, err)
		}
		foundTemplate = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load theme templates: %w", err)
	}
	if !foundTemplate || parsed.Lookup("default.html") == nil {
		return nil, fmt.Errorf("theme requires templates/default.html")
	}

	assetsDir := filepath.Join(dir, "assets")
	if info, err := os.Lstat(assetsDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("theme assets: %w", err)
		}
		assetsDir = ""
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("theme assets must be a directory")
	}

	return &Theme{
		manifest:  manifest,
		templates: parsed,
		assetsDir: assetsDir,
	}, nil
}

func (t *Theme) Name() string {
	return t.manifest.Name
}

func (t *Theme) Execute(w io.Writer, name string, data any) error {
	if strings.TrimSpace(name) == "" {
		name = "default.html"
	}
	name = filepath.ToSlash(filepath.Clean(name))
	if name == "." || name == ".." || strings.HasPrefix(name, "../") || filepath.IsAbs(name) {
		return fmt.Errorf("invalid template %q", name)
	}
	if filepath.Ext(name) == "" {
		name += ".html"
	}
	if t.templates.Lookup(name) == nil {
		return fmt.Errorf("theme template %q not found", name)
	}
	return t.templates.ExecuteTemplate(w, name, data)
}

func (t *Theme) CopyAssets(output string) error {
	if t.assetsDir == "" {
		return nil
	}

	destinationRoot := filepath.Join(output, "_rootmark")
	return filepath.WalkDir(t.assetsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("theme assets cannot contain symlinks: %s", entry.Name())
		}
		rel, err := filepath.Rel(t.assetsDir, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o644)
	})
}

func (t *Theme) Close() error {
	if t.tempDir == "" {
		return nil
	}
	return os.RemoveAll(t.tempDir)
}

const defaultTemplate = `<!doctype html>
<html lang="{{.Site.Language}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light dark">
  <title>{{if .Page.Title}}{{.Page.Title}}{{if .Site.Title}} · {{.Site.Title}}{{end}}{{else}}{{.Site.Title}}{{end}}</title>
  {{- if .Page.Description}}
  <meta name="description" content="{{.Page.Description}}">
  {{- else if .Site.Description}}
  <meta name="description" content="{{.Site.Description}}">
  {{- end}}
  {{- if .Page.Author}}
  <meta name="author" content="{{.Page.Author}}">
  {{- else if .Site.Author}}
  <meta name="author" content="{{.Site.Author}}">
  {{- end}}
  {{- if .Page.Canonical}}
  <link rel="canonical" href="{{.Page.Canonical}}">
  {{- end}}
  {{- if .Site.Favicon}}
  <link rel="icon" href="{{.Site.Favicon}}">
  {{- end}}
  {{- if .Site.RSS}}
  <link rel="alternate" type="application/rss+xml" title="{{.Site.Title}} RSS" href="{{.Site.RSS}}">
  <link rel="alternate" type="application/atom+xml" title="{{.Site.Title}} Atom" href="{{.Site.Atom}}">
  {{- end}}
  {{- if .Page.Title}}
  <meta property="og:title" content="{{.Page.Title}}">
  {{- end}}
  {{- if .Page.Description}}
  <meta property="og:description" content="{{.Page.Description}}">
  {{- else if .Site.Description}}
  <meta property="og:description" content="{{.Site.Description}}">
  {{- end}}
  <meta property="og:type" content="{{if .Page.IsPost}}article{{else}}website{{end}}">
  {{- if .Page.Canonical}}
  <meta property="og:url" content="{{.Page.Canonical}}">
  {{- end}}
  {{- if .Site.Title}}
  <meta property="og:site_name" content="{{.Site.Title}}">
  {{- end}}
  {{- if .Page.ImageCanonical}}
  <meta property="og:image" content="{{.Page.ImageCanonical}}">
  {{- end}}
  {{- if and .Page.IsPost .Page.DateISO}}
  <meta property="article:published_time" content="{{.Page.DateISO}}">
  {{- end}}
  {{- if and .Page.IsPost .Page.UpdatedISO}}
  <meta property="article:modified_time" content="{{.Page.UpdatedISO}}">
  {{- end}}
  {{- range .Page.Tags}}
  <meta property="article:tag" content="{{.}}">
  {{- end}}
  <style>
    :root {
      color-scheme: light;
      --bg: #f5f5f2;
      --panel: #ffffff;
      --text: #181a1b;
      --muted: #686d72;
      --line: #d9d9d4;
      --soft: #efefeb;
      --accent: #111315;
      --accent-text: #ffffff;
      --focus: #536dfe;
      --shadow: 0 18px 50px rgba(20, 22, 24, .07);
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        color-scheme: dark;
        --bg: #0c0e0f;
        --panel: #121516;
        --text: #e9e9e4;
        --muted: #989d9f;
        --line: #292d2f;
        --soft: #181c1e;
        --accent: #f2f2ed;
        --accent-text: #111315;
        --focus: #8ea0ff;
        --shadow: 0 22px 60px rgba(0, 0, 0, .24);
      }
    }
    * { box-sizing: border-box; }
    html { background: var(--bg); }
    body { margin: 0; color: var(--text); background: var(--bg); line-height: 1.7; text-rendering: optimizeLegibility; }
    a { color: inherit; text-decoration-color: color-mix(in srgb, currentColor 42%, transparent); text-underline-offset: .18em; }
    a:hover { text-decoration-color: currentColor; }
    a:focus-visible, button:focus-visible { outline: 2px solid var(--focus); outline-offset: 3px; border-radius: .2rem; }
    .shell { width: min(100% - 2rem, 48rem); margin: 1.25rem auto 3.5rem; }
    .topbar { display: flex; align-items: center; gap: 1rem; min-height: 3.35rem; padding: .65rem .8rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); box-shadow: var(--shadow); }
    .brand { display: inline-flex; align-items: center; gap: .55rem; min-width: 0; font: 600 .88rem/1.2 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; text-decoration: none; letter-spacing: -.02em; }
    .brand img { width: 1.35rem; height: 1.35rem; object-fit: contain; border-radius: .2rem; }
    .prompt { color: var(--muted); font-weight: 500; }
    .topnav { margin-left: auto; display: flex; align-items: center; gap: .8rem; flex-wrap: wrap; justify-content: flex-end; }
    .topnav a { font: 500 .78rem/1.2 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; color: var(--muted); text-decoration: none; }
    .topnav a:hover { color: var(--text); }
    .primary { display: inline-flex; align-items: center; justify-content: center; min-height: 2.1rem; padding: 0 .75rem; border-radius: .45rem; background: var(--accent); color: var(--accent-text); font: 650 .76rem/1 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; text-decoration: none; white-space: nowrap; }
    .primary.link { min-height: auto; padding: 0; background: transparent; color: var(--text); text-decoration: underline; text-underline-offset: .22em; }
    main { padding: 3.2rem .15rem 1rem; }
    main > :first-child { margin-top: 0; }
    h1, h2, h3 { line-height: 1.2; letter-spacing: -.035em; }
    h1 { font-size: clamp(2rem, 8vw, 3rem); margin: 0 0 1.4rem; }
    h2 { margin-top: 2.6rem; }
    h3 { margin-top: 2rem; }
    p, ul, ol, blockquote { margin: 1.15rem 0; }
    img { max-width: 100%; height: auto; border-radius: .55rem; }
    hr { border: 0; border-top: 1px solid var(--line); margin: 2.5rem 0; }
    blockquote { margin-left: 0; padding: .1rem 0 .1rem 1rem; border-left: 2px solid var(--line); color: var(--muted); }
    pre { overflow-x: auto; padding: 1rem 1.1rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--soft); font-size: .88rem; line-height: 1.6; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: .9em; }
    :not(pre) > code { padding: .12rem .3rem; border: 1px solid var(--line); border-radius: .3rem; background: var(--soft); }
    .article-meta, .post-date, .post-tags { color: var(--muted); font: 500 .76rem/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    .article-meta { display: flex; gap: .5rem 1rem; flex-wrap: wrap; margin-bottom: 1.35rem; }
    .post-list { margin-top: 3rem; }
    .post-list > h2 { margin-bottom: 1rem; }
    .post-entry { padding: 1.25rem 0; border-top: 1px solid var(--line); }
    .post-entry:last-child { border-bottom: 1px solid var(--line); }
    .post-entry h3 { margin: .3rem 0 .4rem; font-size: 1.25rem; }
    .post-entry h3 a { text-decoration: none; }
    .post-entry p { margin: .35rem 0 0; color: var(--muted); }
    .post-nav { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 3rem; padding-top: 1.25rem; border-top: 1px solid var(--line); }
    .post-nav a { display: block; padding: .8rem; border: 1px solid var(--line); border-radius: .5rem; text-decoration: none; }
    .post-nav a:last-child { text-align: right; }
    .post-nav small { display: block; margin-bottom: .2rem; color: var(--muted); font: 500 .7rem/1.4 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    footer { margin-top: 3.7rem; padding: 1.2rem .15rem 0; border-top: 1px solid var(--line); color: var(--muted); font: 500 .74rem/1.65 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    .footer-row { display: flex; gap: .8rem 1.1rem; flex-wrap: wrap; align-items: center; }
    .footer-row + .footer-row { margin-top: .35rem; }
    .footer-spacer { flex: 1; }
    footer a { color: inherit; }
    @media (max-width: 680px) {
      .shell { width: min(100% - 1.1rem, 48rem); margin-top: .55rem; }
      .topbar { align-items: flex-start; flex-wrap: wrap; }
      .topnav { width: 100%; margin-left: 0; justify-content: flex-start; }
      main { padding-top: 2.5rem; }
      .post-nav { grid-template-columns: 1fr; }
      .post-nav a:last-child { text-align: left; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <a class="brand" href="{{.Site.Home}}">
        {{- if .Site.Logo}}<img src="{{.Site.Logo}}" alt="">{{end}}
        <span class="prompt" aria-hidden="true">~/</span><span>{{if .Site.Title}}{{.Site.Title}}{{else}}rootmark{{end}}</span>
      </a>
      {{- if or .Site.Navigation .Site.Primary.Label}}
      <nav class="topnav" aria-label="Main navigation">
        {{- range .Site.Navigation}}
        <a href="{{.URL}}">{{.Label}}</a>
        {{- end}}
        {{- if .Site.Primary.Label}}
        <a class="primary{{if eq .Site.Primary.Style "link"}} link{{end}}" href="{{.Site.Primary.URL}}">{{.Site.Primary.Label}}</a>
        {{- end}}
      </nav>
      {{- end}}
    </header>
    <main>
      {{- if .Page.IsPost}}
      <div class="article-meta">
        {{- if .Page.DateISO}}<time datetime="{{.Page.DateISO}}">{{.Page.Date}}</time>{{end}}
        {{- if .Page.Author}}<span>{{.Page.Author}}</span>{{end}}
        {{- if .Page.Tags}}<span>{{range $i, $tag := .Page.Tags}}{{if $i}} · {{end}}{{$tag}}{{end}}</span>{{end}}
      </div>
      {{- end}}
      {{.Page.Content}}
      {{- if and .Page.IsPost (or .Page.Previous .Page.Next)}}
      <nav class="post-nav" aria-label="Post navigation">
        <div>{{if .Page.Previous}}<a href="{{.Page.Previous.URL}}"><small>Previous</small>{{.Page.Previous.Title}}</a>{{end}}</div>
        <div>{{if .Page.Next}}<a href="{{.Page.Next.URL}}"><small>Next</small>{{.Page.Next.Title}}</a>{{end}}</div>
      </nav>
      {{- end}}
    </main>
    {{- if or .Site.Footer.Text .Site.Footer.Links .Site.Footer.GeneratedWith .Site.Footer.DevelopedBy.Label}}
    <footer>
      {{- if or .Site.Footer.Text .Site.Footer.Links}}
      <div class="footer-row">
        {{- if .Site.Footer.Text}}<span>{{.Site.Footer.Text}}</span>{{end}}
        {{- range .Site.Footer.Links}}<a href="{{.URL}}">{{.Label}}</a>{{end}}
      </div>
      {{- end}}
      {{- if or .Site.Footer.GeneratedWith .Site.Footer.DevelopedBy.Label}}
      <div class="footer-row">
        {{- if .Site.Footer.GeneratedWith}}<span>Generated with <a href="https://github.com/bpstr/rootmark">Rootmark</a></span>{{end}}
        <span class="footer-spacer"></span>
        {{- if .Site.Footer.DevelopedBy.Label}}<span>Developed by {{if .Site.Footer.DevelopedBy.URL}}<a href="{{.Site.Footer.DevelopedBy.URL}}">{{.Site.Footer.DevelopedBy.Label}}</a>{{else}}{{.Site.Footer.DevelopedBy.Label}}{{end}}</span>{{end}}
      </div>
      {{- end}}
    </footer>
    {{- end}}
  </div>
</body>
</html>
`
