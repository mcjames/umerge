#!/usr/bin/env bash
# Generates a small, focused Unicode-filename manual test-fixture tree for
# umerge — deliberately separate from gen-fixtures-ascii.sh's much larger
# matrix (mixing both was judged more confusing than clarifying). This set
# is NOT re-deriving the whole 2-way/3-way merge-classifier matrix again —
# that's already covered by the ASCII set. It exists purely to stress
# filename rendering/matching: wide (East-Asian) characters, emoji grapheme
# clusters, right-to-left scripts, and Unicode normalization mismatches.
#
#   umerge A B      2-way
#   umerge A C B    3-way (left middle right — NOT A B C)
#
# Idempotent: re-running wipes and rebuilds OUTDIR from scratch.
#
# Usage: gen-fixtures-unicode.sh [OUTDIR]
#   default OUTDIR: <repo>/../test-fixtures/unicode

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_OUTDIR="$SCRIPT_DIR/../../test-fixtures/unicode"
OUTDIR="${1:-$DEFAULT_OUTDIR}"

rm -rf "$OUTDIR"
mkdir -p "$OUTDIR"/A "$OUTDIR"/B "$OUTDIR"/C

A="$OUTDIR/A"
B="$OUTDIR/B"
C="$OUTDIR/C"

put() { # put <path> <content>  — writes content (with \n escapes expanded)
  mkdir -p "$(dirname "$1")"
  printf '%b' "$2" >"$1"
}

# ── 10s: CJK (wide, East-Asian-width) filenames, the basic presence states ─

put "$A/10-identical-日本語ファイル.txt" "same content\n"
put "$C/10-identical-日本語ファイル.txt" "same content\n"
put "$B/10-identical-日本語ファイル.txt" "same content\n"

put "$A/11-right-edited-中文文件.txt" "alpha\nbravo\ncharlie\n"
put "$C/11-right-edited-中文文件.txt" "alpha\nbravo\ncharlie\n"
put "$B/11-right-edited-中文文件.txt" "alpha\nBRAVO-EDITED\ncharlie\n"

put "$A/12-left-only-한국어파일.txt" "exists only on the left\n"

put "$B/13-right-only-韓國語파일.txt" "exists only on the right\n"

# ── 20s: emoji — multi-codepoint grapheme clusters. go-runewidth sums
# per-rune width; a ZWJ sequence or flag sequence is visually ONE glyph but
# several runes, so this is exactly where column math is most likely to be
# wrong. Worth eyeballing tree alignment carefully on these two.

put "$A/20-simple-emoji-🎉-party.txt" "same content\n"
put "$C/20-simple-emoji-🎉-party.txt" "same content\n"
put "$B/20-simple-emoji-🎉-party.txt" "same content\n"

put "$A/21-zwj-family-👨‍👩‍👧‍👦.txt" "same content\n"
put "$C/21-zwj-family-👨‍👩‍👧‍👦.txt" "same content\n"
put "$B/21-zwj-family-👨‍👩‍👧‍👦.txt" "same content\n"

put "$A/22-flag-sequence-🇯🇵-japan.txt" "same content\n"
put "$C/22-flag-sequence-🇯🇵-japan.txt" "same content\n"
put "$B/22-flag-sequence-🇯🇵-japan.txt" "same content\n"

# ── 30s: right-to-left scripts — narrow/neutral width, but bidirectional.
# umerge does no special bidi handling; worth seeing how the terminal
# renders these names next to the tree's own ASCII gutter/arrows.

put "$A/30-arabic-مرحبا-بالعالم.txt" "left version\n"
put "$C/30-arabic-مرحبا-بالعالم.txt" "left version\n"
put "$B/30-arabic-مرحبا-بالعالم.txt" "right version, different\n"

put "$A/31-hebrew-שלום-עולם.txt" "same content\n"
put "$C/31-hebrew-שלום-עולם.txt" "same content\n"
put "$B/31-hebrew-שלום-עולם.txt" "same content\n"

# ── 40s: Unicode normalization mismatch — THE headline case for a Mac +
# Linux workflow. The two filenames below render as the exact same string
# in any UTF-8-aware terminal — "40-normalization-café.txt" — but are
# different byte sequences: macOS's default filesystem (APFS/HFS+)
# normalizes to NFD (decomposed: e + combining acute U+0301); Linux
# filesystems (ext4, etc.) store whatever bytes were given, here NFC
# (precomposed é, U+00E9). Deliberately NOT distinguished in the filename
# itself (an "-NFC"/"-NFD" suffix would make them look different, which
# defeats the point) — tell them apart with `ls A C B | xxd` or `python3 -c
# "import sys,unicodedata; print(unicodedata.is_normalized('NFC', open(sys.argv[1],'rb').read().decode()))"`
# if you need to confirm which is which later.
#
# If umerge matches entries by exact string equality (it does — see
# entry.lowestName), these will NOT be recognized as the same file: expect
# TWO separate entries (one left-only, one right-only) that LOOK identical
# side by side, rather than one compared pair — likely to read as a
# rendering bug (duplicate line?) before you realize it's a matching bug.
# This is the single most likely real issue this set surfaces, not a
# hypothetical — to see it happen "in the wild" rather than synthesized,
# actually copy the SAME file between your Mac and Linux box (not re-run
# this script on each, which would just give both machines identical
# bytes) and diff that pair for real.
NFC_NAME=$'40-normalization-caf\xc3\xa9.txt'   # é = U+00E9, precomposed
NFD_NAME=$'40-normalization-cafe\xcc\x81.txt'  # e + combining U+0301
put "$A/$NFC_NAME" "same visual name, precomposed bytes (typical Linux)\n"
put "$C/$NFC_NAME" "same visual name, precomposed bytes (typical Linux)\n"
put "$B/$NFD_NAME" "same visual name, decomposed bytes (typical macOS)\n"

