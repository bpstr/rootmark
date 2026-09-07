---
title: Docs preset
description: Turn a Markdown repository into polished documentation.
---

# Docs preset

Use the `docs` preset for product documentation, manuals, architecture notes and knowledge repositories.

```yaml
site:
  title: My project
  repository: https://github.com/example/project

preset: docs
```

The content model does not change. Keep normal Markdown files and folders:

```text
index.md
guides/
  index.md
  installation.md
  configuration.md
reference.md
```

Rootmark derives the documentation structure from that tree.

## What it adds

The docs preset adds:

- hierarchical navigation generated from Markdown routes
- active-page state
- an "On this page" table of contents from `h2` through `h4`
- previous and next page navigation
- responsive desktop and mobile documentation layouts
- an **Edit this page** link when `site.repository` is a public GitHub repository

Navigation order follows the generated route tree. Section `index.md` pages naturally sort before their children.

## Repository links

When `site.repository` points to GitHub, Rootmark derives edit URLs using the `main` branch and the configured `build.source` path.

For example:

```yaml
site:
  repository: https://github.com/example/project

build:
  source: docs
```

A generated page sourced from `docs/getting-started.md` links to:

```text
https://github.com/example/project/edit/main/docs/getting-started.md
```

No repository URL means no edit link.

## Themes

Docs enhancements are applied around the active theme's `<main>` element. Community themes therefore need a normal `<main>` container, which the Rootmark theme format examples already use.

The preset ships its small documentation layout CSS with the generated page so it remains usable with third-party themes. A theme can override the `rootmark-docs-*` classes when it wants a different presentation.

## Publication outputs

Unlike `blog`, the docs preset does not force RSS, Atom or sitemap generation. This keeps local documentation builds zero-config and avoids requiring `site.url`.

You can still enable a sitemap explicitly:

```yaml
preset: docs
sitemap: true

site:
  url: https://docs.example.com/
```
