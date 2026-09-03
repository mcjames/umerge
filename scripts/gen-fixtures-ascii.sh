#!/usr/bin/env bash
# Generates a reusable ASCII-only manual test-fixture tree for umerge:
# three directories, A (left), C (common ancestor/middle), B (right), laid
# out so that
#
#   umerge A B      exercises every 2-way scenario
#   umerge A C B    exercises every 3-way merge-classifier scenario
#
# (note the argument order for the 3-way case: left, middle, right = A C B,
# NOT A B C — umerge's own positional convention, see main.go)
#
# Idempotent: re-running wipes and rebuilds OUTDIR from scratch. Safe to
# regenerate before every release rather than hand-maintaining stale fixtures.
#
# Usage: gen-fixtures-ascii.sh [OUTDIR]
#   default OUTDIR: <repo>/../test-fixtures/ascii (sibling to the umerge
#   and umerge-python checkouts, outside this git repo so it never shows
#   up in `git status`)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_OUTDIR="$SCRIPT_DIR/../../test-fixtures/ascii"
OUTDIR="${1:-$DEFAULT_OUTDIR}"

# Undo any chmod-000 fixtures from a previous run before rm -rf, or cleanup
# fails on this second run.
if [ -d "$OUTDIR" ]; then
  chmod -R u+rwX "$OUTDIR"
  rm -rf "$OUTDIR"
fi
mkdir -p "$OUTDIR"/A "$OUTDIR"/B "$OUTDIR"/C

A="$OUTDIR/A"
B="$OUTDIR/B"
C="$OUTDIR/C"

put() { # put <path> <content>  — writes content (with \n escapes expanded),
        # creating parent dirs
  mkdir -p "$(dirname "$1")"
  printf '%b' "$2" >"$1"
}

putbin() { # putbin <path> <content> — like put, but content contains \x00
  mkdir -p "$(dirname "$1")"
  printf '%b' "$2" >"$1"
}

link() { # link <path> <target-string> — symlink, target need not exist
  mkdir -p "$(dirname "$1")"
  ln -s "$2" "$1"
}

# ── 10s: pure one-sided presence ────────────────────────────────────────────

put "$A/10-left-only.txt"  "Only ever existed on the left.\n"
put "$B/11-right-only.txt" "Only ever existed on the right.\n"
put "$C/12-deleted-both.txt" "Both sides deleted this; only the ancestor still has it.\n"

# ── 20s: content-comparison states, all three sides present ────────────────

# 20: identical everywhere
put "$A/20-identical.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$C/20-identical.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$B/20-identical.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"

# 21: right edited only (left unchanged from ancestor) -> 2-way small diff,
# 3-way clean auto-merge
put "$A/21-right-edited-only.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$C/21-right-edited-only.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$B/21-right-edited-only.txt" "alpha\nbravo\ncharlie\nDELTA-EDITED-BY-RIGHT\necho\n"

# 22: many scattered hunks (right side), same shape as 21 but a longer file
# with edits at three separate, non-adjacent lines
put "$A/22-diff-many-hunks.txt" "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\neleven\ntwelve\n"
put "$C/22-diff-many-hunks.txt" "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\neleven\ntwelve\n"
put "$B/22-diff-many-hunks.txt" "one\nTWO-EDITED\nthree\nfour\nfive\nSIX-EDITED\nseven\neight\nnine\nTEN-EDITED\neleven\ntwelve\n"

# 23: left edited only (right unchanged from ancestor) -> mirror of 21
put "$A/23-left-edited-only.txt" "alpha\nBRAVO-EDITED-BY-LEFT\ncharlie\ndelta\necho\n"
put "$C/23-left-edited-only.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$B/23-left-edited-only.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"

# 24: both sides edit different, non-overlapping lines -> 2-way: A and B
# differ from each other too; 3-way: diff3 -m merges both edits cleanly
put "$A/24-nonoverlap-both-edit.txt" "ALPHA-EDITED-BY-LEFT\nbravo\ncharlie\ndelta\necho\n"
put "$C/24-nonoverlap-both-edit.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$B/24-nonoverlap-both-edit.txt" "alpha\nbravo\ncharlie\ndelta\nECHO-EDITED-BY-RIGHT\n"

# 25: both sides edit the SAME line differently -> 3-way conflict
put "$A/25-overlap-conflict.txt" "alpha\nbravo\nCHARLIE-EDITED-BY-LEFT\ndelta\necho\n"
put "$C/25-overlap-conflict.txt" "alpha\nbravo\ncharlie\ndelta\necho\n"
put "$B/25-overlap-conflict.txt" "alpha\nbravo\nCHARLIE-EDITED-BY-RIGHT\ndelta\necho\n"

