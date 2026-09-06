---
title: Configuration
description: Rootmark's initial configuration surface.
---

# Configuration

Rootmark intentionally starts with a very small configuration surface. The default file is `.github/rootmark.yml`.

```yaml
site:
  title: My site
  description: Optional site description.
  language: en

build:
  source: .
  output: _site
```

All fields are optional.

## `site`

`title` is appended to generated page titles. `description` describes the site configuration and is available for future site-level metadata. `language` sets the generated HTML `lang` attribute and defaults to `en`.

## `build`

`source` is the directory Rootmark scans recursively for Markdown and defaults to the repository root. `output` is the generated site directory and defaults to `_site`.

CLI flags override these values:

```sh
rootmark build --source docs --output _site
```

## Page frontmatter

A Markdown file may define a title and description:

```md
---
title: Installation
description: Install the project.
---

# Installation
```

Frontmatter is optional. `README.md` and hidden directories are ignored by the initial content discovery rules.
