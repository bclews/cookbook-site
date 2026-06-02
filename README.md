# cookbook-site

A command-line tool and Hugo theme that turn [CookBook Manager](https://cookbookmanager.com/)
YAML exports into a static recipe website you can host yourself.

The repository contains two parts:

- **`recipe-tool`** — a small Go CLI that validates recipe YAML, downloads
  recipe images, and generates Hugo markdown.
- **`recipe-site/`** — a dependency-free Hugo site (vanilla CSS and JavaScript)
  that renders the recipes, with client-side search, tag browsing, and a
  print-friendly layout.

It ships with a few example recipes so you can build and preview the site
immediately. Your own recipes stay on your machine — see
[Using your own recipes](#using-your-own-recipes).

## Requirements

- [Go](https://go.dev/doc/install) 1.22.4 or later
- [Hugo](https://gohugo.io/installation/) (extended) 0.161.1 or later

Run `make install` to check that both are present.

## Quick start

Build the site from the bundled example recipes and preview it:

```bash
make build
make serve
```

`make build` validates the YAML, downloads any images, converts the recipes to
Hugo markdown, and runs `hugo --minify`. The output is written to
`recipe-site/public/`. `make serve` starts Hugo's development server at
http://localhost:1313.

## Using your own recipes

1. In CookBook Manager, export your recipes as YAML.
2. Put the exported `.yml` files in a `CookBook-Recipes-YAML/` directory at the
   repository root, **or** drop the export ZIP into `imports/`. Both locations
   are git-ignored, so your recipes are never committed.
3. Run `make build`.

`recipe-tool` looks for recipes in this order and uses the first match:

1. a directory passed with `--yaml-dir`
2. `imports/.extracted/CookBook-Recipes-YAML/` (created when you import a ZIP)
3. `CookBook-Recipes-YAML/`
4. `data/recipes/`
5. `examples/recipes/` (the bundled samples)

To import a ZIP manually:

```bash
make import     # extracts the newest ZIP from imports/
make build
```

## Recipe YAML format

Each recipe is a single `.yml` file. `name` is required; everything else is
optional.

```yaml
name: Weeknight Tomato Soup
description: A simple stovetop tomato soup made from pantry staples.
servings: 4 servings
prep_time: PT5M          # ISO 8601 duration
cook_time: PT25M
source: https://example.com/recipe
image: https://example.com/soup.jpg
tags:                      # browsable tag pages at /tags/
  - Dinner
  - Soup
keywords:                  # extra search/SEO terms, not shown on the page
  - pantry
  - vegetarian
ingredients:
  - 2 tbsp olive oil
  - 1 onion, diced
directions:
  - Soften the onion in the oil.
  - Add the tomatoes and simmer.
nutrition: |
  Calories: 180 kcal
notes: Stir through cream for a richer soup.
```

Times use ISO 8601 durations: `PT15M` (15 minutes), `PT1H` (1 hour),
`PT1H30M` (1 hour 30 minutes). `make validate` reports missing or malformed
fields before you build.

`tags` generate browsable pages under `/tags/`; `keywords` are not displayed
but feed the client-side search and the page's `keywords` metadata. Both accept
either a single value or a list.

## Commands

| Command | Description |
|---|---|
| `make build` | Validate, convert, download images, and build the site |
| `make serve` | Run the Hugo development server |
| `make validate` | Check recipe YAML for errors and warnings |
| `make convert` | Convert YAML to Hugo markdown (`PARALLEL=N` sets image workers) |
| `make quick-build` | Rebuild without re-downloading images |
| `make import` | Extract the newest ZIP from `imports/` |
| `make clean` | Remove generated markdown and `public/` (`FULL=1` also removes images) |
| `make test` | Run the Go unit tests |
| `make lint` | Run `gofmt` and `go vet` |
| `make stats` | Show counts of sources, markdown, images, and output |

Run `make help` for the full list.

## Self-hosting the built site

`recipe-site/public/` is plain HTML, CSS, JavaScript, and images. Serve it with
any static web server — for example:

```bash
# Nginx, Caddy, Apache: point the document root at recipe-site/public/
# Or serve it directly for a quick check:
cd recipe-site/public && python3 -m http.server 8080
```

If you serve the site from a subpath rather than a domain root, set `baseURL`
in `recipe-site/hugo.toml` accordingly and rebuild.

## How it works

```
YAML export ──▶ validate ──▶ download images ──▶ convert to markdown ──▶ hugo build
                (recipe-tool)                      (recipe-tool)          (static HTML)
```

Image filenames are deterministic hashes of the source URL, so re-running a
build reuses already-downloaded images instead of fetching them again.

## Project layout

```
.
├── cmd/recipe-tool/          # CLI entry point (the recipe-tool generator)
├── internal/recipes/         # validate, convert, download, import packages
├── recipe-site/              # the Hugo site
│   ├── layouts/              # Hugo templates
│   ├── static/css, static/js # vanilla CSS and JavaScript
│   ├── content/recipes/      # generated markdown (git-ignored except _index.md)
│   └── hugo.toml             # Hugo configuration
├── examples/recipes/         # sample recipes used by the quick start
├── bin/                      # compiled recipe-tool (git-ignored)
├── Makefile                  # build automation
└── .github/workflows/ci.yml  # lint, test, and a smoke build
```

The Go generator (`cmd/`, `internal/`) lives at the repository root, separate
from the Hugo site in `recipe-site/`.

The generated markdown in `content/recipes/` and the downloaded images in
`static/images/recipes/` are rebuilt from YAML, so they are not tracked in git.

## License

[MIT](LICENSE). This covers the tool and the site templates. Recipe content you
add is yours and is not part of this repository.