# 26: binary, identical everywhere (NUL byte = umerge's binary sniff trigger)
putbin "$A/26-binary-identical.bin" '\x00BINARY-SAME-CONTENT'
putbin "$C/26-binary-identical.bin" '\x00BINARY-SAME-CONTENT'
putbin "$B/26-binary-identical.bin" '\x00BINARY-SAME-CONTENT'

# 27: binary, differs -> 2-way BinaryDifferent; 3-way conflict (diff3 never
# invoked for binary content)
putbin "$A/27-binary-different.bin" '\x00BINARY-LEFT-VERSION'
putbin "$C/27-binary-different.bin" '\x00BINARY-BASE-VERSION'
putbin "$B/27-binary-different.bin" '\x00BINARY-RIGHT-VERSION'

# 28: symlinks, same target string everywhere -> Same / no-op
put  "$A/linktarget.txt" "symlink target file\n"
put  "$C/linktarget.txt" "symlink target file\n"
put  "$B/linktarget.txt" "symlink target file\n"
link "$A/28-symlink-same-target" "linktarget.txt"
link "$C/28-symlink-same-target" "linktarget.txt"
link "$B/28-symlink-same-target" "linktarget.txt"

# 29: symlinks, right points somewhere else -> SymlinkDifferent -> conflict
put  "$B/other-target.txt" "a different symlink target\n"
link "$A/29-symlink-mismatch" "linktarget.txt"
link "$C/29-symlink-mismatch" "linktarget.txt"
link "$B/29-symlink-mismatch" "other-target.txt"

# ── 30s: 3-way-only presence combinations (no ancestor / modify-delete) ────

# 30: no common ancestor, left == right -> auto-resolves (copy right->middle)
put "$A/30-no-ancestor-same.txt" "added independently, identical content\n"
put "$B/30-no-ancestor-same.txt" "added independently, identical content\n"

# 31: no common ancestor, left != right -> conflict
put "$A/31-no-ancestor-conflict.txt" "added independently on the left\n"
put "$B/31-no-ancestor-conflict.txt" "added independently on the right\n"

# 32: right deleted it, left is UNCHANGED from ancestor -> auto-delete middle
put "$A/32-right-deleted-left-unchanged.txt" "shared original content\n"
put "$C/32-right-deleted-left-unchanged.txt" "shared original content\n"

# 33: right deleted it, left MODIFIED it -> modify/delete conflict
put "$A/33-right-deleted-left-modified.txt" "left changed this after the ancestor\n"
put "$C/33-right-deleted-left-modified.txt" "original content\n"

# 34: left deleted it, right is unchanged -> auto-delete middle
put "$C/34-left-deleted-right-unchanged.txt" "shared original content\n"
put "$B/34-left-deleted-right-unchanged.txt" "shared original content\n"

# 35: left deleted it, right modified -> conflict
put "$C/35-left-deleted-right-modified.txt" "original content\n"
put "$B/35-left-deleted-right-modified.txt" "right changed this after the ancestor\n"

# ── 40s: directory-level equivalents ────────────────────────────────────────

# 40/41: whole directory added on one side only -> copies wholesale
put "$A/40-dir-added-left-only/fileA.txt" "first file\n"
put "$A/40-dir-added-left-only/fileB.txt" "second file\n"
put "$B/41-dir-added-right-only/fileA.txt" "first file\n"
put "$B/41-dir-added-right-only/fileB.txt" "second file\n"

# 42: whole directory deleted on both sides (only ancestor has it) -> deleted
put "$C/42-dir-deleted-both/inside.txt" "will be removed from the merge\n"

# 43: directory added independently on both sides, no common ancestor, with
# MIXED children -> Go recurses per-child rather than one blanket conflict
put "$A/43-dir-no-ancestor-mixed/same-child.txt" "identical on both sides\n"
put "$B/43-dir-no-ancestor-mixed/same-child.txt" "identical on both sides\n"
put "$A/43-dir-no-ancestor-mixed/diff-child.txt" "left's version\n"
put "$B/43-dir-no-ancestor-mixed/diff-child.txt" "right's version\n"

# 44: directory deleted on one side, UNCHANGED on the other -> always a
# conflict for directories (no cheap subtree-equality check, unlike files —
# contrast with case 32)
put "$A/44-dir-one-side-deleted/unchanged.txt" "same on left and ancestor\n"
put "$C/44-dir-one-side-deleted/unchanged.txt" "same on left and ancestor\n"

