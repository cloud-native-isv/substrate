#!/usr/bin/env python3
"""Feedback-as-files engine for Spec Kit.

Persists local, unit-scoped feedback entries produced at the wrap-up of a
*qualifying* flow (every skill; complex commands only) into the project
feedback store, using plain Markdown files plus a lightweight local JSON index
(no vector store). Patterned on ``memory-utils.py``.

Layout (relative to workspace root):

    .specify/memory/feedback/
        <created-ts>-<unit-slug>.md   # one file per recorded run
        index.json                    # store metadata + entry mirror
        .gitkeep                      # already present

An entry's ``unit_id`` must name a Spec Kit command or skill
(``/speckit.<command>`` | ``skill:<name>``); ``record`` rejects any other id.
Each entry is local-scoped (``scope: local``) and stays strictly distinct from
the global ``/speckit.review`` report.

Positioning: feedback targets the Spec Kit framework itself (templates,
commands, skills, scripts, docs) — never the LLM, agent CLI, harness, or the
user's project code. Entries are user data: recording and processing are fully
optional and ignorable. This engine performs **no network operations of any
kind**; ``package`` only produces a local zip that the user may send manually,
and ``mark-submitted`` archives the pending batch into ``packages/`` (with an
optional disposition note, ``--notes``) before resetting the local counter
("user confirmed disposition", NOT "uploaded") — a reset therefore always
leaves an auditable package artifact behind.

Identifier discipline: ``--feature`` carries the **requirement key** (e.g.
``038-goal-target``); ``--feature-id`` carries the **Feature registry ID**
(e.g. ``041``). They are different number spaces — never overload one field
with both.
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
import zipfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

FEEDBACK_SUBDIR = Path(".specify") / "memory" / "feedback"
PACKAGES_DIRNAME = "packages"
INDEX_NAME = "index.json"
STORE_NAME = "feedback"
DEFAULT_THRESHOLD = 10
NO_OP_POINT = "No significant optimization points identified this run."
DIST_NAME = "specify-cli"

# A unit id is valid only when it names a Spec Kit command or skill.
_UNIT_ID_RE = re.compile(r"^(?:/speckit\.[a-z0-9._-]+|skill:[a-z0-9._-]+)$")
_SLUG_RE = re.compile(r"[^a-z0-9]+")


class FeedbackError(ValueError):
    """Raised for user-facing validation failures (exit code 2)."""


# --------------------------------------------------------------------------- #
# Helpers (pure)
# --------------------------------------------------------------------------- #
def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def timestamp_id() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")


def validate_unit_id(unit_id: str) -> bool:
    return bool(unit_id) and bool(_UNIT_ID_RE.match(unit_id.strip()))


def unit_slug(unit_id: str) -> str:
    """`/speckit.plan` -> `speckit-plan`; `skill:study-project` -> `skill-study-project`."""
    slug = _SLUG_RE.sub("-", (unit_id or "").strip().lower()).strip("-")
    return slug or "unit"


def make_summary(review: str) -> str:
    for line in (review or "").splitlines():
        stripped = line.strip()
        if stripped:
            return stripped[:200]
    return ""


def count_since_submission(entries: List[Dict[str, Any]], submitted_at: Optional[str]) -> int:
    """Number of entries recorded since the last submission (all when never submitted)."""
    if not submitted_at:
        return len(entries)
    return sum(1 for e in entries if str(e.get("created", "")) > submitted_at)


def should_prompt(count: int, threshold: int) -> bool:
    return count >= threshold


def resolve_threshold(explicit: Optional[int], stored: Optional[int]) -> int:
    if explicit is not None:
        return explicit
    env = os.environ.get("SPECKIT_FEEDBACK_THRESHOLD")
    if env:
        try:
            return int(env)
        except ValueError:
            pass
    if stored is not None:
        return stored
    return DEFAULT_THRESHOLD


def resolve_workspace_root(explicit: Optional[str]) -> Path:
    """Locate the project workspace root that owns the ``.specify/`` store.

    Priority: explicit CLI argument > script self-location (an engine copy
    installed under ``*/.specify/scripts/`` anchors its parent project) >
    nearest CWD ancestor containing ``.specify/`` > CWD itself. Self-location
    must outrank the walk-up: when the agent's CWD sits inside a skill
    directory that contains a stray nested ``.specify/`` (created by an
    earlier bug), the walk-up would capture that nested tree and split the
    store. Falling back to bare CWD is only a last resort outside any project.
    """
    if explicit:
        return Path(explicit).resolve()
    script = Path(__file__).resolve()
    parts = script.parts
    for i, part in enumerate(parts):
        if part == ".specify" and i + 1 < len(parts) and parts[i + 1] == "scripts":
            return Path(*parts[:i])
    cwd = Path.cwd().resolve()
    for candidate in (cwd, *cwd.parents):
        if (candidate / ".specify").is_dir():
            return candidate
    return cwd


# --------------------------------------------------------------------------- #
# Input reading
# --------------------------------------------------------------------------- #
def read_review(args: argparse.Namespace) -> str:
    if getattr(args, "review", None) is not None:
        return args.review
    if getattr(args, "review_file", None):
        return Path(args.review_file).read_text(encoding="utf-8")
    return ""


def read_points(args: argparse.Namespace) -> List[str]:
    raw = ""
    if getattr(args, "points", None) is not None:
        raw = args.points
    elif getattr(args, "points_file", None):
        raw = Path(args.points_file).read_text(encoding="utf-8")
    points = [line.strip().lstrip("-").strip() for line in raw.splitlines()]
    return [p for p in points if p]


# --------------------------------------------------------------------------- #
# Frontmatter (minimal, dependency-free)
# --------------------------------------------------------------------------- #
def _scalar(value: str) -> Any:
    try:
        return json.loads(value)
    except (ValueError, TypeError):
        return value.strip().strip('"')


def dump_frontmatter(meta: Dict[str, Any]) -> str:
    order = ["id", "unit_id", "unit_type", "run_id", "scope",
             "probe", "kind", "slice",
             "feature", "feature_id", "disposition", "migrated_from", "partial", "created", "summary"]
    lines = ["---"]
    for key in order:
        if key not in meta:
            continue
        value = meta[key]
        if value is None or value == "":
            if key not in ("partial",):
                continue
        lines.append(f"{key}: {json.dumps(value, ensure_ascii=False)}")
    lines.append("---")
    return "\n".join(lines)


def parse_frontmatter(text: str) -> Tuple[Dict[str, Any], str]:
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return {}, text
    meta: Dict[str, Any] = {}
    idx = 1
    while idx < len(lines) and lines[idx].strip() != "---":
        line = lines[idx]
        if ":" in line:
            key, _, raw = line.partition(":")
            meta[key.strip()] = _scalar(raw.strip())
        idx += 1
    body = "\n".join(lines[idx + 1:]) if idx < len(lines) else ""
    return meta, body.lstrip("\n")


def compose_entry(meta: Dict[str, Any], review: str, points: List[str], partial: bool) -> str:
    review_body = review.strip()
    if partial and not review_body.startswith("**Partial run**"):
        review_body = f"**Partial run** — {review_body}"
    bullet_lines = "\n".join(f"- {p}" for p in points)
    body = f"## Review\n{review_body}\n\n## Optimization Points\n{bullet_lines}"
    return dump_frontmatter(meta) + "\n\n" + body.strip() + "\n"


# --------------------------------------------------------------------------- #
# Store / index
# --------------------------------------------------------------------------- #
def feedback_dir(workspace_root: Path) -> Path:
    return Path(workspace_root).resolve() / FEEDBACK_SUBDIR


def ensure_feedback_dir(workspace_root: Path) -> Path:
    target = feedback_dir(workspace_root)
    target.mkdir(parents=True, exist_ok=True)
    return target


def index_path(workspace_root: Path) -> Path:
    return feedback_dir(workspace_root) / INDEX_NAME


def empty_index() -> Dict[str, Any]:
    return {
        "store": STORE_NAME,
        "updated": None,
        "threshold": DEFAULT_THRESHOLD,
        "count_since_submission": 0,
        "submitted_at": None,
        "entries": [],
    }


def load_index(workspace_root: Path) -> Dict[str, Any]:
    path = index_path(workspace_root)
    if not path.exists():
        return empty_index()
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (ValueError, OSError):
        return empty_index()
    if not isinstance(data, dict) or not isinstance(data.get("entries"), list):
        return empty_index()
    base = empty_index()
    base.update(data)
    return base


def save_index(workspace_root: Path, index: Dict[str, Any]) -> None:
    index["entries"] = sorted(
        index.get("entries", []), key=lambda e: e.get("created", ""), reverse=True
    )
    index["store"] = STORE_NAME
    index["updated"] = now_iso()
    ensure_feedback_dir(workspace_root)
    index_path(workspace_root).write_text(
        json.dumps(index, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


def entry_meta(meta: Dict[str, Any], filename: str) -> Dict[str, Any]:
    return {
        "id": meta.get("id", ""),
        "file": filename,
        "unit_id": meta.get("unit_id", ""),
        "unit_type": meta.get("unit_type", ""),
        "run_id": meta.get("run_id", ""),
        "probe": meta.get("probe", ""),
        "kind": meta.get("kind", ""),
        "slice": meta.get("slice", ""),
        "feature": meta.get("feature", ""),
        "feature_id": meta.get("feature_id", ""),
        "disposition": meta.get("disposition", ""),
        "partial": bool(meta.get("partial", False)),
        "created": meta.get("created", ""),
        "summary": meta.get("summary", ""),
    }


# --------------------------------------------------------------------------- #
# Actions
# --------------------------------------------------------------------------- #
def action_record(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    unit_id = (args.unit_id or "").strip()
    unit_type = (args.unit_type or "").strip()
    if unit_id.startswith("custom:"):
        if not _CUSTOM_UNIT_RE.match(unit_id):
            raise FeedbackError(
                "Invalid --unit-id; expected 'custom:<owner>/<name>'.")
        if unit_type != "custom-unit":
            raise FeedbackError("--unit-type must be 'custom-unit' for custom: units.")
    else:
        if not validate_unit_id(unit_id):
            raise FeedbackError(
                "Invalid --unit-id; expected '/speckit.<command>' or 'skill:<name>'."
            )
        if unit_type == "custom-unit":
            raise FeedbackError(
                "--unit-type 'custom-unit' requires a 'custom:' --unit-id.")
        if unit_type not in ("skill", "command"):
            raise FeedbackError("--unit-type must be 'skill' or 'command'.")
    run_id = (args.run_id or "").strip()
    if not run_id:
        raise FeedbackError("--run-id is required.")

    review = read_review(args)
    if not review.strip():
        raise FeedbackError("record requires a non-empty --review / --review-file.")
    points = read_points(args)
    if not points:
        raise FeedbackError(
            "record requires --points / --points-file with at least one line "
            f"(use '{NO_OP_POINT}' for a clean run)."
        )

    # Probe resolution (requirement 041): with the registry installed, every
    # entry MUST carry probe attribution (engine-resolved; never caller-set).
    # Registry absent (un-upgraded workspace) → legacy record without fields.
    registry = load_probe_registry(workspace_root)
    probe = None
    if registry["available"]:
        lifecycle_point = (getattr(args, "lifecycle_point", None) or "").strip() or None
        probe = resolve_probe(registry, unit_id, lifecycle_point)
        if probe is None:
            if lifecycle_point:
                raise FeedbackError(
                    f"no probe object for unit: {unit_id} "
                    f"with lifecycle_point: {lifecycle_point}"
                )
            raise FeedbackError(f"no probe object for unit: {unit_id}")

    index = load_index(workspace_root)
    threshold = resolve_threshold(args.threshold, index.get("threshold"))
    index["threshold"] = threshold
    entries = index["entries"]

    existing = next(
        (e for e in entries if e.get("unit_id") == unit_id and e.get("run_id") == run_id),
        None,
    )
    if existing:
        rel = (FEEDBACK_SUBDIR / existing["file"]).as_posix()
        count = index.get("count_since_submission", 0)
        return {
            "id": existing.get("id", ""),
            "path": rel,
            "duplicate": True,
            "count_since_submission": count,
            "threshold": threshold,
            "should_prompt": should_prompt(count, threshold),
        }

    created = now_iso()
    slug = unit_slug(unit_id)
    entry_id = f"{timestamp_id()}-{slug}"
    target_dir = ensure_feedback_dir(workspace_root)
    filename = f"{entry_id}.md"
    counter = 1
    while (target_dir / filename).exists():
        filename = f"{entry_id}-{counter}.md"
        counter += 1
    entry_id = filename[:-3]

    partial = bool(args.partial)
    meta = {
        "id": entry_id,
        "unit_id": unit_id,
        "unit_type": unit_type,
        "run_id": run_id,
        "scope": "local",
        "feature": (args.feature or "").strip(),
        "feature_id": (getattr(args, "feature_id", "") or "").strip(),
        "partial": partial,
        "created": created,
        "summary": make_summary(review),
    }
    if probe is not None:
        meta["probe"] = probe["object_id"]
        meta["kind"] = probe["kind"]
        meta["slice"] = probe["slice"]
    (target_dir / filename).write_text(
        compose_entry(meta, review, points, partial), encoding="utf-8"
    )

    entries.append(entry_meta(meta, filename))
    index["count_since_submission"] = index.get("count_since_submission", 0) + 1
    save_index(workspace_root, index)

    count = index["count_since_submission"]
    rel = (FEEDBACK_SUBDIR / filename).as_posix()
    return {
        "id": entry_id,
        "path": rel,
        "duplicate": False,
        "count_since_submission": count,
        "threshold": threshold,
        "should_prompt": should_prompt(count, threshold),
    }


def action_status(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    index = load_index(workspace_root)
    threshold = resolve_threshold(args.threshold, index.get("threshold"))
    count = index.get("count_since_submission", 0)
    entries = index.get("entries", [])
    return {
        "count_since_submission": count,
        "threshold": threshold,
        "should_prompt": should_prompt(count, threshold),
        "total_entries": len(entries),
        "submitted_at": index.get("submitted_at"),
        "legacy_remaining": sum(1 for e in entries if not e.get("probe")),
        "external_count": sum(1 for e in entries if e.get("kind") == "external"),
    }


def action_list(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    limit = None if args.limit == 0 else max(1, args.limit)
    unit_id = (args.unit_id or "").strip()
    unit_type = (args.unit_type or "").strip()
    since = (args.since or "").strip()
    contains = (args.contains or "").strip().lower()
    slice_filter = (getattr(args, "slice", None) or "").strip()
    kind_filter = (getattr(args, "kind", None) or "").strip()
    disposition = (getattr(args, "disposition", None) or "").strip()

    items: List[Dict[str, Any]] = []
    for entry in load_index(workspace_root).get("entries", []):
        if unit_id and entry.get("unit_id") != unit_id:
            continue
        if unit_type and entry.get("unit_type") != unit_type:
            continue
        if since and str(entry.get("created", "")) < since:
            continue
        if slice_filter and entry.get("slice", "") != slice_filter:
            continue
        if kind_filter and entry.get("kind", "") != kind_filter:
            continue
        if disposition:
            current = entry.get("disposition", "") or "open"
            if current != disposition:
                continue
        if contains:
            haystack = str(entry.get("summary", ""))
            entry_file = feedback_dir(workspace_root) / entry.get("file", "")
            if entry_file.is_file():
                haystack += "\n" + entry_file.read_text(encoding="utf-8")
            if contains not in haystack.lower():
                continue
        item = dict(entry)
        item["path"] = (FEEDBACK_SUBDIR / entry["file"]).as_posix()
        items.append(item)
    items.sort(key=lambda e: e.get("created", ""), reverse=True)
    matches = items if limit is None else items[:limit]
    return {"count": len(matches), "matches": matches}


_VALID_DISPOSITIONS = ("processed", "ignored")


def action_dispose(args: argparse.Namespace) -> Dict[str, Any]:
    """Mark one entry's disposition (local metadata; body never rewritten)."""
    workspace_root = resolve_workspace_root(args.workspace_root)
    target_state = (getattr(args, "to", None) or "").strip()
    if target_state not in _VALID_DISPOSITIONS:
        raise FeedbackError(
            "--to must be one of " + " | ".join(_VALID_DISPOSITIONS) + ".")
    entry_id = (getattr(args, "id", None) or "").strip()
    if not entry_id:
        raise FeedbackError("--id is required.")
    index = load_index(workspace_root)
    for entry in index.get("entries", []):
        if entry.get("id") != entry_id:
            continue
        entry["disposition"] = target_state
        entry_file = feedback_dir(workspace_root) / entry.get("file", "")
        if entry_file.is_file():
            meta, body = parse_frontmatter(entry_file.read_text(encoding="utf-8"))
            meta["disposition"] = target_state
            entry_file.write_text(
                dump_frontmatter(meta) + "\n\n" + body.strip() + "\n",
                encoding="utf-8",
            )
        save_index(workspace_root, index)
        return {"id": entry_id, "disposition": target_state}
    raise FeedbackError(f"no entry with id: {entry_id}")


