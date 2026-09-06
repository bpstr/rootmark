---
title: Rootmark
description: Markdown repositories in, static websites out.
---

# Rootmark

Rootmark is a small static site generator for repositories that should stay focused on Markdown content.

There is one content model: Markdown files. Rootmark discovers them, renders them with a built-in page layout, and writes static HTML.

- [Getting started](./getting-started/)
- [Configuration](./configuration/)

## Repository shape

A basic Rootmark site can be this small:

```text
.
├── index.md
├── about.md
└── .github/
    └── rootmark.yml
```

No application source tree or JavaScript runtime is required in the content repository.
