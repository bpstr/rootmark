---
title: Portfolio preset
description: Build a compact project portfolio from Markdown files.
---

# Portfolio preset

The `portfolio` preset is for developer portfolios, studio work, case studies and small project showcases.

```yaml
site:
  title: Jane Developer
  description: Selected software and product work.

preset: portfolio
```

A portfolio starts with `index.md` as the landing-page introduction. Other Markdown pages become projects automatically:

```text
index.md
rootmark.md
assign.md
commerce-platform.md
```

## Project metadata

Portfolio projects reuse Rootmark's existing frontmatter:

```md
---
title: Rootmark
description: A tiny Markdown-first static site generator.
date: 2026-09-07
image: images/rootmark.png
tags:
  - Go
  - static sites
  - open source
---

# Rootmark

Project details go here.
```

The generated landing page shows projects newest-first when dates are available, then alphabetically. Cards can include the project image, description, year and tags.

Project pages receive the same image and metadata treatment before their Markdown content.

## Normal pages

If a Markdown page should remain a normal page instead of appearing in the project grid, opt it out:

```md
---
title: About
project: false
---

# About
```

This is useful for pages such as About, Contact or Colophon while keeping the repository flat.

## Themes

Like the docs preset, portfolio enhancement is applied to the active theme's `<main>` element and uses `rootmark-portfolio-*` classes. The built-in styles provide a polished responsive grid, while community themes can override those classes.

The preset does not automatically enable RSS or sitemap generation. A sitemap can still be enabled explicitly when the portfolio has a public `site.url`.