# 45: directory present on all three sides with mixed per-child states ->
# each child resolves independently; the directory's own Resolution should
# stay blank (not "mixed")
put "$A/45-dir-all-present-mixed/identical.txt" "same everywhere\n"
put "$C/45-dir-all-present-mixed/identical.txt" "same everywhere\n"
put "$B/45-dir-all-present-mixed/identical.txt" "same everywhere\n"
put "$A/45-dir-all-present-mixed/left-edit.txt" "EDITED on the left\n"
put "$C/45-dir-all-present-mixed/left-edit.txt" "original\n"
put "$B/45-dir-all-present-mixed/left-edit.txt" "original\n"
put "$A/45-dir-all-present-mixed/conflict.txt" "left's edit\n"
put "$C/45-dir-all-present-mixed/conflict.txt" "original\n"
put "$B/45-dir-all-present-mixed/conflict.txt" "right's edit\n"

# ── 50s: cross-cutting edge cases ───────────────────────────────────────────

# 50: empty file everywhere
put "$A/50-empty-file.txt" ""
put "$C/50-empty-file.txt" ""
put "$B/50-empty-file.txt" ""

# 51: empty directory everywhere
mkdir -p "$A/51-empty-dir" "$C/51-empty-dir" "$B/51-empty-dir"

# 52: unreadable file (chmod 000) -> CompareError. Content is identical to
# what's readable elsewhere, so this isolates the permission error from a
# content-diff.
put "$A/52-unreadable-file.txt" "same content, unreadable on the left\n"
put "$C/52-unreadable-file.txt" "same content, unreadable on the left\n"
put "$B/52-unreadable-file.txt" "same content, unreadable on the left\n"
chmod 000 "$A/52-unreadable-file.txt"

# 53: unreadable DIRECTORY (can't even list it) -> BuildTree currently
# swallows the ReadDir error to "no children" silently; verify this doesn't
# crash and decide if it should surface more visibly.
put "$A/53-unreadable-dir/child.txt" "you should not be able to see this listed\n"
put "$C/53-unreadable-dir/child.txt" "readable on the ancestor\n"
put "$B/53-unreadable-dir/child.txt" "readable on the right\n"
chmod 000 "$A/53-unreadable-dir"

# 54: filename with spaces, small diff to also confirm hunk rendering
put "$A/54 file with spaces.txt" "left content\n"
put "$C/54 file with spaces.txt" "left content\n"
put "$B/54 file with spaces.txt" "right content, different\n"

# 55: TYPE MISMATCH — same name is a plain file on the left, a directory on
# the right. BuildTree marks the Entry IsDir=true because *any* side being a
# directory wins, which leaves the left-side plain file's path essentially
# orphaned — never compared, never obviously surfaced. Flagged as worth
# deliberately checking, not assumed handled.
put "$A/55-type-mismatch" "this is a plain file on the left\n"
put "$B/55-type-mismatch/inside.txt" "this is a directory on the right\n"

# 56: symlink vs. regular file, same name, present on all three (ancestor
# matches the left symlink) -> SymlinkDifferent -> 3-way conflict without
# ever attempting diff3
link "$A/56-symlink-vs-regular-file" "linktarget.txt"
link "$C/56-symlink-vs-regular-file" "linktarget.txt"
put  "$B/56-symlink-vs-regular-file" "a real file, not a link\n"

# 57: dangling/broken symlink, identical target string everywhere -> Same,
# should not crash despite the target not existing
link "$A/57-broken-symlink" "this-target-does-not-exist"
link "$C/57-broken-symlink" "this-target-does-not-exist"
link "$B/57-broken-symlink" "this-target-does-not-exist"

# 58: gitignore-matched file — excluded by default, visible with
# --no-gitignore. .gitignore itself must live at each queried root.
put "$A/.gitignore" "*.log\n"
put "$C/.gitignore" "*.log\n"
put "$B/.gitignore" "*.log\n"
put "$A/58-gitignored.log" "should be hidden by default\n"
put "$C/58-gitignored.log" "should be hidden by default\n"
put "$B/58-gitignored.log" "should be hidden by default, differs too\n"

# 59: a .git/ directory — always excluded, regardless of --no-gitignore
put "$A/.git/dummy" "umerge must never show this\n"
put "$B/.git/dummy" "umerge must never show this\n"

# 60: deep nesting, also carrying a right-edited-only diff at the bottom, to
# confirm indentation and recursion at depth together
put "$A/60-deeply-nested/level1/level2/level3/level4/level5/bottom.txt" "deep content\n"
put "$C/60-deeply-nested/level1/level2/level3/level4/level5/bottom.txt" "deep content\n"
put "$B/60-deeply-nested/level1/level2/level3/level4/level5/bottom.txt" "deep content, edited on the right\n"

# ── manifest ─────────────────────────────────────────────────────────────

cat >"$OUTDIR/MANIFEST.md" <<'EOF'
# umerge ASCII test fixtures

Generated by `scripts/gen-fixtures-ascii.sh`. Regenerate anytime — this
directory is disposable, the script is the source of truth.

- 2-way: `umerge A B`
- 3-way: `umerge A C B`   (left middle right — NOT A B C)

