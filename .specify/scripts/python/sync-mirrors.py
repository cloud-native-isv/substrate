#!/usr/bin/env python3
"""Single-source mirror sync for spec-kit (P1, harvested from ai-website-cloner-template).

Canonical sources fan out to their runtime mirrors in one command:

  templates/            -> .specify/templates/   (excluding commands/, see below)
  skills/<name>/        -> .specify/skills/<name>/   (repo skills/ may be empty placeholder)
  agents/               -> .specify/agents/templates/  (Agent Template layer; instances/
                           and execution/ are project-local, never mirrored)
  scripts/              -> .specify/scripts/
  shared/               -> .specify/shared/
  templates/commands/*  -> per-tool copies (delegated to regen-command-copies.py)

The .specify/templates/commands/ mirror is intentionally NO LONGER maintained:
per-tool command copies are generated directly from templates/commands/ by
regen-command-copies.py, so the subtree is excluded from the templates pair on
both sides.

Modes:
  --check   report drift, exit 2 if any (CI gate); never writes
  --write   sync all mirrors from canonical sources (default)

Write-mode failures are collected per file, never fatal mid-pass: an unwritable
mirror file (e.g. a root-owned leftover) is reported as FAIL, the remaining
files still sync, and the run ends with exit 1 plus a failure summary — a
stale mirror must never pass silently.

Junk entries (__pycache__, node_modules, .DS_Store) are ignored on both sides
and never copied.

Extra files that exist only in a mirror are never deleted (archive-not-delete
discipline), but for pairs marked STRICT below they FAIL --check. Rationale:
`specify init` distributes a framework tree by copying the canonical source over
the mirror (shutil.copytree(..., dirs_exist_ok=True)) — an additive merge that
never syncs mirror->source. So a mirror-only file is unreachable by init and can
never land in a downstream project, however correct it looks locally. Two valid
remedies, and the check deliberately does not presume which: add the missing
canonical source (if the file should ship), or delete the mirror copy (if it is
a superseded leftover).
"""

from __future__ import annotations

import argparse
import filecmp
import shutil
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]

# (source, mirror, strict_extras, exclude_parts) pairs; source is canonical.
# strict_extras=True means a mirror-only file is an ERROR under --check: the tree
# is entirely framework-owned, so an unsourced file can never reach downstream
# projects via `specify init`. skills/ is intentionally lenient because a project
# may legitimately keep its own local skills in the mirror. exclude_parts are
# path components skipped on BOTH sides of the pair; the templates pair excludes
# commands/ because the .specify/templates/commands mirror is retired (per-tool
# copies come straight from templates/commands/ via regen-command-copies.py).
MIRROR_PAIRS = [
    ("templates", ".specify/templates", False, {"commands"}),
    ("skills", ".specify/skills", False, set()),
    ("agents", ".specify/agents/templates", False, set()),
    ("scripts", ".specify/scripts", True, set()),
    ("shared", ".specify/shared", False, set()),
]

IGNORE_NAMES = {"__pycache__", "node_modules", ".DS_Store", ".gitkeep"}


def iter_files(root: Path, exclude_parts=frozenset()):
    if not root.exists():
        return
    for path in sorted(root.rglob("*")):
        if path.is_dir():
            continue
        if any(
            part in IGNORE_NAMES or part in exclude_parts
            for part in path.relative_to(root).parts
        ):
            continue
        yield path.relative_to(root)


def compare_pair(src: Path, dst: Path, exclude_parts=frozenset()):
    """Return (missing_in_dst, differing, extra_in_dst) relative paths."""
    src_files = set(iter_files(src, exclude_parts))
    dst_files = set(iter_files(dst, exclude_parts))
    missing = sorted(src_files - dst_files)
    extra = sorted(dst_files - src_files)
    differing = sorted(
        rel
        for rel in src_files & dst_files
        if not filecmp.cmp(src / rel, dst / rel, shallow=False)
    )
    return missing, differing, extra


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--check", action="store_true", help="report drift, exit 2 if any")
    mode.add_argument("--write", action="store_true", help="sync mirrors (default)")
    args = parser.parse_args()
    check_only = args.check

    drift = False
    orphans = False
    failures: list[tuple[str, str]] = []
    for src_name, dst_name, strict_extras, exclude_parts in MIRROR_PAIRS:
        src = REPO_ROOT / src_name
        dst = REPO_ROOT / dst_name
        if not src.exists():
            continue
        src_files = list(iter_files(src, exclude_parts))
        if not src_files:
            # placeholder-only source (e.g. empty skills/): mirror is canonical, skip
            print(f"skip  {src_name}/ (placeholder-only source; {dst_name}/ is canonical)")
            continue
        missing, differing, extra = compare_pair(src, dst, exclude_parts)
        if not (missing or differing):
            print(f"ok    {src_name}/ == {dst_name}/ ({len(src_files)} files)")
        else:
            drift = True
            for rel in missing:
                print(f"MISS  {dst_name}/{rel}")
            for rel in differing:
                print(f"DIFF  {dst_name}/{rel}")
            if not check_only:
                synced = 0
                for rel in missing + differing:
                    target = dst / rel
                    try:
                        target.parent.mkdir(parents=True, exist_ok=True)
                        shutil.copy2(src / rel, target)
                        synced += 1
                    except OSError as exc:
                        failures.append((f"{dst_name}/{rel}", str(exc)))
                        print(f"FAIL  {dst_name}/{rel}: {exc}")
                if synced:
                    print(f"sync  {src_name}/ -> {dst_name}/ ({synced} files)")
        # Extras are never deleted (archive-not-delete discipline), but in a
        # strict tree they are a distribution defect, not a note.
        for rel in extra:
            if strict_extras:
                orphans = True
                print(f"ORPHAN {dst_name}/{rel} (no canonical {src_name}/ source)")
            else:
                print(f"note  extra file only in mirror: {dst_name}/{rel}")

    # Per-tool command copies: delegate to the existing canonical generator.
    regen = REPO_ROOT / "scripts/python/regen-command-copies.py"
    regen_args = [sys.executable, str(regen)] + (["--check"] if check_only else [])
    result = subprocess.run(regen_args, cwd=REPO_ROOT)
    if result.returncode != 0:
        drift = True

    if failures:
        print(
            f"\nSYNC FAILURES: {len(failures)} file(s) stayed stale — fix and re-run:"
        )
        for rel, err in failures:
            print(f"  FAIL {rel}: {err}")
        print(
            "  Remedy for root-owned mirrors: `sudo chown -R $USER <dir>`, then "
            "re-run `python3 scripts/python/sync-mirrors.py --write`"
        )
        return 1
    if check_only and orphans:
        print(
            "ORPHANS detected — a mirror-only file cannot be distributed by "
            "`specify init`, which only copies canonical source -> mirror.\n"
            "  Fix by either: (a) adding the canonical source (e.g. "
            "scripts/bash/<name>.sh) if the file should ship to downstream "
            "projects, or (b) deleting the mirror copy if it is a superseded "
            "leftover."
        )
    if check_only and drift:
        print("DRIFT detected — run: python3 scripts/python/sync-mirrors.py --write")
    if check_only and (drift or orphans):
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
