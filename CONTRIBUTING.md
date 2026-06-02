# Contributing

Thanks for your interest in improving cookbook-site. This repository ships the
tooling and site templates only — never commit recipe content or downloaded
images (see `.gitignore`).

## Getting set up

You need Go 1.22.4+ and Hugo (extended) 0.161.1+. Run `make install` to check
both are present.

```bash
make build   # build the site from the bundled example recipes
make serve    # preview at http://localhost:1313
```

## Before opening a pull request

```bash
make lint    # gofmt check + go vet
make test    # CGO_ENABLED=0 go test ./...
```

Both must pass; CI runs the same checks plus a smoke build of the example
recipes. Please:

- Keep changes focused and match the style of the surrounding code.
- Add or update tests for behavior changes in `internal/recipes/`.
- Update the relevant README/`CLAUDE.md` docs when you change commands, flags,
  the recipe format, or site configuration.
- Edit the source YAML or the converter, never the generated markdown in
  `recipe-site/content/recipes/`.

## Reporting bugs

Open an issue with the `recipe-tool version`, your Go and Hugo versions, the
command you ran, and the output. A minimal example recipe that reproduces the
problem helps a lot.
