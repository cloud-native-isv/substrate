#!/usr/bin/env bash
# scaffold.sh — Scaffold Hugo docs-deployment infrastructure into a target project.
#
# Usage:
#   scaffold.sh --site-name <name> [options]
#
# Options:
#   --site-name <name>   Required. Deploy site name (CI deploy-pages site-name input).
#   --platform <name>    Code-hosting/CI platform (default: aoneci). Templates live in
#                        scripts/ci-templates/<name>/; registry: scripts/ci-templates/README.md.
#   --title <title>      Site title in hugo.yaml (default: "<site-name> 文档").
#   --branch <branch>    Production branch (default: main).
#   --image <image>      Hugo docker image — environment-specific, override per environment
#                        (default: reg.docker.alibaba-inc.com/xuanji-images/hugo:latest).
#   --docs-dir <dir>     Docs directory name relative to project root (default: docs).
#   --root <dir>         Target project root (default: current directory).
#   --force              Overwrite files that already exist.
#   --help               Show this help.
#
# Creates (platform-independent):
#   <docs-dir>/hugo.yaml
#   <docs-dir>/layouts/index.html
#   <docs-dir>/layouts/_default/single.html
#   <docs-dir>/layouts/_default/list.html
#   <docs-dir>/layouts/partials/title.html
#   <docs-dir>/scripts/build-docs.sh            (chmod +x)
# Creates (platform-specific, per --platform):
#   CI pipeline rendered from scripts/ci-templates/<platform>/deploy-pages.yaml.tpl
#   (aoneci: .aoneci/deploy-pages.yaml). Platforms without a template emit a
#   warning and scaffold the platform-independent artifacts only.
# Appends `.hugo-content/` to .gitignore (idempotent).
# Output: JSON summary on stdout: {"created": [...], "skipped": [...], "warnings": [...]}.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/ci-templates"

SITE_NAME=""
TITLE=""
BRANCH="main"
PLATFORM="aoneci"
IMAGE="reg.docker.alibaba-inc.com/xuanji-images/hugo:latest"
DOCS_DIR="docs"
ROOT="$(pwd)"
FORCE=0

usage() { awk 'NR > 1 && !/^#/ {exit} NR > 1 {print}' "$0" | sed 's/^# \{0,1\}//'; }

while [ $# -gt 0 ]; do
  case "$1" in
    --site-name) SITE_NAME="$2"; shift 2 ;;
    --platform)  PLATFORM="$2"; shift 2 ;;
    --title)     TITLE="$2"; shift 2 ;;
    --branch)    BRANCH="$2"; shift 2 ;;
    --image)     IMAGE="$2"; shift 2 ;;
    --docs-dir)  DOCS_DIR="$2"; shift 2 ;;
    --root)      ROOT="$2"; shift 2 ;;
    --force)     FORCE=1; shift ;;
    --help|-h)   usage; exit 0 ;;
    *) echo "ERROR: unknown option: $1" >&2; exit 2 ;;
  esac
done

[ -n "$SITE_NAME" ] || { echo "ERROR: --site-name is required (see --help)" >&2; exit 2; }
[ -n "$TITLE" ] || TITLE="$SITE_NAME 文档"
[ -d "$ROOT" ] || { echo "ERROR: --root directory does not exist: $ROOT" >&2; exit 2; }
[ -d "$TEMPLATES_DIR/$PLATFORM" ] || {
  KNOWN=$(cd "$TEMPLATES_DIR" 2>/dev/null && ls -d */ 2>/dev/null | tr -d '/' | tr '\n' ' ')
  echo "ERROR: unknown --platform '$PLATFORM' (known: ${KNOWN:-none}; see scripts/ci-templates/README.md)" >&2
  exit 2
}

ci_target_for_platform() {
  case "$1" in
    aoneci) printf '%s' ".aoneci/deploy-pages.yaml" ;;
    github) printf '%s' ".github/workflows/deploy-pages.yaml" ;;
    *)      printf '%s' "" ;;
  esac
}

CREATED=()
SKIPPED=()
WARNINGS=()

json_list() {
  local out="" item
  for item in "$@"; do
    [ -n "$out" ] && out="$out, "
    out="$out\"$item\""
  done
  printf '[%s]' "$out"
}

