# Hugo Presentation Layer

How the documentation space is published as a static site. Loaded on demand — `SKILL.md`
carries only the dispatch.

## Ownership model

`docs/` is both the **Hugo project root** and the **single content source**. Hugo runs with
`docs/` as its working directory, and every documentation path is *mounted* into Hugo
rather than copied, so the Markdown tree is never duplicated or rewritten.

| Path | Owner | Notes |
|------|-------|-------|
| `docs/<type>/**.md` | documentation | The content. Never touched by the site layer. |
| `docs/hugo.toml` | shared | Scaffold owns only the managed mount block; the rest is yours. |
| `docs/layouts/**` | scaffold (yours after edit) | Minimal self-contained templates. |
| `docs/static/css/site.css` | scaffold (yours after edit) | Single stylesheet, no build step. |
| `docs/.gitignore` | scaffold | Ignores `public/`, `resources/`, `.hugo_build.lock`. |
| `docs/public/` | Hugo | Build output. Never committed, never documentation. |

Scaffold-owned directories (`layouts`, `static`, `public`, `resources`, `themes`,
`archetypes`) are **not** documentation: the reconcile loop must not triage them as content,
and they carry no `.md`, so `docs-utils.py --action validate` stays untouched by them.

## Why mounts, and why `index.md` is remapped

Two project conventions collide with Hugo defaults, and mounts resolve both:

1. **Hugo reads `content/` by default; our Markdown lives in typed directories.** Each
   documentation directory is mounted to `content/<dir>`.
2. **Directory indexes are `index.md`** (project naming convention; a nested `README.md` is
   forbidden). In Hugo an `index.md` makes its directory a *leaf bundle*, which demotes every
   sibling page to a mere resource — the siblings stop being pages at all. So each `index.md`
   is mounted as `content/<dir>/_index.md` (branch bundle) and excluded from the directory
   mount via `excludeFiles`, leaving the file itself untouched on disk.

A directory containing no Markdown at all (pure media, e.g. `docs/assets/`) is mounted into
`static/<dir>` instead. Content directories additionally get a Markdown-excluding static
mount, so images stored beside their prose are published at the same relative path.

Because an explicit mount **replaces** Hugo's default mount for that component, the managed
block re-declares `static → static` for the scaffold's own stylesheet.

## Link and image resolution

The Markdown keeps repo-native relative links (`../concepts/overview.md`) — no site-specific
syntax, no shortcodes. Two render hooks translate them at build time:

- `layouts/_default/_markup/render-link.html` resolves a `.md` destination through Hugo's
  page graph (`.Page.GetPage`) and emits the target page's real URL. Correct under any URL
  scheme; anything that does not resolve to a page (external links, bare fragments,
  unpublished paths) passes through untouched.
- `layouts/_default/_markup/render-image.html` resolves a relative image path against the
  content file's own directory, then emits it relative to the current page URL.

With `relativeURLs = true` every generated URL is page-relative, so the output works under a
sub-path (project Pages sites) and is browsable straight off disk via `file://`.

> Pitfall, verified on 0.141.0: the map form of `uglyURLs` (`[uglyURLs] page = true`) is
> **not** honored — Hugo silently falls back to pretty URLs. Do not try to fix link shapes
> with `uglyURLs`; the render hooks are the mechanism.

## Publish scope

Everything under `docs/` is published — the six formal types, `notes/`, `archive/`, and media.
`notes/` frontmatter uses `status: draft`, which is *not* Hugo's `draft` key, so notes are
published like any other page. To withhold a zone instead, drop its mount from the managed
block and record the decision in `hugo.toml` above the block.

## Commands

```bash
# scaffold or repair the site layer (safe local writes; never clobbers your edits)
python3 ${SKILL_HOME}/scripts/scaffold-hugo.py --action scaffold --root . --site-title "<title>"

# report drift only (missing files, stale mounts, reserved-name collisions) — no writes
python3 ${SKILL_HOME}/scripts/scaffold-hugo.py --action check --root .

# print the computed mount block without touching disk
python3 ${SKILL_HOME}/scripts/scaffold-hugo.py --action mounts --root .

# build the static site into docs/public (skipped with guidance if hugo is absent)
python3 ${SKILL_HOME}/scripts/scaffold-hugo.py --action build --root . [--base-url <url>]

# local preview with live reload
cd docs && hugo server
```

File actions are reported per path: `created`, `unchanged`, `kept` (you edited it — left
alone), `overwritten` (only with `--force`), `mounts-synced`, `unmanaged` (the managed block
markers are missing from `hugo.toml`; nothing was written). A repeat run on an unchanged tree
reports `unchanged` for every path and writes nothing.

## CI guidance

No workflow file is generated — wire the build into whatever CI the project already uses. The
step is the same everywhere: install Hugo extended, build with the deployment `baseURL`,
publish `docs/public`. For GitHub Actions:

```yaml
- uses: peaceiris/actions-hugo@v3
  with:
    hugo-version: latest
    extended: true
- name: Build documentation site
  working-directory: docs          # docs/ is the Hugo project root
  run: hugo --minify --baseURL "${{ steps.pages.outputs.base_url }}/"
- uses: actions/upload-pages-artifact@v3
  with:
    path: docs/public
```

Two CI notes: run Hugo from `docs/`, not the repository root (the config lives in `docs/`), and
pass `--baseURL` explicitly when the site is served from a sub-path. A build that must fail on
broken content can add `--panicOnWarning`.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `Error: Unable to locate config file` | Hugo run from the repository root | `cd docs` first, or use `--action build` |
| A section renders one page and swallows its siblings | `index.md` reached Hugo unmapped (leaf bundle) | Re-run `--action scaffold` to regenerate the remaps |
| New documentation directory missing from the site | Mount block is stale | `--action check` reports it; `--action scaffold` syncs it |
| `toml: table X already exists` | Duplicate table appended to `hugo.toml` | Merge your keys into the existing table |
| Raw HTML stripped from a page | `markup.goldmark.renderer.unsafe` disabled | Keep `unsafe = true` in `hugo.toml` |
| Images 404 | Media directory added after scaffolding | Re-run `--action scaffold` to add the static mount |
| `'assets' is a Hugo component name` | A docs directory shadows a Hugo component | Works (it is mounted explicitly); rename for clarity if convenient |

Verified against `hugo v0.141.0+extended`. Module mounts require Hugo ≥ 0.56; the extended
build is not required by this scaffold (no SCSS).