def action_mark_submitted(args: argparse.Namespace) -> Dict[str, Any]:
    """Archive the pending batch, then reset the local counter.

    Every reset leaves an auditable package artifact behind (F2): the entries
    accumulated since the last submission are zipped into ``packages/`` before
    the counter resets. An optional ``--notes`` disposition summary is stored
    inside the archive as ``SUBMISSION-NOTES.md`` (F3). This is purely local
    bookkeeping — it does NOT upload or transmit anything.
    """
    workspace_root = resolve_workspace_root(args.workspace_root)
    index = load_index(workspace_root)
    reset_from = index.get("count_since_submission", 0)

    submitted_at = index.get("submitted_at")
    entries = index.get("entries", [])
    if submitted_at:
        selected = [e for e in entries if str(e.get("created", "")) > submitted_at]
    else:
        selected = list(entries)
    selected.sort(key=lambda e: e.get("created", ""))
    package = write_package(workspace_root, index, selected,
                            notes=getattr(args, "notes", None))

    new_submitted_at = now_iso()
    index["count_since_submission"] = 0
    index["submitted_at"] = new_submitted_at
    save_index(workspace_root, index)
    return {
        "submitted_at": new_submitted_at,
        "reset_from": reset_from,
        "packaged": package.get("packaged", 0),
        "zip": package.get("zip"),
    }


