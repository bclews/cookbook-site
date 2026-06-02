# CLAUDE.md

Guidance for Claude Code (claude.ai/code) when working in this repository.

## Overview

A Go CLI (`recipe-tool`) plus a Hugo site that convert CookBook Manager YAML
exports into a self-hosted static recipe website. The repository ships only the
tooling and a few example recipes — real recipe exports are git-ignored and
never committed here.

## Commands

```bash
make build       # validate, convert, download images, build the Hugo site
make serve       # Hugo dev server (needs recipes converted first)
make validate    # validate recipe YAML
make convert     # YAML -> Hugo markdown (PARALLEL=N sets image workers, default 10)
make quick-build # rebuild without re-downloading images
make test        # CGO_ENABLED=0 go test ./recipe-site/...
make lint        # gofmt check + go vet
make clean       # remove generated markdown and public/ (FULL=1 also images)
```

Run a single test:

```bash
CGO_ENABLED=0 go test ./recipe-site/internal/recipes/ -run TestName -v
```

## Architecture

- `recipe-site/cmd/recipe-tool/main.go` — CLI: `validate`, `convert`, `import`,
  `cleanup`. Resolves the Hugo site root by walking up from the working
  directory to the nearest `hugo.toml` (`siteRootDir`).
- `recipe-site/internal/recipes/`
  - `types.go` — `Recipe` struct; `StringOrSlice` accepts a YAML string or list.
  - `validate.go` — required/recommended field checks.
  - `convert.go` — YAML to Hugo markdown; `FindYAMLDirectory` discovery order.
  - `download.go` — parallel image downloads with rate limiting, retries, a
    circuit breaker, and SSRF guards; deterministic hash filenames with caching.
  - `import.go` — ZIP extraction with path-traversal and size limits.
- `recipe-site/layouts/`, `static/`, `hugo.toml` — the Hugo site.

### Data flow

YAML (`CookBook-Recipes-YAML/`, an `imports/` ZIP, or `examples/recipes/`) →
validate → download images → convert to `content/recipes/*.md` → `hugo` builds
`recipe-site/public/`.

## Conventions

- Recipe markdown in `content/recipes/` is generated — edit the source YAML or
  the converter, never the markdown. Only `_index.md` is tracked.
- Real recipe exports and downloaded images are git-ignored; keep it that way.
  This repo is public, so never commit recipe content or images.
- Run `make lint` and `make test` after changing Go code.
- Image filenames are deterministic hashes, so the same source produces the
  same filenames and re-runs reuse the cache.
