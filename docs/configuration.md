---
title: Configuration
description: Configure a Rootmark site.
---

# Configuration

Rootmark intentionally keeps configuration small. The default file is `.github/rootmark.yml` and every field is optional.

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

build:
  source: .
  output: _site

theme: https://github.com/example/rootmark-theme

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

`url` is the canonical public site URL. Rootmark combines it with each generated route to emit canonical URLs. `author` becomes author metadata.

`favicon` and `logo` may point to local site files or absolute URLs. Local paths are resolved from every generated page. Ordinary non-Markdown files in the content tree are copied to the same path in the generated site, so a root-level `favicon.svg` needs no special assets directory.

`repository` exposes the source repository URL to themes.

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

A Markdown file can define a title, description and theme template:

```md
---
title: Installation
description: Install the project.
template: default
---

# Installation
```

Frontmatter is optional. If `template` is omitted, the active theme's `default.html` is used.
