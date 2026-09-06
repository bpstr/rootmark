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
