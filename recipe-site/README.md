# recipe-site

The Hugo site that renders the recipes. It uses no JavaScript or CSS
frameworks: search, dark mode, and ingredient checkboxes are implemented in
plain JavaScript under `static/js/`, and all styling lives in
`static/css/style.css`.

For build and recipe-import instructions, see the [root README](../README.md).
This file covers customising the site itself.

## Layout

```
layouts/
├── index.html              # homepage
├── index.json              # JSON search index (served at /index.json)
├── _default/
│   ├── baseof.html         # base template
│   ├── list.html           # section and listing pages
│   ├── taxonomy.html       # a single tag's recipes
│   └── terms.html          # the tag index
├── recipes/single.html     # an individual recipe page
└── partials/
    ├── header.html         # header and search box
    ├── footer.html
    ├── recipe-card.html    # card used in listings
    ├── duration.html       # formats an ISO 8601 duration
    └── total-time.html     # sums prep and cook time
static/
├── css/style.css           # all styles
└── js/
    ├── search.js           # client-side search against /index.json
    ├── main.js             # dark mode, ingredient checkboxes
    └── theme-init.js       # applies the saved theme before first paint
```

## Common changes

**Site title and description.** Edit `hugo.toml`:

```toml
title = "Recipe Collection"

[params]
  description = "A collection of recipes"
```

**Colours.** Edit the CSS variables at the top of `static/css/style.css`
(`:root { --color-primary: ...; --color-accent: ...; }`).

**Which recipe fields show.** Toggle the `show*` params in `hugo.toml`
(`showPrepTime`, `showCookTime`, `showServings`, `showNutrition`) or edit
`layouts/recipes/single.html`.

**Recipes per page.** Change `pagerSize` under `[pagination]` in `hugo.toml`.

## Search

The search index is generated at `/index.json` whenever the site is built,
driven by `layouts/index.json` and the `[outputs]` block in `hugo.toml`.
`static/js/search.js` fetches it and ranks matches by title, description, tags,
and ingredients. No server is involved.

## Generated content

`content/recipes/` holds the markdown produced by `recipe-tool convert`. Only
`_index.md` (the section page) is tracked in git; the individual recipe files
are regenerated from YAML on every build, so don't edit them by hand — change
the source YAML or the converter instead.
