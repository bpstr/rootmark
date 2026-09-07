---
title: Rootmark
description: Markdown repositories in, static websites out.
---

# Rootmark

Rootmark is a small standalone Markdown-to-HTML site generator written in Go.

The repository stays about content. Markdown files and ordinary static assets are the site; `.github/rootmark.yml` contains the small amount of executable configuration needed to turn them into static HTML.

The built-in theme is narrow, responsive, developer-friendly and automatically adapts to light or dark mode without JavaScript. Community themes can completely replace it with one public repository URL.

## Start small

```text
index.md
about.md
.github/rootmark.yml
```

Build it:

```sh
rootmark build
```

Then publish `_site/` anywhere that serves static files.

## Pick a preset

Rootmark keeps the same Markdown content model while offering four focused presentations:

- `simple` — direct Markdown-to-page rendering
- `blog` — chronological posts, RSS/Atom and publishing metadata
- `docs` — hierarchical documentation navigation, TOC and edit links
- `portfolio` — project cards and case-study pages

Rootmark's own documentation now uses the `docs` preset, so this site is also its integration test.

Continue with [Getting started](/getting-started/), [Configuration](/configuration/), [Docs preset](/docs/), [Portfolio preset](/portfolio/) or [Themes](/themes/).
