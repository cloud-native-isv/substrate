# Platform: github (structural stub — no implementation yet)

This directory reserves the structure for GitHub-hosted deployments
(GitHub Actions + GitHub Pages). Only the structure exists; the concrete
template is intentionally NOT implemented. When implementing, follow the
contract below so `scaffold.sh` can pick it up without code changes.

## Expected template file

- Name: `deploy-pages.yaml.tpl` (scaffold.sh looks for exactly this name).
- Rendered to: `.github/workflows/deploy-pages.yaml` (the platform's
  canonical CI location — scaffold.sh must learn this mapping when the
  template is implemented).

## Placeholder contract (shared by every platform template)

`scaffold.sh` substitutes these tokens before writing:

| Placeholder | Meaning |
|-------------|---------|
| `__SITE_NAME__` | Deploy site name (`--site-name`) |
| `__TITLE__` | Site title (`--title`) |
| `__BRANCH__` | Production branch (`--branch`) |
| `__IMAGE__` | Hugo build image (`--image`) |
| `__DOCS_DIR__` | Docs directory (`--docs-dir`) |

## Mandatory pipeline semantics (mirror the aoneci template)

1. Trigger on push to the repository.
2. Build step guarded by a docs-directory existence check
   (`if [ -d __DOCS_DIR__ ]`), followed by an unconditional `mkdir -p dist`
   so the deploy/publish step always finds its artifact directory.
3. Build invocation: `bash __DOCS_DIR__/scripts/build-docs.sh`.
4. Publish the `dist/` directory to the platform's pages service, pinned to
   `__BRANCH__` for production.

Until the template exists, `scaffold.sh --platform github` scaffolds all
platform-independent artifacts (Hugo config, layouts, build script) and
emits a warning that the CI file must be authored manually per this README.