# ── 50s: a directory itself with a wide CJK name, containing a small mix
# of children — stresses indentation, the collapse/expand arrow, and
# recursion together (this exact combination broke once before, see
# CLAUDE.md's "Collapse arrows" note on byte-length-of-arrow assumptions).
put "$A/50-混合ディレクトリ/ascii-child.txt" "plain ascii child\n"
put "$C/50-混合ディレクトリ/ascii-child.txt" "plain ascii child\n"
put "$B/50-混合ディレクトリ/ascii-child.txt" "plain ascii child\n"
put "$A/50-混合ディレクトリ/中文子文件.txt" "cjk child\n"
put "$C/50-混合ディレクトリ/中文子文件.txt" "cjk child\n"
put "$B/50-混合ディレクトリ/中文子文件.txt" "cjk child\n"
put "$A/50-混合ディレクトリ/🎉-emoji-child.txt" "emoji child\n"
put "$C/50-混合ディレクトリ/🎉-emoji-child.txt" "emoji child\n"
put "$B/50-混合ディレクトリ/🎉-emoji-child.txt" "emoji child\n"

# ── 60s: a long, all-wide-character filename — stresses runewidth.Truncate
# (must truncate on a whole-rune, whole-width-cell boundary, never mid-rune
# and never leaving the column count wrong).
put "$A/60-長長長長長長長長長長長長長長長長長長長長長長長長長長い名前のファイル.txt" "left content\n"
put "$C/60-長長長長長長長長長長長長長長長長長長長長長長長長長長い名前のファイル.txt" "left content\n"
put "$B/60-長長長長長長長長長長長長長長長長長長長長長長長長長長い名前のファイル.txt" "right content, different\n"

# ── 70s: a genuine 3-way merge conflict on a wide-character filename, to
# confirm the resolution-status marker ('c') renders correctly alongside a
# wide name rather than only having been tested against ASCII ones.
put "$A/70-競合ファイル.txt" "left's conflicting edit\n"
put "$C/70-競合ファイル.txt" "original\n"
put "$B/70-競合ファイル.txt" "right's conflicting edit\n"

# ── manifest ─────────────────────────────────────────────────────────────

cat >"$OUTDIR/MANIFEST.md" <<'EOF'
# umerge Unicode-filename test fixtures

Generated by `scripts/gen-fixtures-unicode.sh`. Deliberately small and
narrow — this is NOT the full 2-way/3-way merge-classifier matrix (see
`test-fixtures/ascii/MANIFEST.md` for that); it exists only to stress
filename rendering and matching under non-ASCII names.

- 2-way: `umerge A B`
- 3-way: `umerge A C B`   (left middle right — NOT A B C)

| # | Name | Tests |
|---|------|-------|
| 10 | 10-identical-日本語ファイル.txt | baseline: wide CJK name, identical, column alignment sanity check |
| 11 | 11-right-edited-中文文件.txt | wide CJK name + a real content diff/merge |
| 12 | 12-left-only-한국어파일.txt | wide name + one-sided presence marker alignment |
| 13 | 13-right-only-韓國語파일.txt | mirrored |
| 20 | 20-simple-emoji-🎉-party.txt | single-codepoint-ish wide emoji baseline |
| 21 | 21-zwj-family-👨‍👩‍👧‍👦.txt | ZWJ grapheme cluster — several runes, one visual glyph; likely width-math stress point |
| 22 | 22-flag-sequence-🇯🇵-japan.txt | regional-indicator pair — another multi-rune-one-glyph case |
| 30 | 30-arabic-مرحبا-بالعالم.txt | RTL script + a content diff, no bidi handling in umerge |
| 31 | 31-hebrew-שלום-עולם.txt | second RTL script sample |
| 40 | 40-normalization-café.txt (×2, byte-different) | **the headline case** — see the long comment in the script. Two files that render as the identical string "café.txt" but are different byte sequences (NFC vs NFD). Expect umerge to show these as two SEPARATE entries (one left-only, one right-only) that look identical side by side, rather than recognizing them as one file — exact-string matching in `entry.lowestName` has no normalization step. This is exactly the kind of mismatch a real Mac<->Linux copy can produce; confirm deliberately whether that's acceptable for 1.0 or worth a follow-up (Unicode NFC-normalize names before matching). |
| 50 | 50-混合ディレクトリ/ | wide-named DIRECTORY containing ascii/CJK/emoji children — indentation + collapse arrow + recursion together |
| 60 | 60-長...長い名前のファイル.txt | long all-wide-character name — truncation must land on a whole rune/column, never mid-glyph |
| 70 | 70-競合ファイル.txt | genuine 3-way conflict on a wide name — confirms the conflict marker renders correctly next to it |

Everything here uses plain ASCII file *content* on purpose (except where
the filename itself is the point) — isolating "does the name render/match
correctly" from "does the diff/merge logic work," which the ASCII set
already covers exhaustively.
EOF

echo "Generated Unicode fixture set at: $OUTDIR"
echo "  2-way: umerge \"$A\" \"$B\""
echo "  3-way: umerge \"$A\" \"$C\" \"$B\"   (left middle right — not A B C)"
echo "See $OUTDIR/MANIFEST.md for the full case-by-case notes."