def action_reindex(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    prior = load_index(workspace_root)
    submitted_at = prior.get("submitted_at")
    threshold = resolve_threshold(args.threshold, prior.get("threshold"))

    target = feedback_dir(workspace_root)
    entries: List[Dict[str, Any]] = []
    if target.exists():
        for path in sorted(target.glob("*.md")):
            meta, _ = parse_frontmatter(path.read_text(encoding="utf-8"))
            if not meta:
                continue
            entries.append(entry_meta(meta, path.name))

    index = empty_index()
    index["threshold"] = threshold
    index["submitted_at"] = submitted_at
    if prior.get("upstream_repo"):
        index["upstream_repo"] = prior["upstream_repo"]
    index["entries"] = entries
    index["count_since_submission"] = count_since_submission(entries, submitted_at)
    save_index(workspace_root, index)
    return {"reindexed": len(entries)}


# --------------------------------------------------------------------------- #
# Upstream detection / packaging (no network operations — red line)
# --------------------------------------------------------------------------- #
def _speckit_version() -> str:
    try:
        from importlib import metadata
        return metadata.version(DIST_NAME)
    except Exception:
        return "unknown"


def detect_upstream(index: Dict[str, Any]) -> Dict[str, Any]:
    """Resolve the upstream repo URL for manual feedback delivery.

    Priority: user-configured ``upstream_repo`` in index.json > PEP 610
    ``direct_url.json`` install metadata (records the git URL the custom
    spec-kit build was installed from) > none (user must ``--set``).
    Detection only reads local files; it never touches the network.
    """
    configured = (index.get("upstream_repo") or "").strip()
    if configured:
        return {"url": configured, "source": "configured", "commit": None}
    try:
        from importlib import metadata
        raw = metadata.distribution(DIST_NAME).read_text("direct_url.json")
        if raw:
            data = json.loads(raw)
            url = (data.get("url") or "").strip()
            vcs = data.get("vcs_info") or {}
            if url and vcs.get("vcs") == "git":
                return {
                    "url": url,
                    "source": "install-metadata",
                    "commit": vcs.get("commit_id"),
                }
    except Exception:
        pass
    return {"url": None, "source": None, "commit": None}


def _send_guidance(upstream: Dict[str, Any]) -> List[str]:
    url = upstream.get("url")
    if not url:
        return [
            "Upstream repo unknown — set it once via: "
            "--action upstream --set <repo-url>",
            "Then send the zip manually (issue attachment or MR); "
            "this engine never sends anything itself.",
        ]
    host_kind = "GitHub" if "github" in url.lower() else "GitLab"
    if host_kind == "GitHub":
        how = "open an issue on the upstream repo and attach the zip"
    else:
        how = ("open an issue and attach the zip, or submit an MR adding it "
               "under the repo's feedback intake directory")
    return [
        f"Send the zip manually to the upstream repo ({host_kind}): {url}",
        f"Suggested: {how}.",
        "Sending is entirely manual and optional — this engine performs no "
        "network operations.",
    ]


def action_upstream(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    index = load_index(workspace_root)
    set_url = (args.set_url or "").strip()
    if set_url:
        index["upstream_repo"] = set_url
        save_index(workspace_root, index)
    return detect_upstream(index)


def write_package(
    workspace_root: Path,
    index: Dict[str, Any],
    selected: List[Dict[str, Any]],
    notes: Optional[str] = None,
) -> Dict[str, Any]:
    """Zip ``selected`` entries into ``packages/``; source files are never touched.

    When ``notes`` is given, a ``SUBMISSION-NOTES.md`` disposition record is
    added to the archive so every counter reset leaves auditable context.
    """
    upstream = detect_upstream(index)
    if not selected:
        return {"packaged": 0, "zip": None, "missing": [], "upstream": upstream,
                "note": "No feedback entries to package."}

    store_dir = feedback_dir(workspace_root)
    packages_dir = store_dir / PACKAGES_DIRNAME
    packages_dir.mkdir(parents=True, exist_ok=True)
    zip_name = f"feedback-{timestamp_id()}.zip"
    zip_path = packages_dir / zip_name
    counter = 1
    while zip_path.exists():
        zip_path = packages_dir / f"feedback-{timestamp_id()}-{counter}.zip"
        counter += 1

    manifest_lines = [
        "# Feedback Package Manifest",
        "",
        "> This feedback targets the Spec Kit framework itself (templates, "
        "commands, skills, scripts, docs) — not the LLM, agent CLI, harness, "
        "or any user project code.",
        "",
        f"- **Generated**: {now_iso()}",
        f"- **Entries**: {len(selected)}",
        f"- **Time range**: {selected[0].get('created', '-')} → "
        f"{selected[-1].get('created', '-')}",
        f"- **spec-kit version**: {_speckit_version()}",
        f"- **Install source**: {upstream.get('url') or 'unknown'}"
        + (f" @ {upstream['commit']}" if upstream.get("commit") else ""),
        "",
        "| Created | Unit | Partial | Probe | Slice | Summary |",
        "|---------|------|---------|-------|-------|---------|",
    ]
    missing: List[str] = []
    packaged: List[str] = []
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for entry in selected:
            src = store_dir / entry["file"]
            if not src.exists():
                missing.append(entry["file"])
                continue
            zf.write(src, arcname=entry["file"])  # read-only: sources untouched
            packaged.append(entry["file"])
            summary = str(entry.get("summary", "")).replace("|", "\\|")[:120]
            manifest_lines.append(
                f"| {entry.get('created', '-')} | {entry.get('unit_id', '?')} "
                f"| {entry.get('partial', False)} | {entry.get('probe', '-') or '-'} "
                f"| {entry.get('slice', '-') or '-'} | {summary} |"
            )
        zf.writestr("MANIFEST.md", "\n".join(manifest_lines) + "\n")
        if notes and notes.strip():
            zf.writestr(
                "SUBMISSION-NOTES.md",
                "# Submission Notes (disposition of this batch)\n\n"
                f"- **Recorded**: {now_iso()}\n\n{notes.strip()}\n",
            )

    rel_zip = zip_path.relative_to(workspace_root).as_posix()
    return {"packaged": len(packaged), "zip": rel_zip, "missing": missing,
            "upstream": upstream}


def action_package(args: argparse.Namespace) -> Dict[str, Any]:
    """Zip pending entries for manual delivery. Source files are never touched.

    External entries (``kind: external``) are host-project-local by contract
    (engine-cli C-4): they are excluded from the upstream package and only
    surfaced as an ``excluded_external`` count.
    """
    workspace_root = resolve_workspace_root(args.workspace_root)
    index = load_index(workspace_root)
    submitted_at = index.get("submitted_at")
    entries = index.get("entries", [])
    if not args.all and submitted_at:
        selected = [e for e in entries if str(e.get("created", "")) > submitted_at]
    else:
        selected = list(entries)
    selected.sort(key=lambda e: e.get("created", ""))
    excluded_external = sum(1 for e in selected if e.get("kind") == "external")
    selected = [e for e in selected if e.get("kind") != "external"]

    result = write_package(workspace_root, index, selected)
    result["excluded_external"] = excluded_external
    if not result.get("zip"):
        result["upstream"] = detect_upstream(index)
        return result
    result["next_steps"] = _send_guidance(result["upstream"]) + [
        "After you have dealt with the batch (sent or deliberately ignored), "
        "reset the local counter: --action mark-submitted"]
    return result


# --------------------------------------------------------------------------- #
# Probe registry (requirement 041 — Feedback Probe 化重构)
# --------------------------------------------------------------------------- #
PROBE_DEFS_REL = Path(".specify") / "shared" / "definitions" / "probe-definitions.md"
EXTERNAL_PROBES_DIRNAME = "probes"
_PROBE_ID_RE = re.compile(r"^[a-z][a-z0-9-]*$")
_CUSTOM_UNIT_RE = re.compile(r"^custom:[a-z0-9._/-]+$")
_PROBE_KINDS = ("internal", "external")
_CLASS_FIELDS = ("class_id", "kind", "collection", "target_slice",
                 "processing", "insertion_type")
_OBJECT_FIELDS = ("object_id", "class_id", "unit", "lifecycle_point")


def probe_defs_path(workspace_root: Path) -> Path:
    return workspace_root / PROBE_DEFS_REL


def _table_rows(text: str, heading: str) -> List[Dict[str, str]]:
    """Parse the first pipe table under ``## <heading>``; keys = header cells."""
    lines = text.splitlines()
    start = None
    for i, line in enumerate(lines):
        if line.strip().lower() == f"## {heading}".lower():
            start = i + 1
            break
    if start is None:
        return []
    headers: Optional[List[str]] = None
    rows: List[Dict[str, str]] = []
    for line in lines[start:]:
        stripped = line.strip()
        if stripped.startswith("## "):
            break
        if not stripped.startswith("|"):
            continue
        cells = [c.strip() for c in stripped.strip("|").split("|")]
        if headers is None:
            headers = cells
            continue
        if all(set(c) <= {"-", ":"} for c in cells):
            continue
        if len(cells) == len(headers):
            rows.append(dict(zip(headers, cells)))
    return rows


def _load_external_probes(workspace_root: Path,
                          errors: List[str]) -> List[Dict[str, str]]:
    out: List[Dict[str, str]] = []
    probes_dir = feedback_dir(workspace_root) / EXTERNAL_PROBES_DIRNAME
    if not probes_dir.is_dir():
        return out
    for path in sorted(probes_dir.glob("*.md")):
        meta, _body = parse_frontmatter(path.read_text(encoding="utf-8"))
        obj = {f: str(meta.get(f, "")).strip() for f in _OBJECT_FIELDS}
        if not obj["object_id"]:
            errors.append(f"external probe {path.name}: missing object_id")
            continue
        out.append(obj)
    return out


def load_probe_registry(workspace_root: Path) -> Dict[str, Any]:
    """Load + structurally validate the merged probe truth source.

    ``available`` is False when the framework definitions file is absent
    (un-upgraded workspace): callers fall back to legacy behavior instead of
    hard-failing, per red line 2 (the mechanism must never block work).
    """
    errors: List[str] = []
    classes: List[Dict[str, str]] = []
    objects: List[Dict[str, str]] = []
    defs = probe_defs_path(workspace_root)
    available = defs.is_file()
    external: List[Dict[str, str]] = []
    if available:
        text = defs.read_text(encoding="utf-8")
        classes = _table_rows(text, "Classes")
        objects = _table_rows(text, "Objects")
        seen_classes = set()
        for c in classes:
            cid = c.get("class_id", "")
            for field in _CLASS_FIELDS:
                if not c.get(field, "").strip():
                    errors.append(f"class '{cid or '?'}': field '{field}' must be non-empty")
            if not _PROBE_ID_RE.match(cid):
                errors.append(f"class '{cid}': id must match ^[a-z][a-z0-9-]*$")
            if cid in seen_classes:
                errors.append(f"class '{cid}': duplicate class_id")
            seen_classes.add(cid)
            if c.get("kind", "") not in _PROBE_KINDS:
                errors.append(f"class '{cid}': kind must be one of internal|external")
        class_ids = {c.get("class_id", "") for c in classes}
        seen_objects = set()
        for o in objects:
            oid = o.get("object_id", "")
            if not _PROBE_ID_RE.match(oid):
                errors.append(f"object '{oid}': id must match ^[a-z][a-z0-9-]*$")
            if oid in seen_objects:
                errors.append(f"object '{oid}': duplicate object_id")
            seen_objects.add(oid)
            if o.get("class_id", "") not in class_ids:
                errors.append(f"object '{oid}': unknown class '{o.get('class_id', '')}'")
            if not validate_unit_id(o.get("unit", "")):
                errors.append(
                    f"object '{oid}': unit '{o.get('unit', '')}' must be /speckit.* or skill:*")
        external = _load_external_probes(workspace_root, errors)
        class_kind = {c.get("class_id", ""): c.get("kind", "") for c in classes}
        for e in external:
            oid = e.get("object_id", "")
            if not oid.startswith("ext-"):
                errors.append(f"external probe '{oid}': object_id MUST start with 'ext-'")
            if oid in seen_objects:
                errors.append(f"external probe '{oid}': duplicate object_id")
            seen_objects.add(oid)
            kind = class_kind.get(e.get("class_id", ""), "")
            if kind != "external":
                errors.append(
                    f"external probe '{oid}': class '{e.get('class_id', '')}' "
                    "must be kind=external")
            if not _CUSTOM_UNIT_RE.match(e.get("unit", "")):
                errors.append(
                    f"external probe '{oid}': unit '{e.get('unit', '')}' must match custom:<owner>/<name>")
    return {
        "available": available,
        "defs_path": defs.as_posix(),
        "classes": classes,
        "objects": objects,
        "external": external,
        "errors": errors,
    }


def _class_index(registry: Dict[str, Any]) -> Dict[str, Dict[str, str]]:
    return {c.get("class_id", ""): c for c in registry["classes"]}


def merged_probe_objects(registry: Dict[str, Any]) -> List[Dict[str, Any]]:
    """Internal + external objects with class features merged in."""
    classes = _class_index(registry)

    def _merge(obj: Dict[str, str], is_external: bool) -> Dict[str, Any]:
        c = classes.get(obj.get("class_id", ""), {})
        return {
            "object_id": obj.get("object_id", ""),
            "class_id": obj.get("class_id", ""),
            "unit": obj.get("unit", ""),
            "lifecycle_point": obj.get("lifecycle_point", ""),
            "kind": c.get("kind", ""),
            "slice": c.get("target_slice", ""),
            "collection": c.get("collection", ""),
            "processing": c.get("processing", ""),
            "external": is_external,
        }

    out = [_merge(o, False) for o in registry["objects"]]
    out += [_merge(e, True) for e in registry["external"]]
    return out


def resolve_probe(
    registry: Dict[str, Any], unit_id: str, lifecycle_point: Optional[str] = None
) -> Optional[Dict[str, Any]]:
    if lifecycle_point:
        for obj in merged_probe_objects(registry):
            if obj["unit"] == unit_id and obj["lifecycle_point"] == lifecycle_point:
                return obj
        return None
    for obj in merged_probe_objects(registry):
        if obj["unit"] == unit_id:
            return obj
    return None


def scan_embed_units(workspace_root: Path) -> Dict[str, str]:
    """unit_id → file, from ``## Feedback`` embeds (framework repo layout)."""
    found: Dict[str, str] = {}
    cmd_dir = workspace_root / "templates" / "commands"
    if cmd_dir.is_dir():
        for path in sorted(cmd_dir.glob("*.md")):
            if "## Feedback" in path.read_text(encoding="utf-8"):
                found[f"/speckit.{path.stem}"] = path.relative_to(workspace_root).as_posix()
    skills_dir = workspace_root / "skills"
    if skills_dir.is_dir():
        for path in sorted(skills_dir.glob("*/SKILL.md")):
            if "## Feedback" in path.read_text(encoding="utf-8"):
                found[f"skill:{path.parent.name}"] = path.relative_to(workspace_root).as_posix()
    return found


def action_probes(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    registry = load_probe_registry(workspace_root)
    errors = list(registry["errors"])
    if args.validate or args.reconcile:
        if not registry["available"]:
            raise FeedbackError(f"probe registry not found: {probe_defs_path(workspace_root)}")
        if args.reconcile:
            embeds = scan_embed_units(workspace_root)
            declared = {o.get("unit", ""): o for o in registry["objects"]}
            wrapup_classes = {
                c.get("class_id", "")
                for c in registry["classes"]
                if c.get("insertion_type", "") == "wrap-up"
            }
            for unit in sorted(u for u in embeds if u not in declared):
                errors.append(f"embed without probe object: {unit} ({embeds[unit]})")
            for obj in registry["objects"]:
                if obj.get("class_id", "") not in wrapup_classes:
                    continue
                if obj.get("unit", "") not in embeds:
                    errors.append(
                        f"probe object without embed: {obj.get('object_id', '')} "
                        f"({obj.get('unit', '')})")
        if errors:
            raise FeedbackError("probe registry invalid:\n- " + "\n- ".join(errors))
        return {
            "ok": True,
            "reconciled": bool(args.reconcile),
            "classes": len(registry["classes"]),
            "objects": len(registry["objects"]),
            "external_objects": len(registry["external"]),
            "embeds": len(scan_embed_units(workspace_root)) if args.reconcile else None,
        }
    return {
        "available": registry["available"],
        "classes": registry["classes"],
        "objects": merged_probe_objects(registry),
    }


PROBE_MAP_NAME = "probe-map.md"


def _map_slug(name: str) -> str:
    """Deterministic Mermaid node id: keep [a-zA-Z0-9_] only."""
    return _SLUG_RE.sub("_", name).strip("_")


def compose_probe_map(registry: Dict[str, Any]) -> str:
    """Render the derived probe map (deterministic: sorted, no timestamps).

    Rebuilding from an unchanged truth source MUST be byte-identical — that
    invariant is what makes drift detectable (SC-003).
    """
    classes = _class_index(registry)
    objects = merged_probe_objects(registry)
    lines: List[str] = [
        "# Feedback Probe Map(反馈插点结构图)",
        "",
        "> **派生物** — 由 `--action map` 自 probe 真源整体重建,禁止手工编辑;",
        "> 真源:`shared/definitions/probe-definitions.md` + 项目外部 probe。",
        "",
    ]
    for kind, kind_label in (("internal", "internal(内部 — 目标为 Spec Kit 框架)"),
                             ("external", "external(外部 — 目标为宿主项目自定义单元)")):
        group = sorted((o for o in objects if o.get("kind") == kind),
                       key=lambda o: (o.get("class_id", ""), o.get("object_id", "")))
        by_class: Dict[str, List[Dict[str, Any]]] = {}
        for o in group:
            by_class.setdefault(o.get("class_id", ""), []).append(o)
        # render EVERY class of this kind — zero-object classes stay visible
        kind_classes = sorted(
            (c for c in registry["classes"]
             if c.get("kind") == kind and c.get("class_id")),
            key=lambda c: c.get("class_id", ""))
        if not kind_classes:
            continue
        lines.append(f"## {kind_label}")
        lines.append("")
        for cid in [c.get("class_id", "") for c in kind_classes]:
            c = classes.get(cid, {})
            members = sorted(by_class.get(cid, []),
                             key=lambda x: x.get("object_id", ""))
            lines.append(f"### {cid}  [slice: {c.get('target_slice', '-')}]")
            lines.append("")
            lines.append(f"- **收集内容**: {c.get('collection', '-')}")
            lines.append(f"- **处理流程**: {c.get('processing', '-')}")
            lines.append(f"- **适用插入位置**: {c.get('insertion_type', '-')}")
            if members:
                lines.append(f"- **Objects** ({len(members)}):")
                for o in members:
                    lines.append(
                        f"  - `{o.get('object_id')}` — {o.get('unit')} @ "
                        f"{o.get('lifecycle_point')}")
            else:
                lines.append("- **Objects** (0 — 尚无实例;外部类经模式三注入)")
            lines.append("")
    lines.append("## 结构总览(Mermaid)")
    lines.append("")
    lines.append("```mermaid")
    lines.append("graph TD")
    lines.append("  root[Feedback Probe]")
    for kind in ("internal", "external"):
        group = [o for o in objects if o.get("kind") == kind]
        kind_classes = sorted(
            (c for c in registry["classes"]
             if c.get("kind") == kind and c.get("class_id")),
            key=lambda c: c.get("class_id", ""))
        if not group and not kind_classes:
            continue
        kind_node = _map_slug(f"kind {kind}")
        lines.append(f"  root --> {kind_node}[{kind}]")
        by_class: Dict[str, List[Dict[str, Any]]] = {}
        for o in group:
            by_class.setdefault(o.get("class_id", ""), []).append(o)
        for cid in [c.get("class_id", "") for c in kind_classes]:
            class_node = _map_slug(f"class {cid}")
            lines.append(f"  {kind_node} --> {class_node}[{cid}]")
            for o in sorted(by_class.get(cid, []), key=lambda x: x.get("object_id", "")):
                obj_node = _map_slug(f"obj {o.get('object_id', '')}")
                lines.append(f"  {class_node} --> {obj_node}[{o.get('object_id')}]")
    lines.append("```")
    lines.append("")
    lines.append("## 明细表")
    lines.append("")
    lines.append("| Object | Class | Kind | 插入位置(unit @ lifecycle) | 收集内容 | 处理流程 |")
    lines.append("|--------|-------|------|------------------------------|----------|----------|")
    for o in sorted(objects, key=lambda x: (x.get("kind", ""), x.get("class_id", ""),
                                            x.get("object_id", ""))):
        c = classes.get(o.get("class_id", ""), {})
        lines.append(
            f"| `{o.get('object_id')}` | {o.get('class_id')} | {o.get('kind')} "
            f"| {o.get('unit')} @ {o.get('lifecycle_point')} "
            f"| {c.get('collection', '-')} | {c.get('processing', '-')} |")
    lines.append("")
    return "\n".join(lines)


def action_map(args: argparse.Namespace) -> Dict[str, Any]:
    workspace_root = resolve_workspace_root(args.workspace_root)
    registry = load_probe_registry(workspace_root)
    if not registry["available"]:
        raise FeedbackError(f"probe registry not found: {probe_defs_path(workspace_root)}")
    if registry["errors"]:
        raise FeedbackError(
            "probe registry invalid (run --action probes --validate):\n- "
            + "\n- ".join(registry["errors"][:10]))
    target_dir = ensure_feedback_dir(workspace_root)
    (target_dir / PROBE_MAP_NAME).write_text(
        compose_probe_map(registry), encoding="utf-8")
    rel = (FEEDBACK_SUBDIR / PROBE_MAP_NAME).as_posix()
    return {
        "map": rel,
        "classes": len(registry["classes"]),
        "objects": len(registry["objects"]),
        "external_objects": len(registry["external"]),
    }


CLEANUP_LOG_NAME = "cleanup-log.md"


def action_cleanup(args: argparse.Namespace) -> Dict[str, Any]:
    """Post-package cleanup (req 041 Mode 2): remove entries that a package
    already archived — the zip is the record; the active store converges.

    Scope is strictly the packaged batch (engine-cli C-5): only entry files
    actually inside the named zip are removed, never un-packaged entries.
    """
    workspace_root = resolve_workspace_root(args.workspace_root)
    packages_dir = feedback_dir(workspace_root) / PACKAGES_DIRNAME
    spec = (getattr(args, "package", None) or "").strip()
    if spec == "latest":
        zips = sorted(packages_dir.glob("feedback-*.zip")) if packages_dir.is_dir() else []
        if not zips:
            raise FeedbackError(f"no packages found under {packages_dir}")
        zip_path = zips[-1]
    else:
        if not spec:
            raise FeedbackError("--package <zip-path|latest> is required.")
        zip_path = Path(spec)
        if not zip_path.is_absolute():
            zip_path = workspace_root / spec
    if not zip_path.is_file():
        raise FeedbackError(f"package not found: {zip_path}")

    with zipfile.ZipFile(zip_path) as zf:
        names = set(zf.namelist())
        if "MANIFEST.md" not in names:
            raise FeedbackError(f"package has no MANIFEST.md: {zip_path}")
        packaged = {n for n in names if n.endswith(".md")
                    and n not in ("MANIFEST.md", "SUBMISSION-NOTES.md")}

    index = load_index(workspace_root)
    targeted = [e for e in index.get("entries", []) if e.get("file") in packaged]
    dry_run = bool(getattr(args, "dry_run", False))
    removed: List[str] = []
    if not dry_run:
        store = feedback_dir(workspace_root)
        for entry in targeted:
            source = store / entry["file"]
            if source.is_file():
                source.unlink()
            removed.append(entry.get("id", entry["file"]))
        index["entries"] = [e for e in index.get("entries", [])
                            if e.get("file") not in packaged]
        index["count_since_submission"] = count_since_submission(
            index["entries"], index.get("submitted_at"))
        save_index(workspace_root, index)
        log_path = store / CLEANUP_LOG_NAME
        stamp = now_iso()
        lines = [f"## {stamp} — {zip_path.name}"]
        for entry in targeted:
            lines.append(
                f"- removed {entry.get('id', '?')} ({entry.get('unit_id', '?')}, "
                f"created {entry.get('created', '-')}) — archived in "
                f"{zip_path.name}")
        with log_path.open("a", encoding="utf-8") as fh:
            fh.write("\n".join(lines) + "\n")
    would_remove = [e.get("id", e["file"]) for e in targeted]
    return {
        "package": zip_path.relative_to(workspace_root).as_posix()
        if zip_path.is_relative_to(workspace_root) else str(zip_path),
        "dry_run": dry_run,
        "would_remove": would_remove if dry_run else [],
        "removed": removed,
        "remaining_entries": len(load_index(workspace_root)["entries"]),
    }


MIGRATION_LOG_NAME = "migration-log.md"


def action_migrate_legacy(args: argparse.Namespace) -> Dict[str, Any]:
    """One-shot legacy-entry convergence (req 041, engine-cli C-8).

    The plan file (agent-produced, user-confirmed) carries one directive per
    line: ``<entry-id> -> delete | re-register``. The engine only executes;
    it never decides dispositions itself. Every outcome lands in
    ``migration-log.md``.
    """
    workspace_root = resolve_workspace_root(args.workspace_root)
    plan_path = Path(getattr(args, "plan_file", "") or "")
    if not str(plan_path):
        raise FeedbackError("--plan-file is required.")
    if not plan_path.is_absolute():
        plan_path = workspace_root / plan_path
    if not plan_path.is_file():
        raise FeedbackError(f"plan file not found: {plan_path}")

    registry = load_probe_registry(workspace_root)
    if not registry["available"]:
        raise FeedbackError(f"probe registry not found: {probe_defs_path(workspace_root)}")

    directives: List[Tuple[str, str]] = []
    for raw in plan_path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if "->" not in line:
            raise FeedbackError(f"unparsable plan line: {raw!r}")
        entry_id, _, action = (part.strip() for part in line.partition("->"))
        action = action.split("#", 1)[0].strip()  # tolerate inline comments
        entry_id = entry_id.split("#", 1)[0].strip()
        if action not in ("delete", "re-register"):
            raise FeedbackError(f"unknown disposition {action!r} for {entry_id}")
        directives.append((entry_id, action))

    index = load_index(workspace_root)
    by_id = {e.get("id"): e for e in index.get("entries", [])}
    unknown = [eid for eid, _ in directives if eid not in by_id]
    if unknown:
        raise FeedbackError("plan references unknown entry ids: " + ", ".join(unknown))

    store = feedback_dir(workspace_root)
    deleted: List[str] = []
    re_registered: List[str] = []
    log_lines = [f"## {now_iso()} — plan {plan_path.name}"]
    for entry_id, action in directives:
        entry = by_id[entry_id]
        entry_file = store / entry["file"]
        if action == "delete":
            if entry_file.is_file():
                entry_file.unlink()
            index["entries"] = [e for e in index["entries"] if e.get("id") != entry_id]
            deleted.append(entry_id)
            log_lines.append(
                f"- {entry_id} | delete | rationale: per approved plan | "
                f"unit {entry.get('unit_id', '?')}")
        else:  # re-register in place with probe attribution
            if not entry_file.is_file():
                raise FeedbackError(f"entry file missing for re-register: {entry_id}")
            meta, body = parse_frontmatter(entry_file.read_text(encoding="utf-8"))
            probe = resolve_probe(registry, str(meta.get("unit_id", "")))
            if probe is None:
                raise FeedbackError(
                    f"cannot re-register {entry_id}: no probe object for unit "
                    f"{meta.get('unit_id', '?')}")
            meta["probe"] = probe["object_id"]
            meta["kind"] = probe["kind"]
            meta["slice"] = probe["slice"]
            meta["migrated_from"] = entry_id
            entry_file.write_text(
                dump_frontmatter(meta) + "\n\n" + body.strip() + "\n",
                encoding="utf-8",
            )
            mirror = next(e for e in index["entries"] if e.get("id") == entry_id)
            mirror.update({"probe": probe["object_id"], "kind": probe["kind"],
                           "slice": probe["slice"]})
            re_registered.append(entry_id)
            log_lines.append(
                f"- {entry_id} | re-register | probe {probe['object_id']} "
                f"| unit {meta.get('unit_id', '?')}")

    index["count_since_submission"] = count_since_submission(
        index["entries"], index.get("submitted_at"))
    save_index(workspace_root, index)
    with (store / MIGRATION_LOG_NAME).open("a", encoding="utf-8") as fh:
        fh.write("\n".join(log_lines) + "\n")
    remaining = sum(1 for e in index["entries"] if not e.get("probe"))
    return {
        "deleted": deleted,
        "re_registered": re_registered,
        "legacy_remaining": remaining,
        "log": (FEEDBACK_SUBDIR / MIGRATION_LOG_NAME).as_posix(),
    }


def action_probe_inject(args: argparse.Namespace) -> Dict[str, Any]:
    """Mode 3 (req 041, engine-cli C-6): inject an external probe for a
    host-project custom unit into the project-side registry."""
    workspace_root = resolve_workspace_root(args.workspace_root)
    unit = ((getattr(args, "unit", None) or getattr(args, "unit_id", None) or "")).strip()
    if not _CUSTOM_UNIT_RE.match(unit):
        raise FeedbackError("--unit must match custom:<owner>/<name>.")
    registry = load_probe_registry(workspace_root)
    if not registry["available"]:
        raise FeedbackError(f"probe registry not found: {probe_defs_path(workspace_root)}")
    external_classes = [c for c in registry["classes"]
                        if c.get("kind") == "external" and c.get("class_id")]
    if not external_classes:
        raise FeedbackError("registry declares no kind=external class to instantiate.")
    class_id = external_classes[0]["class_id"]
    lifecycle = (getattr(args, "lifecycle_point", None) or "wrap-up").strip()

    slug = _SLUG_RE.sub("-", unit.removeprefix("custom:")).strip("-")
    object_id = f"ext-{slug}"
    probes_dir = feedback_dir(workspace_root) / EXTERNAL_PROBES_DIRNAME
    probes_dir.mkdir(parents=True, exist_ok=True)
    target = probes_dir / f"{object_id}.md"
    if target.exists():
        raise FeedbackError(
            f"probe object already exists: {object_id} ({target.name})")

    notes = ""
    notes_file = getattr(args, "notes_file", None)
    if notes_file:
        notes = Path(notes_file).read_text(encoding="utf-8").strip()
    elif getattr(args, "notes", None):
        notes = str(args.notes).strip()

    meta = {
        "object_id": object_id,
        "class_id": class_id,
        "unit": unit,
        "lifecycle_point": lifecycle,
    }
    fm = "\n".join(f"{k}: {json.dumps(v, ensure_ascii=False)}"
                   for k, v in meta.items())
    target.write_text(
        f"---\n{fm}\n---\n\n{notes or 'External probe for a host-project custom unit.'}\n",
        encoding="utf-8",
    )
    # validate the registry still passes with the new object
    recheck = load_probe_registry(workspace_root)
    if recheck["errors"]:
        target.unlink(missing_ok=True)
        raise FeedbackError(
            "injection would invalidate the registry:\n- "
            + "\n- ".join(recheck["errors"]))
    return {
        "object_id": object_id,
        "class_id": class_id,
        "path": (FEEDBACK_SUBDIR / EXTERNAL_PROBES_DIRNAME
                 / f"{object_id}.md").as_posix(),
        "note": "external probe feedback stays host-project-local; "
                "it is excluded from upstream packages.",
    }


# --------------------------------------------------------------------------- #
# Rendering / CLI
# --------------------------------------------------------------------------- #
def render_text(action: str, payload: Dict[str, Any]) -> str:
    if action == "probes":
        if payload.get("ok"):
            scope = "reconciled zero-gap" if payload.get("reconciled") else "valid"
            return (f"Probe registry {scope}: {payload.get('classes')} classes, "
                    f"{payload.get('objects')} internal + "
                    f"{payload.get('external_objects')} external objects"
                    + (f", {payload.get('embeds')} embeds" if payload.get("reconciled") else ""))
        if not payload.get("available"):
            return ("No probe registry found at "
                    ".specify/shared/definitions/probe-definitions.md — "
                    "run /speckit.instructions to install it.")
        lines = ["# Probe overview", ""]
        objects = payload.get("objects", [])
        classes = payload.get("classes", [])
        for kind in ("internal", "external"):
            kind_classes = sorted(
                (c for c in classes
                 if c.get("kind") == kind and c.get("class_id")),
                key=lambda c: c.get("class_id", ""))
            if not kind_classes:
                continue
            lines.append(f"## {kind}")
            lines.append("")
            by_class: Dict[str, List[Dict[str, Any]]] = {}
            for o in objects:
                if o.get("kind") == kind:
                    by_class.setdefault(o.get("class_id", ""), []).append(o)
            for c in kind_classes:
                cid = c.get("class_id", "")
                members = sorted(by_class.get(cid, []),
                                 key=lambda x: x.get("object_id", ""))
                lines.append(
                    f"- {cid}  [slice: {c.get('target_slice', '-')}] — "
                    f"{c.get('collection', '-')} → {c.get('processing', '-')}")
                if members:
                    for o in members:
                        lines.append(
                            f"  - {o.get('object_id')}   ({o.get('unit')} @ "
                            f"{o.get('lifecycle_point')})")
                else:
                    lines.append("  - (0 objects — 尚无实例;外部类经模式三注入)")
            lines.append("")
        return "\n".join(lines).rstrip()
    if action == "package":
        if not payload.get("zip"):
            return payload.get("note", "Nothing to package.")
        lines = [
            f"Packaged {payload['packaged']} feedback entr"
            f"{'y' if payload['packaged'] == 1 else 'ies'} (sources untouched):",
            f"  zip: {payload['zip']}",
        ]
        if payload.get("missing"):
            lines.append(f"  missing entry files skipped: {payload['missing']}")
        lines.extend(f"  {step}" for step in payload.get("next_steps", []))
        return "\n".join(lines)
    if action == "upstream":
        url = payload.get("url")
        if not url:
            return ("Upstream repo: unknown — configure once via "
                    "--action upstream --set <repo-url>")
        commit = f" @ {payload['commit']}" if payload.get("commit") else ""
        return f"Upstream repo ({payload.get('source')}): {url}{commit}"
    if action == "list":
        matches = payload.get("matches", [])
        if not matches:
            return "No matching entries."
        lines = []
        for item in matches:
            lines.append(f"- [{item.get('unit_type', '?')}] {item.get('unit_id', '?')}")
            lines.append(f"  path: {item.get('path', '')}")
            lines.append(
                f"  run_id: {item.get('run_id', '-')} | feature: {item.get('feature') or '-'}"
                f" | feature_id: {item.get('feature_id') or '-'}"
                f" | partial: {item.get('partial', False)}"
            )
            lines.append(f"  created: {item.get('created', '-')}")
            if item.get("summary"):
                lines.append(f"  summary: {item['summary']}")
        return "\n".join(lines)
    return json.dumps(payload, ensure_ascii=False, indent=2)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Spec Kit feedback-as-files engine")
    parser.add_argument(
        "--action",
        required=True,
        choices=["record", "status", "list", "dispose", "mark-submitted", "reindex",
                 "package", "cleanup", "upstream", "probes", "map", "migrate-legacy", "probe-inject"],
    )
    parser.add_argument("--package", default=None,
                        help="cleanup: path to the package zip (workspace-"
                             "relative or absolute), or 'latest'")
    parser.add_argument("--unit", default=None,
                        help="probe-inject: target custom unit custom:<owner>/<name>")
    parser.add_argument("--lifecycle-point", default=None,
                        help="record: resolve the probe object by (unit, lifecycle_point), "
                             "e.g. gate-<slug> for gate probes; "
                             "probe-inject: lifecycle point (default wrap-up)")
    parser.add_argument("--notes-file", default=None,
                        help="probe-inject: collection-intent note file")
    parser.add_argument("--plan-file", default=None,
                        help="migrate-legacy: approved disposition plan, one "
                             "'<entry-id> -> delete|re-register' per line")
    parser.add_argument("--dry-run", action="store_true",
                        help="cleanup: list what would be removed, change nothing")
    parser.add_argument("--slice", default=None,
                        help="list: filter entries by target system slice "
                             "(inherited from the entry's probe class)")
    parser.add_argument("--kind", default=None,
                        help="list: filter entries by probe kind "
                             "(internal|external)")
    parser.add_argument("--disposition", default=None,
                        help="list: filter entries by disposition "
                             "(processed|ignored|open)")
    parser.add_argument("--id", default=None,
                        help="dispose: target entry id")
    parser.add_argument("--to", default=None,
                        help="dispose: target disposition state "
                             "(processed|ignored)")
    parser.add_argument("--validate", action="store_true",
                        help="probes: structurally validate the merged probe "
                             "registry (exit 2 listing every violation)")
    parser.add_argument("--reconcile", action="store_true",
                        help="probes: additionally reconcile internal probe "
                             "objects against live ## Feedback embeds "
                             "(two-way zero-gap required)")
    parser.add_argument("--workspace-root", default=None)
    parser.add_argument("--unit-id", default=None)
    parser.add_argument("--unit-type", default=None)
    parser.add_argument("--run-id", default=None)
    parser.add_argument("--review", default=None)
    parser.add_argument("--review-file", default=None)
    parser.add_argument("--points", default=None)
    parser.add_argument("--points-file", default=None)
    parser.add_argument("--partial", action="store_true")
    parser.add_argument("--feature", default=None,
                        help="record: requirement key (e.g. 038-goal-target); "
                             "NOT the Feature registry ID")
    parser.add_argument("--feature-id", dest="feature_id", default=None,
                        help="record: Feature registry ID (e.g. 041); distinct "
                             "number space from --feature")
    parser.add_argument("--notes", default=None,
                        help="mark-submitted: disposition summary for this batch, "
                             "archived as SUBMISSION-NOTES.md inside the package")
    parser.add_argument("--threshold", type=int, default=None)
    parser.add_argument("--since", default=None)
    parser.add_argument("--limit", type=int, default=5,
                        help="list: max entries returned; 0 = no limit")
    parser.add_argument("--contains", default=None,
                        help="list: case-insensitive substring filter over entry "
                             "summary + body (engine-side read; output stays summary-level)")
    parser.add_argument("--all", action="store_true",
                        help="package: include entries from before the last "
                             "mark-submitted as well")
    parser.add_argument("--set", dest="set_url", default=None,
                        help="upstream: persist the upstream repo URL")
    parser.add_argument("--format", choices=["text", "json"], default="text")
    return parser


_ACTIONS = {
    "record": action_record,
    "status": action_status,
    "list": action_list,
    "dispose": action_dispose,
    "mark-submitted": action_mark_submitted,
    "reindex": action_reindex,
    "package": action_package,
    "upstream": action_upstream,
    "probes": action_probes,
    "map": action_map,
    "cleanup": action_cleanup,
    "migrate-legacy": action_migrate_legacy,
    "probe-inject": action_probe_inject,
}


_DESTRUCTIVE_ACTIONS = {"package", "cleanup", "migrate-legacy",
                        "mark-submitted", "dispose"}


def _cwd_project_root() -> Optional[Path]:
    """Nearest ancestor of the CURRENT working directory containing .specify/."""
    cur = Path.cwd().resolve()
    for candidate in [cur, *cur.parents]:
        if (candidate / ".specify").is_dir():
            return candidate
    return None


def warn_anchor_mismatch(action: str, args: argparse.Namespace) -> None:
    """F16 guard: destructive actions warn when the engine's self-location
    anchor disagrees with the CWD's project root (cross-project invocation
    without an explicit --workspace-root — the 041 T021 incident class).
    Non-blocking (red line 2): a warning, never a gate."""
    if action not in _DESTRUCTIVE_ACTIONS:
        return
    if getattr(args, "workspace_root", None):
        return  # explicit override always silences the check
    try:
        anchored = resolve_workspace_root(None)
    except Exception:
        return
    cwd_root = _cwd_project_root()
    if cwd_root is None or anchored == cwd_root:
        return
    print(
        f"warning: destructive action '{action}' is anchored at {anchored} "
        f"(engine self-location) while the CWD project appears to be "
        f"{cwd_root}. Pass --workspace-root to target the intended store.",
        file=sys.stderr,
    )


def main(argv: Optional[List[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    warn_anchor_mismatch(args.action, args)
    try:
        payload = _ACTIONS[args.action](args)
    except FeedbackError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    if args.format == "json" or args.action in ("record", "status", "mark-submitted", "reindex"):
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print(render_text(args.action, payload))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
