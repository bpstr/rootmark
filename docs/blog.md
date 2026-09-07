---
title: Blog preset
description: Publish a chronological Markdown blog with feeds and site metadata.
---

# Blog preset

The blog preset keeps Rootmark's single `Content` model and adds publication defaults around it. There are no posts directories or collections to configure.

```yaml
site:
  title: My blog
  description: Notes about software and other things.
  url: https://example.com/
  author: Example Author

preset: blog
```

A dated Markdown file is treated as a post:

```md
---
title: Hello world
description: My first post.
date: 2026-09-07
tags:
  - notes
---

# Hello world

Write normally in Markdown.
```

An undated Markdown file remains a normal page, so `about.md` still becomes `/about/`.

## Generated output

With `preset: blog`, Rootmark automatically provides:

- newest-first post listing on `/`
- post date, author and tag metadata
- previous/next post navigation
- RSS 2.0 at `/rss.xml`
- Atom at `/atom.xml`
- `/sitemap.xml`
- `/robots.txt` pointing to the sitemap
- canonical and Open Graph metadata
- generated `/404.html`
- draft exclusion with `draft: true`

`site.url` is required because feeds and sitemap entries must use absolute URLs.

If the repository already contains `index.md`, its rendered content is kept as the introduction and the chronological post list is appended below it. If no `index.md` exists, Rootmark creates the blog home from `site.title` and `site.description`.

A custom `404.md` replaces the generated not-found page. A custom `robots.txt` is copied unchanged instead of the generated default.

## Publication metadata

Blog posts can use:

```yaml
---
title: Shipping Rootmark
description: Release notes and implementation details.
date: 2026-09-07
updated: 2026-09-08
author: Example Author
image: images/rootmark.png
tags:
  - go
  - static-sites
canonical: https://example.com/original-post/
draft: false
---
```

`date` and `updated` accept `YYYY-MM-DD` or RFC3339 values. Page-level `author` overrides `site.author`. `image` is used for social metadata, and `canonical` can override the generated canonical URL.

If `description` is omitted, Rootmark uses the first prose paragraph as a compact description.

## Feeds and sitemap

RSS, Atom, sitemap and robots generation are enabled by the blog preset. RSS and Atom contain dated content only and are ordered newest first.

They can be disabled explicitly:

```yaml
preset: blog
feed: false
sitemap: false
```

Disabling `sitemap` also disables automatic `robots.txt` generation.

## Themes

The built-in theme displays blog metadata and previous/next navigation automatically. Community themes receive the same publication data through `.Page`, `.Posts`, `.Site.RSS` and `.Site.Atom`; see [Themes](/themes/) for the full template context.
