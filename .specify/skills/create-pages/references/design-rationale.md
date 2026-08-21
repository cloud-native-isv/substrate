# Design Rationale — create-pages

Why each stage boundary and guarantee exists. Every item was observed as a real
failure, not inferred. Observations marked *(v0.163.3)* were reproduced in this
repository against Hugo v0.163.3 extended; the earlier ones come from the
original implementation run (Hugo v0.163.2 in
`reg.docker.alibaba-inc.com/xuanji-images/hugo:latest`).

## Stage boundaries

### Why three stages instead of one scaffold

The first implementation fused rendering and hosting into a single script: one
run wrote Hugo config, layouts, a build script, a CI pipeline, and a staging
directory. Two consequences showed up immediately — a hosting change forced a
re-run that also rewrote rendering artifacts, and the build script existed only
to serve one CI pipeline's expectations (a repository-root `dist/`). Splitting
content → rendering → hosting makes each failure attributable and lets a project
change its hosting target without touching a single rendered file.

### Why the docs directory is the only content source

Upper-layer systems (framework detectors, IDE indexers, AI instruction
generators, code-review heuristics) judge a project's type from its root files.
A root-level `hugo.toml` / `layouts/` makes a Go backend project look like a
Hugo docs project. Keeping every artifact inside the docs directory preserves
the project's true identity and makes the capability removable: deleting the
docs directory must leave core logic and build flow untouched.

The one sanctioned exception is the platform CI file. Hosting platforms discover
pipelines at a fixed repository-root path (`.aoneci/`, `.github/workflows/`), so
it physically cannot live in the docs directory. It is one file, it is named in
the report, and nothing else joins it.

### Why one renderer, not one per hosting target

The retired second renderer copied the whole Markdown tree into
`docs/.hugo-content/`, renamed files inside the copy, and published a
repository-root `dist/`. It worked, but it duplicated every document (two
sources of truth during a build), needed a root `.gitignore` entry, and wrote
outside the docs directory. Mount-based rendering achieves the same result with
zero duplication, so the staging renderer had no remaining justification.

## Rendering guarantees

### `index.md` stays `index.md`; the mount remaps it

Hugo bundle semantics: `index.md` in a directory is a **leaf bundle** — sibling
`.md` files in that directory are not rendered as pages. `_index.md` is a
**branch bundle** and siblings render normally. The docs taxonomy uses
`index.md` as each type directory's index, so a naive build silently drops
individual document pages: 18 output files instead of 39 (only index pages).
The mount block therefore maps each `<dir>/index.md` to
`content/<dir>/_index.md` and excludes it from the directory mount. The file on
disk keeps the name the docs convention requires.

### Scaffold-owned directories are never mounted as content

`layouts/index.html` reachable as content makes Hugo try to build a page from
it and abort with a security-policy error (`"text/html" is not whitelisted`);
a config file reachable as content gets copied into the output and published.
The mount computation therefore treats `layouts`, `static`, `public`,
`resources`, `themes`, `archetypes` as scaffold-owned, never as content — the
same list a `create-docs` reconcile run must skip when triaging.

### Title fallback partial *(v0.163.3)*

Documents here carry no YAML front matter, and Hugo returns an **empty**
`.Title` for such pages. Observed before the fix: a page rendered
`<title>· Spec Kit</title>`, and every list/nav entry built from `.LinkTitle`
rendered as a blank link. `partials/title.html` falls back to the first `<h1>`
of the rendered content — the document's authoritative title — then to a
humanized filename. Verified after the fix: `Spec Driven Development · Demo
Docs`, and non-empty link text on home, list, and nav.

### Raw HTML safety — `unsafe = true`

Documentation legitimately contains inline HTML (`<details>`, `<br>`, badges,
`<span style=...>`). Goldmark's default renderer **omits** raw HTML with a
warning, silently losing content; observed 137 `span style` occurrences
preserved in one material after enabling it.

### `relativeURLs = true`, `disableKinds`

Relative URLs keep the built site working under a hosting sub-path and when
browsed straight off disk. `disableKinds = ["taxonomy", "term", "RSS"]` stops
Hugo emitting empty `categories/`/`tags/` indexes and an unused feed for a site
that uses none.

### Config keys kept current *(v0.163.3)*

`languageCode` was deprecated in Hugo v0.158.0 in favour of `locale`, and
`.Site.LanguageCode` in favour of `.Site.Language.Locale`; both emitted build
warnings and are now fixed. **Still outstanding**: `module.mounts.excludeFiles`
was deprecated in v0.153.0 and replaced by a `files` setting — the mount
generator still emits `excludeFiles`, so builds print one deprecation warning.
It functions on v0.163.3; migrating it requires checking the `files` semantics
against current Hugo documentation, because the index remap depends on the
exclusion behaviour.

### Shipped-asset upgrades do not reach an existing site by themselves

The scaffolder classifies any file whose content differs from the shipped asset
as user-edited and reports it `kept`. That protects real edits, but it also
means a fixed layout in a newer skill version stays out of an already-scaffolded
site until someone passes `--force` — which also discards genuine local edits.
Diff before forcing.

## Hosting guarantees

### CI guard — `if [ -d <docs> ]`

The pipeline triggers on every push. Without the guard, deleting the docs
directory fails the build step (the directory the build `cd`s into is missing).
The guard makes the step a no-op, and the follow-up unconditional
`mkdir -p <docs>/public` guarantees the publish step always finds its
`deploy-dir`.

### Build from inside the docs directory

`hugo.toml` lives in the docs directory, which is the Hugo project root. Running
Hugo from the repository root picks up no config and produces an empty site, so
every pipeline invokes `(cd <docs> && hugo --minify)` and publishes
`<docs>/public`.
