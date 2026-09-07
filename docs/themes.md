---
title: Themes
description: Build and use community Rootmark themes.
---

# Themes

Rootmark themes are deliberately small and non-executable. They are HTML templates and static assets stored in a public Git repository.

Choose a theme by URL:

```yaml
theme: https://github.com/example/rootmark-theme
```

Rootmark clones the public repository for the build. No theme package manager or installation step is required.

## Theme repository

A minimal theme contains:

```text
theme.yml
templates/
  default.html
```

Static assets are optional:

```text
assets/
  theme.css
  logo.svg
```

The manifest declares the theme format:

```yaml
name: Terminal Notes
format: 1
```

Format `1` requires `templates/default.html`. Other `.html` files in `templates/` are optional page layouts.

Theme assets are copied to `_rootmark/` in the generated site. Templates receive `.ThemeAssets`, already resolved relative to the current page:

```html
<link rel="stylesheet" href="{{.ThemeAssets}}theme.css">
```

## Template context

Templates are Go `html/template` files. The format exposes data rather than executable plugins.

```text
.Site.Title
.Site.Description
.Site.Language
.Site.URL
.Site.Author
.Site.Favicon
.Site.Logo
.Site.Repository
.Site.Home
.Site.RSS
.Site.Atom
.Site.Navigation[]       # .Label .URL
.Site.Primary            # .Label .URL .Style
.Site.Footer.Text
.Site.Footer.Links[]     # .Label .URL
.Site.Footer.GeneratedWith
.Site.Footer.DevelopedBy # .Label .URL

.Page.Title
.Page.Description
.Page.Content
.Page.URL
.Page.Canonical
.Page.Source
.Page.Date
.Page.DateISO
.Page.Updated
.Page.UpdatedISO
.Page.Image
.Page.ImageCanonical
.Page.Author
.Page.Tags
.Page.IsPost
.Page.IsBlogIndex
.Page.Previous           # post data or nil
.Page.Next               # post data or nil

.Posts[]                 # newest-first dated content
  .Title
  .Description
  .URL
  .Canonical
  .Date
  .DateISO
  .Updated
  .UpdatedISO
  .Image
  .Author
  .Tags

.Root                    # relative path back to site root
.ThemeAssets             # relative URL to the theme asset directory
```

`.Site.RSS` and `.Site.Atom` are empty when feed generation is disabled. `.Posts` is available when Rootmark needs dated publication content; the blog preset marks dated pages with `.Page.IsPost` and provides previous/next post values.

A minimal template can be as small as:

```html
<!doctype html>
<html lang="{{.Site.Language}}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Page.Title}}</title>
  {{if .Site.RSS}}<link rel="alternate" type="application/rss+xml" href="{{.Site.RSS}}">{{end}}
  <link rel="stylesheet" href="{{.ThemeAssets}}theme.css">
</head>
<body>
  <main>{{.Page.Content}}</main>
</body>
</html>
```

A blog-aware theme can render publication metadata without introducing another content API:

```html
{{if .Page.IsPost}}
  <time datetime="{{.Page.DateISO}}">{{.Page.Date}}</time>
{{end}}

{{if .Page.IsBlogIndex}}
  {{range .Posts}}
    <article>
      <a href="{{.URL}}">{{.Title}}</a>
      <time datetime="{{.DateISO}}">{{.Date}}</time>
    </article>
  {{end}}
{{end}}
```

## Additional templates

A theme may add layouts such as `landing.html`:

```text
templates/
  default.html
  landing.html
```

A page selects it in frontmatter:

```md
---
title: Product
template: landing
---
```

Use `template: landing` or `template: landing.html`.

## Safety boundary

Rootmark does not load binaries, scripts, Go plugins or arbitrary commands from themes. Theme repositories can control generated HTML and ship browser assets, but they do not execute code inside the Rootmark build process.
