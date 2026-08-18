# Design Rationale — create-pages

Why behind each core principle. Every item below was observed as a real failure
during the original implementation run (docker builds against
`reg.docker.alibaba-inc.com/xuanji-images/hugo:latest`, Hugo v0.163.2).

## 1. Isolation — everything inside `docs/`

Upper-layer systems (framework detectors, IDE indexers, AI instruction
generators, code-review heuristics) judge a project's type from its root
files. A root-level `hugo.yaml` / `layouts/` makes a Go backend project look
like a Hugo/docs project. Keeping all Hugo artifacts inside `docs/` preserves
the project's true identity and makes the whole docs capability removable:
`rm -rf docs/` must leave core logic and build flow untouched.

## 2. CI guard — `if [ -d docs ]`

The deploy pipeline triggers on every push. Without the guard, deleting
`docs/` fails the build step (`bash docs/scripts/build-docs.sh`: file not
found). The guard makes the step a no-op; the follow-up `mkdir -p dist`
guarantees the deploy step always finds its `deploy-dir`.

## 3. `index.md` → `_index.md` transform

Hugo bundle semantics:

- `index.md` in a directory = **leaf bundle** — sibling `.md` files in the
  same directory are NOT rendered as pages.
- `_index.md` = **branch bundle / section page** — siblings render normally.

The docs taxonomy uses `index.md` as each type directory's index. Without the
transform, a build "succeeds" but silently drops every individual document
page: observed 18 output files instead of 39 (only the index pages rendered).
The transform happens in a staging copy (`.hugo-content/`), never in `docs/`
itself, so the docs convention (`index.md`) stays intact.

## 4. Staging exclusions

The staging copy must exclude `layouts/`, `scripts/`, `hugo.yaml`, and
`.hugo-content` itself:

- `layouts/index.html` inside the content dir → Hugo tries to create a page
  from it and aborts with a security-policy error
  (`"text/html" is not whitelisted`).
- `hugo.yaml` inside the content dir → silently copied into `dist/` and
  deployed.
- `--exclude=hugo.yaml` in `tar` matches the file at any depth; this is
  intentional (no doc is ever named `hugo.yaml`).

## 5. Raw HTML safety — `unsafe: true`

Collected external materials contain inline HTML (`<span style=...>` etc.).
Goldmark's default renderer **omits** raw HTML (with a warning), causing
content loss. `markup.goldmark.renderer.unsafe: true` renders it. Observed:
137 `span style` occurrences preserved in one material after the fix.

## 6. Clean output — `disableKinds: [taxonomy, term]`

Without it Hugo emits empty `categories/` and `tags/` index pages for a site
that uses no taxonomies.

## 7. Title fallback partial

Docs files carry metadata as an HTML blockquote line, not YAML front matter.
Hugo (≥0.163, observed) returns an **empty** `.Title` for pages without front
matter — verified: a bare `# Hello World` file produced `<title></title>`,
while the same file with front matter worked. The `title.html` partial falls
back to the first `<h1>` of the rendered content, which is the docs'
authoritative title.

## 8. `locale` instead of `languageCode`

`languageCode` is deprecated since Hugo 0.158 (build warning); `locale` is the
replacement.

## 9. Output location

`hugo` is invoked with `--contentDir <staging>` and `--destination
<project-root>/dist`. The config's `publishDir: dist` is only a fallback for
manual runs from `docs/`; the CI deploy step expects `dist/` at the project
root (`deploy-dir: dist/`).
