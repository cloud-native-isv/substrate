# Verification — create-pages

One section per stage. Run the stage you touched, plus stage 2 whenever
rendering assets changed. Substitute `<docs>` for the docs directory (default
`docs`) and `<image>` for the `--image` value chosen at stage 3 (the aoneci
default image only pulls inside that environment).

## Stage 1 — 本地文档库 containment

```bash
find <docs> -name '*.md' | wc -l          # > 0; the library has content
git status --short | grep -v '^.. <docs>/' # only the platform CI file may appear
```

Expected: every path this skill wrote is under `<docs>/`, with exactly one
allowed exception — the rendered CI file (`.aoneci/deploy-pages.yaml` or
`.github/workflows/deploy-pages.yaml`). Any other root-level path is a
containment violation: report it and remove it.

## Stage 2 — Hugo 渲染

```bash
python3 "${SKILL_HOME}/scripts/scaffold-hugo.py" --action check --root .     # drift, no writes
python3 "${SKILL_HOME}/scripts/scaffold-hugo.py" --action scaffold --root . --site-title "<t>"
python3 "${SKILL_HOME}/scripts/scaffold-hugo.py" --action scaffold --root . # idempotence: all `unchanged`
cd <docs> && hugo --logLevel warn                                          # build + surface warnings
```

Expected:

- `Pages > 0` and **no ERROR lines**; page count ≈ number of `.md` documents
  plus section pages.
- The second `scaffold` run reports every file `unchanged` and writes nothing
  (zero churn). Files you edited locally are reported `kept`.
- Exactly one deprecation warning is currently expected —
  `module.mounts.excludeFiles` (see design-rationale). Any *other* deprecation
  or warning is a finding.

Output checks:

```bash
find <docs>/public -name '*.html' | wc -l                            # > 0
test -f <docs>/public/hugo.toml && echo BUG || echo OK               # config must not be published
grep -o '<title>[^<]*</title>' <docs>/public/<some-page>/index.html  # non-empty on both sides of '·'
grep -c '<a href="[^"]*"></a>' <docs>/public/index.html || true      # 0; no blank link text
ls <docs>/public | grep -E '^(categories|tags)$' && echo BUG || echo OK
test -d dist && echo BUG || echo OK                                  # no repository-root output
```

A blank `<title>` (`· Site`) or blank link text means the title fallback partial
is missing or not wired into the layout — that regression is what
`partials/title.html` exists to prevent.

## Stage 3 — Pages 服务

### local

```bash
cd <docs> && hugo serve --port 1313 &
curl -sf http://localhost:1313/ > /dev/null && echo OK
kill %1
```

Expected: HTTP 200 on the home page and live reload on edit. Nothing is written
to disk by this target.

### Hosting platform (aoneci / github)

```bash
bash "${SKILL_HOME}/scripts/scaffold-ci.sh" --site-name <name> --platform aoneci
python3 -c "import yaml;d=yaml.safe_load(open('.aoneci/deploy-pages.yaml'));print(d['jobs']['deploy']['steps'][2]['inputs']['deploy-dir'])"
```

Expected: JSON reports the CI file `created` (or `skipped` when it already
exists — never silently overwritten), the file parses as YAML, and `deploy-dir`
is `<docs>/public/`. A platform without a template (`github` today) must report a
warning and create nothing.

Container build test — proves the pipeline's build step works in the CI image:

```bash
docker run --rm -v "$PWD:/workspace" -w /workspace <image> \
  sh -c 'if [ -d <docs> ]; then (cd <docs> && hugo --minify); fi; mkdir -p <docs>/public; ls <docs>/public | head'
```

**Fallback** for sandbox docker daemons where host mounts are invisible inside
containers (observed in this environment):

```bash
docker run -d --name hugo-verify <image> sleep 600
tar czf /tmp/site.tar.gz <docs>
docker cp /tmp/site.tar.gz hugo-verify:/tmp/
docker exec hugo-verify sh -c 'mkdir -p /w && tar xzf /tmp/site.tar.gz -C /w && cd /w/<docs> && hugo --minify'
```

### No-docs guard test

Simulate the CI run block exactly (never `bash` the YAML file itself — it is not
executable):

```bash
mv <docs> <docs>.bak
if [ -d <docs> ]; then (cd <docs> && hugo --minify); fi
mkdir -p <docs>/public
ls <docs>/public          # expect: empty directory, exit 0
rm -rf <docs> && mv <docs>.bak <docs>
```

Expected: no build attempt, an empty publish directory created, exit code 0.

## Cleanup

```bash
docker rm -f hugo-verify        # if the fallback container was used
rm -rf <docs>/public            # build artifact; gitignored by the scaffold
```
