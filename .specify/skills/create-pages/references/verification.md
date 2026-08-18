# Verification — create-pages

Run after scaffolding to prove the generated pipeline works. All checks use
the same Hugo image the CI uses — the `--image` value chosen at scaffold
time (environment-specific; the aoneci default image only pulls in that
environment). The build test itself is platform-independent: it exercises
`docs/scripts/build-docs.sh`, which every platform's CI pipeline invokes.

## 1. Build test

Prefer a volume mount (`<image>` = the `--image` parameter):

```bash
docker run --rm -v "$PWD:/workspace" -w /workspace \
  <image> \
  bash docs/scripts/build-docs.sh
```

**Fallback** (sandbox docker daemons where host mounts are invisible inside
containers — observed in this environment): copy the project in via `docker cp`
(`<ci-dir>` = the platform's CI directory, e.g. `.aoneci` for aoneci):

```bash
docker run -d --name hugo-verify <image> sleep 600
tar czf /tmp/site.tar.gz <ci-dir> docs .gitignore
docker cp /tmp/site.tar.gz hugo-verify:/tmp/
docker exec hugo-verify sh -c 'mkdir -p /w && tar xzf /tmp/site.tar.gz -C /w && cd /w && bash docs/scripts/build-docs.sh'
```

Expected: `Pages > 0`, `Total in N ms`, **no ERROR lines**.

## 2. Output checks

```bash
find dist -name "*.html" | wc -l                       # > 0; ≈ number of .md docs + section pages
test -f dist/hugo.yaml && echo BUG || echo OK          # config must NOT be deployed
grep -o '<title>[^<]*</title>' dist/<some-page>/index.html   # non-empty; extracted from first H1
ls dist/ | grep -E '^(categories|tags)$' && echo BUG || echo OK  # taxonomies disabled
```

## 3. No-docs guard test

Simulate the CI run block exactly (never `bash` the YAML file itself — it is
not executable):

```bash
mv docs docs.bak && rm -rf dist
if [ -d docs ]; then bash docs/scripts/build-docs.sh; fi
mkdir -p dist
ls dist                      # expect: empty directory, exit 0
mv docs.bak docs
```

Expected: no build attempt, empty `dist/` created, exit code 0.

## 4. Cleanup

```bash
docker rm -f hugo-verify   # if the fallback container was used
rm -rf dist                # build artifact; gitignored anyway
```
