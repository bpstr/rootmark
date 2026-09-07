---
title: Configuration
description: Configure a Rootmark site.
---

# Configuration

Rootmark intentionally keeps configuration small. The default file is `.github/rootmark.yml` and every field is optional unless a feature needs an absolute public URL.

```yaml
site:
  title: My site
  description: Optional site description.
  language: en
  url: https://example.com/
  author: Example Author
  favicon: favicon.svg
  logo: logo.svg
  repository: https://github.com/example/project

preset: simple

theme: https://github.com/example/rootmark-theme

feed: false
sitemap: false

build:
  source: .
  output: _site

navigation:
  - label: Docs
    url: /docs/
  - label: GitHub
    url: https://github.com/example/project

primary:
  label: Get started
  url: /getting-started/
  style: button

footer:
  text: Built in the open.
  links:
    - label: License
      url: /license/
  generated_with: true
  developed_by:
    label: Example
    url: https://example.com
```

## Site

`title` is the site name and is appended to page titles. `description` becomes the fallback page description. `language` sets the HTML `lang` attribute and defaults to `en`.

`url` is the canonical public site URL. Rootmark combines it with each generated route to emit canonical URLs. It is required when feeds or sitemap generation are enabled. `author` becomes the default site/page author.

`favicon` and `logo` may point to local site files or absolute URLs. Local paths are resolved from every generated page. Ordinary non-Markdown files in the content tree are copied to the same path in the generated site, so a root-level `favicon.svg` needs no special assets directory.

`repository` exposes the source repository URL to themes.

## Preset

`preset` defaults to `simple`.

```yaml
preset: simple
```

`simple` renders the Markdown tree directly. `blog` keeps the same content model but treats dated content as posts and enables chronological publishing features, feeds, sitemap, social metadata and a generated 404.

```yaml
preset: blog
```

See [Blog preset](/blog/) for the complete behavior.

## Publication outputs

`feed` controls both RSS 2.0 (`/rss.xml`) and Atom (`/atom.xml`). `sitemap` controls `/sitemap.xml` and automatic `/robots.txt` generation.

They default to `false` for the simple preset and `true` for the blog preset. Explicit values always win:

```yaml
preset: blog
feed: false
sitemap: false
```

If the source already contains `robots.txt`, Rootmark preserves it instead of generating the default crawler policy.

## Build

`source` is the directory Rootmark scans recursively and defaults to the repository root. `output` is the generated site directory and defaults to `_site`.

CLI flags override these values:

```sh
rootmark build --source docs --output _site
```

## Theme

`theme` is a public HTTPS Git repository URL. Rootmark fetches it for the build and loads the Rootmark theme format from that repository. Leave it unset to use the built-in theme.

See [Themes](/themes/) for the theme repository format.

## Navigation

`navigation` is an ordered list of label/URL pairs. URLs beginning with `/` are site-root-relative and are converted to correct relative links on nested pages. Absolute URLs are kept unchanged.

## Primary CTA

`primary` defines one prominent action available to themes. `style` accepts `button` or `link` and defaults to `button`.

```yaml
primary:
  label: Open GitHub
  url: https://github.com/example/project
  style: button
```

## Footer

The footer can provide free text, links and small attribution elements. `generated_with` defaults to `true` and can be disabled explicitly.

```yaml
footer:
  text: MIT licensed.
  links:
    - label: GitHub
      url: https://github.com/example/project
  generated_with: true
  developed_by:
    label: Example
    url: https://example.com
```

Themes decide how these values look; Rootmark only supplies the data.

## Page frontmatter

Frontmatter is optional. The complete built-in metadata surface is:

```md
---
title: Installation
description: Install the project.
template: default
date: 2026-09-07
updated: 2026-09-08
draft: false
image: images/cover.png
author: Example Author
tags:
  - go
  - docs
canonical: https://example.com/custom-canonical/
---

# Installation
```

`title`, `description` and `template` work for every preset. If `title` is omitted, Rootmark uses the first level-one heading and then the filename. If `description` is omitted, Rootmark derives a short description from the first prose paragraph.

`date`, `updated`, `image`, `author`, `tags`, `canonical` and `draft` provide publication metadata. In the blog preset, dated content is treated as a post and participates in listings, feeds and previous/next navigation. `draft: true` excludes a page from the production build.

If `template` is omitted, the active theme's `default.html` is used.
