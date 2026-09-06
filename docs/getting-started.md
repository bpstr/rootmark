---
title: Getting started
description: Install Rootmark and build a Markdown site.
---

# Getting started

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/bpstr/rootmark/main/install.sh | sh
```

You can also install it with Go:

```sh
go install github.com/bpstr/rootmark/cmd/rootmark@latest
```

## Add content

Create `index.md` in the repository root:

```md
# Hello world

This is my site.
```

Add more Markdown files wherever they naturally belong:

```text
index.md
about.md
docs/install.md
```

Rootmark generates:

```text
_site/index.html
_site/about/index.html
_site/docs/install/index.html
```

## Build

Run:

```sh
rootmark build
```

Rootmark reads `.github/rootmark.yml` when present. Configuration is optional for a root-level site.

## Publish with GitHub Pages

Use a GitHub Actions workflow to run `rootmark build`, upload `_site/`, and deploy it with the official GitHub Pages actions.

For a new repository, first open **Settings → Pages** and set **Source** to **GitHub Actions**. GitHub requires this repository setting before a workflow can perform the first deployment.

The Rootmark repository dogfoods this setup: its docs workflow builds this `docs/` directory using the Rootmark binary compiled from the same checkout.