# write_file <path> <content> — write content unless path exists (skip without --force).
write_file() {
  local path="$1" content="$2"
  if [ -e "$path" ] && [ "$FORCE" -ne 1 ]; then
    SKIPPED+=("$path")
    return 0
  fi
  mkdir -p "$(dirname "$path")"
  printf '%s\n' "$content" > "$path"
  CREATED+=("$path")
}

sub() { # substitute placeholders in $1 (pure bash — no sed escaping issues)
  local s="$1"
  s="${s//__SITE_NAME__/$SITE_NAME}"
  s="${s//__TITLE__/$TITLE}"
  s="${s//__BRANCH__/$BRANCH}"
  s="${s//__IMAGE__/$IMAGE}"
  s="${s//__DOCS_DIR__/$DOCS_DIR}"
  printf '%s' "$s"
}

cd "$ROOT"

if [ ! -d "$DOCS_DIR" ]; then
  WARNINGS+=("$DOCS_DIR/ does not exist — scaffolding proceeds (CI guard handles absence), but add docs content before expecting a real site")
fi

# ---------------------------------------------------------------- CI pipeline
CI_TPL="$TEMPLATES_DIR/$PLATFORM/deploy-pages.yaml.tpl"
CI_TARGET="$(ci_target_for_platform "$PLATFORM")"
if [ -f "$CI_TPL" ] && [ -n "$CI_TARGET" ]; then
  CI_YAML=$(sub "$(cat "$CI_TPL")")
  write_file "$CI_TARGET" "$CI_YAML"
elif [ -f "$CI_TPL" ]; then
  WARNINGS+=("platform '$PLATFORM' has a template but no registered target path in scaffold.sh — CI file not written")
else
  WARNINGS+=("platform '$PLATFORM' has no template yet (see scripts/ci-templates/$PLATFORM/README.md) — scaffolded platform-independent artifacts only; author the CI file manually")
fi

# ---------------------------------------------------------------- hugo config
HUGO_YAML=$(sub "$(cat <<'EOF'
baseURL: /
locale: zh-CN
title: __TITLE__
publishDir: dist
disableKinds: [taxonomy, term]
markup:
  goldmark:
    renderer:
      unsafe: true
EOF
)")
write_file "$DOCS_DIR/hugo.yaml" "$HUGO_YAML"

# ---------------------------------------------------------------- layouts
TITLE_PARTIAL=$(cat <<'EOF'
{{- $title := .Title -}}
{{- if not $title -}}
  {{- with findRE `<h1[^>]*>(.*?)</h1>` .Content 1 -}}
    {{- $title = index . 0 | replaceRE `<[^>]+>` "" | plainify -}}
  {{- end -}}
{{- end -}}
{{- $title -}}
EOF
)
write_file "$DOCS_DIR/layouts/partials/title.html" "$TITLE_PARTIAL"