| # | Name | Tests | Expect (2-way A/B) | Expect (3-way merge `m`) |
|---|------|-------|---------------------|---------------------------|
| 10 | 10-left-only.txt | one-sided presence | left-only | copy -> middle, took=left |
| 11 | 11-right-only.txt | one-sided presence | right-only | copy -> middle, took=right |
| 12 | 12-deleted-both.txt | ancestor-only | (invisible to 2-way; A/B lack it) | auto-delete middle |
| 20 | 20-identical.txt | identical everywhere | Same | no-op |
| 21 | 21-right-edited-only.txt | right edits, left unchanged | Different (small) | clean auto-merge |
| 22 | 22-diff-many-hunks.txt | scattered multi-hunk diff | Different (3 hunks) | clean auto-merge |
| 23 | 23-left-edited-only.txt | left edits, right unchanged | Different (small) | clean auto-merge |
| 24 | 24-nonoverlap-both-edit.txt | both edit different lines | Different | clean diff3 -m merge |
| 25 | 25-overlap-conflict.txt | both edit the SAME line | Different | conflict |
| 26 | 26-binary-identical.bin | binary, identical | Same (byte short-circuit) | no-op |
| 27 | 27-binary-different.bin | binary, differs | BinaryDifferent | conflict (no diff3) |
| 28 | 28-symlink-same-target | symlink, same target | Same | no-op |
| 29 | 29-symlink-mismatch | symlink, different target | SymlinkDifferent | conflict |
| 30 | 30-no-ancestor-same.txt | no ancestor, L==R | Same | auto-resolve |
| 31 | 31-no-ancestor-conflict.txt | no ancestor, L!=R | Different | conflict |
| 32 | 32-right-deleted-left-unchanged.txt | modify/delete, unchanged | (2-way: left-only) | auto-delete middle |
| 33 | 33-right-deleted-left-modified.txt | modify/delete, modified | (2-way: left-only) | conflict |
| 34 | 34-left-deleted-right-unchanged.txt | mirrored | (2-way: right-only) | auto-delete middle |
| 35 | 35-left-deleted-right-modified.txt | mirrored | (2-way: right-only) | conflict |
| 40 | 40-dir-added-left-only/ | one-sided dir add | left-only | whole subtree copied |
| 41 | 41-dir-added-right-only/ | mirrored | right-only | whole subtree copied |
| 42 | 42-dir-deleted-both/ | ancestor-only dir | (invisible to 2-way) | whole subtree deleted |
| 43 | 43-dir-no-ancestor-mixed/ | no-ancestor dir, mixed kids | Different | per-child: same-child auto-resolves, diff-child conflicts |
| 44 | 44-dir-one-side-deleted/ | dir modify/delete | left-only | ALWAYS conflict, even though unchanged (contrast with #32) |
| 45 | 45-dir-all-present-mixed/ | mixed per-child states | Different | per-child resolution; parent stays blank/unresolved |
| 50 | 50-empty-file.txt | empty file | Same | no-op |
| 51 | 51-empty-dir/ | empty directory | present, no children | present, no children |
| 52 | 52-unreadable-file.txt | chmod 000 file | CompareError | CompareError |
| 53 | 53-unreadable-dir/ | chmod 000 directory | check: silently empty? should it warn? | same question |
| 54 | 54 file with spaces.txt | filename with spaces | Different | clean auto-merge |
| 55 | 55-type-mismatch | file vs. dir, same name | check: what renders? | check: what does merge do? (flagged, not confirmed handled) |
| 56 | 56-symlink-vs-regular-file | symlink vs. regular file | SymlinkDifferent | conflict |
| 57 | 57-broken-symlink | dangling symlink target | Same, no crash | no-op |
| 58 | 58-gitignored.log | .gitignore filtering | hidden by default; shows with --no-gitignore | same |
| 59 | .git/dummy | always-excluded .git/ | never shown, --no-gitignore or not | same |
| 60 | 60-deeply-nested/.../bottom.txt | deep nesting + recursion | Different (deep) | clean auto-merge |

## Also exercise interactively against this tree (not fixture content per se)

Selection (`s`, blocked-with-flash under a selected ancestor, bulk
`a`/`b`/`d`/`M`, `Esc` clears) - Hidden items (`h` on a file vs. a
directory, un-hiding one descendant) - Focus mode (`f`, `t`) - manual `R`
override on a conflict - `--ascii` vs default Unicode rendering -
`--read-only` - `--merge vim` vs `--merge emacs` - `git difftool -d`.
EOF

echo "Generated ASCII fixture set at: $OUTDIR"
echo "  2-way: umerge \"$A\" \"$B\""
echo "  3-way: umerge \"$A\" \"$C\" \"$B\"   (left middle right — not A B C)"
echo "See $OUTDIR/MANIFEST.md for the full case-by-case expectations."
