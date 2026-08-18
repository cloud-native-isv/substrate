# CI platform templates — registry

`create-pages` separates platform-independent artifacts (Hugo config,
layouts, build script) from the platform-specific CI pipeline. Each
subdirectory here is one code-hosting platform; `scaffold.sh --platform
<name>` renders that platform's template.

## Platform registry

| Platform | Subdirectory | Rendered CI file | Status |
|----------|--------------|------------------|--------|
| aoneci | `aoneci/` | `.aoneci/deploy-pages.yaml` | implemented |
| github | `github/` | `.github/workflows/deploy-pages.yaml` | stub (structure only — see its README) |

## Extension contract (adding a new platform)

1. Create `<platform>/deploy-pages.yaml.tpl` using the shared placeholder
   tokens (`__SITE_NAME__`, `__TITLE__`, `__BRANCH__`, `__IMAGE__`,
   `__DOCS_DIR__`) and the mandatory pipeline semantics (push trigger,
   docs-dir guarded build, unconditional `mkdir -p dist`, publish `dist/`).
   Contract details: `github/README.md` doubles as the reference spec.
2. Register the platform's rendered-file path in `scaffold.sh`
   (`ci_target_for_platform`).
3. Add a row to the registry table above.
4. Optional: a `<platform>/README.md` with platform-specific notes.

Platforms without a `.tpl` file are still scaffoldable: all
platform-independent artifacts are created and a warning directs the user
to author the CI file manually.