INDEX_HTML=$(cat <<'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ partial "title.html" . }} | {{ .Site.Title }}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 1.5rem; line-height: 1.6; color: #222; }
    h1 { border-bottom: 1px solid #ddd; padding-bottom: .5rem; }
    a { color: #0366d6; text-decoration: none; }
    a:hover { text-decoration: underline; }
    ul { padding-left: 1.2rem; }
    .section-list li { margin: .75rem 0; font-size: 1.1rem; }
  </style>
</head>
<body>
  <h1>{{ .Site.Title }}</h1>
  {{ .Content }}
  <ul class="section-list">
    {{ range .Pages.ByTitle }}
    <li><a href="{{ .RelPermalink }}">{{ partial "title.html" . }}</a></li>
    {{ end }}
  </ul>
</body>
</html>
EOF
)
write_file "$DOCS_DIR/layouts/index.html" "$INDEX_HTML"

SINGLE_HTML=$(cat <<'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ partial "title.html" . }} | {{ .Site.Title }}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 1.5rem; line-height: 1.6; color: #222; }
    h1 { border-bottom: 1px solid #ddd; padding-bottom: .5rem; }
    a { color: #0366d6; text-decoration: none; }
    a:hover { text-decoration: underline; }
    blockquote { border-left: 3px solid #ddd; margin: 0; padding: .5rem 1rem; color: #555; background: #f8f8f8; }
    code { background: #f0f0f0; padding: .15rem .4rem; border-radius: 3px; font-size: .9em; }
    pre code { display: block; padding: 1rem; overflow-x: auto; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: .5rem; text-align: left; }
    th { background: #f5f5f5; }
    img { max-width: 100%; }
    .nav { margin-bottom: 1rem; font-size: .9rem; color: #666; }
    .nav a { color: #666; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="{{ "" | relURL }}">首页</a>
    {{ if .Parent }}
    {{ if not .Parent.IsHome }} > <a href="{{ .Parent.RelPermalink }}">{{ partial "title.html" .Parent }}</a>{{ end }}
    {{ end }}
    > {{ partial "title.html" . }}
  </div>
  <article>
    <h1>{{ partial "title.html" . }}</h1>
    {{ .Content }}
  </article>
</body>
</html>
EOF
)
write_file "$DOCS_DIR/layouts/_default/single.html" "$SINGLE_HTML"

LIST_HTML=$(cat <<'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ partial "title.html" . }} | {{ .Site.Title }}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 1.5rem; line-height: 1.6; color: #222; }
    h1 { border-bottom: 1px solid #ddd; padding-bottom: .5rem; }
    a { color: #0366d6; text-decoration: none; }
    a:hover { text-decoration: underline; }
    blockquote { border-left: 3px solid #ddd; margin: 0; padding: .5rem 1rem; color: #555; background: #f8f8f8; }
    ul { padding-left: 1.2rem; }
    .nav { margin-bottom: 1rem; font-size: .9rem; color: #666; }
    .section-list li { margin: .5rem 0; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="{{ "" | relURL }}">首页</a>
    {{ if not .IsHome }} > {{ partial "title.html" . }}{{ end }}
  </div>
  <article>
    <h1>{{ partial "title.html" . }}</h1>
    {{ .Content }}
    {{ if .Pages }}
    <ul class="section-list">
      {{ range .Pages.ByTitle }}
      <li><a href="{{ .RelPermalink }}">{{ partial "title.html" . }}</a></li>
      {{ end }}
    </ul>
    {{ end }}
  </article>
</body>
</html>
EOF
)
write_file "$DOCS_DIR/layouts/_default/list.html" "$LIST_HTML"

# ---------------------------------------------------------------- build script
BUILD_SH=$(cat <<'EOF'
#!/usr/bin/env bash
# Build docs site with Hugo.
# Designed to be called from docs/ as its content root.
# Output goes to <project-root>/dist/ (matching .aoneci/deploy-pages.yaml deploy-dir).
set -euo pipefail

DOCS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="$(cd "$DOCS_DIR/.." && pwd)"

cd "$DOCS_DIR"

# Stage content: copy everything except infrastructure files, then rename
# index.md -> _index.md. Hugo treats index.md as a leaf bundle which hides
# sibling .md files; _index.md is the section-page semantics we need.
rm -rf .hugo-content
mkdir .hugo-content
tar cf - \
  --exclude=.hugo-content \
  --exclude=layouts \
  --exclude=scripts \
  --exclude=hugo.yaml \
  . | tar xf - -C .hugo-content

find .hugo-content -name index.md -exec sh -c \
  'mv "$1" "${1%/index.md}/_index.md"' _ {} \;

# Build site. Output goes to project-root/dist/.
hugo --contentDir .hugo-content --destination "$PROJECT_ROOT/dist"
EOF
)
write_file "$DOCS_DIR/scripts/build-docs.sh" "$BUILD_SH"
[ -f "$DOCS_DIR/scripts/build-docs.sh" ] && chmod +x "$DOCS_DIR/scripts/build-docs.sh"

# ---------------------------------------------------------------- .gitignore
GITIGNORE_NOTE="# Hugo docs build staging (added by create-pages)"
if [ -f .gitignore ] && grep -qxF '.hugo-content/' .gitignore; then
  SKIPPED+=(".gitignore (already ignores .hugo-content/)")
else
  {
    [ -f .gitignore ] && echo ""
    echo "$GITIGNORE_NOTE"
    echo ".hugo-content/"
  } >> .gitignore
  CREATED+=(".gitignore (appended .hugo-content/)")
fi

printf '{"created": %s, "skipped": %s, "warnings": %s}\n' \
  "$(json_list "${CREATED[@]+"${CREATED[@]}"}")" \
  "$(json_list "${SKIPPED[@]+"${SKIPPED[@]}"}")" \
  "$(json_list "${WARNINGS[@]+"${WARNINGS[@]}"}")"
