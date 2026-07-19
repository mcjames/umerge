# umerge — roadmap

**Positioning:** the terminal-native directory diff tool — and a drop-in
`git difftool -d` backend. Nobody owns terminal-native directory diff today
(`delta`/`difftastic` own single-file text diff; Beyond Compare/Araxis own
the GUI directory-diff space). That's the gap umerge is aimed at.

Priorities below are ordered by how much they move umerge from "prototype"
to "tool people trust on a real, messy tree and reach for by default."
Items carried over from the original Python port are noted as such;
everything else is new, added after evaluating umerge against Beyond
Compare / Araxis Merge and against git's own tooling.

**Testing policy:** the Python version had no automated tests (all
interactive). Priority 0 below is a one-time catchup burst for the
existing untested Go code. After that, the policy going forward is: any
new filesystem-mutating function (copy/delete, and later sync) gets a
table-driven test with `t.TempDir()` fixtures written before its
key-handling is wired up; any new pure logic (filtering, resolution-state
transitions, config parsing) gets tests in the same sitting it's written.
Bubble Tea `View()`/rendering and simple key-dispatch glue stay
interactively tested — low value to automate — until a state machine (e.g.
a multi-step modal prompt) grows enough branching to be worth unit-testing
`Update()` directly with synthetic `KeyMsg`s.

---

## Priority 0 — Testing catchup — ✅ DONE (2026-07-18)

One-time backfill for code that predates the testing policy above. Ordered
by leverage: the tree-diff core first, since Priority 2 is about to modify
it directly and a regression there breaks everything downstream.

**Done:** all four files below written; `go build ./...`, `go vet ./...`,
and `go test ./...` are clean — 40 tests passing across
`internal/entry`, `internal/fileops`, `internal/mergetool`, and
`internal/ui`.

### Must do (before/alongside Priority 1)
- `internal/entry/entry_test.go` — `BuildPair`, `BuildTriple`, `Flatten`.
  Table-driven, `t.TempDir()` fixtures. Cover: file present on only one
  side, interleaved sort order across mismatched dir contents (exercises
  `lowestName`), nested dirs with correct `Depth`, and `Flatten` with a mix
  of collapsed/expanded dirs.
- `internal/fileops/fileops_test.go` — `CompareTwoFiles`, `CompareThreeFiles`.
  Cover: identical files, single hunk, multiple hunks, and for
  `CompareThreeFiles` each of the four `====N` marker cases (left differs,
  middle differs, right differs, all differ) since this drives 3-way
  conflict marking in Priority 7.

### Should do (right after)
- `internal/mergetool/mergetool_test.go` — `Command`, `vimCommand`,
  `emacsCommand`, `presentPaths`. Pure — assert on `cmd.Args`/`cmd.Path`,
  no process execution needed. Catches argument-order bugs (wrong file
  ends up in the diff pane).
- `internal/ui/compare.go` — `allSidesPresent` (trivial) and
  `compareEntry` (needs real temp files; verify `Entry.Compare`/
  `NumDiffs`/`LMDiffs`/`MRDiffs` transitions, including the error path).

### Explicitly deferred
- `app.go` `Update`/`View`/key handling, and `walkAndCompare`/
  `startCompare` channel plumbing. Revisit `Update()` once Priority 1's
  copy/delete confirmation prompts (and later the 3-way `S`/`m` multi-step
  prompts) land — that's when a modal state machine emerges that's worth
  unit-testing.

---

## Priority 1 — File operations (baseline usability) — ✅ DONE (2026-07-18)

Without these umerge is read-only. Ported from Python. Per the testing
policy: write `t.TempDir()`-based tests for each copy/delete function
before wiring its key-handling into `Update()`.

**Done:** `fileops.Copy`/`fileops.Delete` (mirroring `cp -R`/`rm -Rf`
semantics), `internal/ui/ops.go` (copy/delete orchestration: path
computation for absent destinations, subtree rebuild + recompare after a
directory copy, tree splicing after delete), and the 2-way/3-way key
wiring in `app.go` including the 3-way multi-step prompt state machine.
34 new tests (`fileops`, `ui/ops_test.go`, `ui/app_test.go`) plus 3 manual
end-to-end smoke tests against the real compiled binary in a live
pty (2-way copy, 2-way delete, 3-way `a`→`c` prompt) — all passing.
`--help` text and the `umerge.1` man page are kept in sync (standing
practice now, see below).

**Bug found and fixed 2026-07-18: copying from an absent source silently
did nothing** (e.g. "b then c" when the entry was absent from the right;
"c then a" when absent from the middle/parent). Root cause: `copyEntry`
no-op'd whenever the source side was nil, with no feedback. This is
**not** a case of faithfully porting a Python design — Python's own
letter-based 3-way copy (`Model3.__copy_aux`) has its equivalent guard
commented out (`#  if item.left is None: return`), so it just lets `cp`
fail and marks the item `ERROR`, indistinguishable from a real I/O
failure. Neither behavior (silent no-op, or a fake-looking generic error)
is right. Fixed properly instead: the source side is validated the moment
the *first* letter is pressed (`Model.beginCopy` in `app.go`), before any
prompt is shown — if that column is empty there's a visible flash message
("Nothing to copy: right is absent") and, in 3-way mode, the destination
prompt never starts. A destination is never invalid to choose — copying
*to* an absent side is the normal case, since that's what creates it. An
invalid destination choice (same column as source, or an unrelated key)
now also flashes "Invalid choice" instead of silently cancelling. New
`Model.flash` field, cleared at the top of every keypress. Verified with
unit tests plus live-pty smoke tests for both reported directions
(confirmed empirically that the pty smoke-test harness needs an explicit
`TIOCSWINSZ` window size set, or Bubble Tea's `View()` renders nothing —
easy to mistake for the app being broken when it's the harness).

**Second, deeper bug found and fixed 2026-07-18 (same day, reported after
the above): copying a file into a destination whose intermediate
directories were never enumerated on that side at all silently failed.**
Concretely: `left/a/` exists but is empty; `right/a/sub/file.txt` exists.
Navigating all the way down to `file.txt` (present on the right, absent
on the left) and copying it over did nothing — no file appeared, no
visible error. Root cause: `fileops.Copy` shelled out straight to
`cp -R src dest`, and plain `cp` refuses to create a destination's missing
*parent* directories (`sub/` in this example) — it only creates the final
path component, assuming everything above it already exists. Python's
`FileOpsPOSIX.copy_primitive` has the exact same `cp -R` call and would
fail identically; this isn't a Python-vs-Go divergence, just a limitation
neither version had addressed. Fixed in `fileops.Copy`: `os.MkdirAll` the
destination's parent directory before invoking `cp -R`, so copying a
deeply-nested file always succeeds regardless of how many missing
directory levels are needed, matching what a user actually expects
"copy this over" to do. Also fixed the reason this failure was invisible
in the first place: `copyEntry`/`deleteEntry` now set `Model.flash` with
the actual error on failure (previously only the internal `Compare` state
was marked `CompareError`, with no user-visible signal at all — not even
color, since `CompareError` had no distinct style in `rowCols`, an gap
Priority 12 already listed as future work but which turned out to matter
now). Added `styleError` (per Priority 12) so failed entries stay visibly
red rather than reverting to normal white the instant the flash message
clears. Verified with a unit test mirroring the exact reported directory
shape, plus a live-pty smoke test reproducing the original report
end-to-end.

**Third bug found and fixed 2026-07-18 (same day): copying a directory
with contents didn't show those contents in the tree until an unrelated
collapse/expand.** The copy itself was correct on disk — this was purely
a stale-UI bug. `Model.flat` is a separately-maintained flattened cache
of the tree (re-derived from `Model.entries` only when `reflatten()` is
called); `copyEntry`'s `rebuildChildren` replaces `e.Children` with a
freshly-enumerated subtree, but nothing told `m.flat` to catch up, so the
rendered tree kept pointing at the old (often empty) children until
collapse/expand happened to call `reflatten()` for its own reasons. Fixed
by calling `m.reflatten()` right after `rebuildChildren` in `copyEntry`.
Added a regression test asserting on `m.flat` by object identity — not
just length, since a same-length stale slice would pass a naive check —
plus a live-pty smoke test confirming a copied directory's contents
render immediately with no collapse/expand needed.

### Copy (2-way)
- `a` — copy current item left → right
- `b` — copy current item right → left

### Copy (3-way, multi-step prompt)
**Corrected 2026-07-18 against actual source** (`Match3.letter_to_subpart`
and the Controller's own prompt text) — the column mapping is `a`=left,
`b`=right, `c`=middle, not the left/middle/right guess originally written
here:
- `a` → sub-prompt "Copy from A (left) to:" → `b` (right) or `c` (middle)
- `b` → sub-prompt "Copy from B (right) to:" → `a` (left) or `c` (middle)
- `c` → sub-prompt "Copy from C (middle) to:" → `a` (left) or `b` (right)

### Copy/delete semantics, verified against `FileOpsPOSIX.py`
- Copy is literally `cp -R src dest`; delete is literally `rm -Rf path`.
- If the copy destination already exists as a **directory**, it's deleted
  first, then `cp -R` recreates it fresh — copy fully replaces the
  destination directory rather than merging into it. If the destination
  exists as a **file**, `cp -R` overwrites it directly, no pre-delete step.
- **No confirmation prompt anywhere**, for copy or delete, in the Python
  version — `d` runs `rm -Rf` immediately on every present side of the
  current item. Decided to match this in the Go version rather than add a
  confirmation dialog.
- 3-way delete removes whichever of left/middle/right are present for
  that item (same "remove everywhere it exists" idea as 2-way, one more
  side).
- Selection-based bulk delete (`d` acting on a selection instead of the
  cursor item) depends on Priority 5, not built yet — for now `d` always
  acts on just the cursor item.

### Delete
- `d` — delete current item (both/all sides); if a selection exists, delete
  all selected items instead

---

## Priority 2 — Filtering & performance ("is this a real tool" bar)

This is what separates umerge from a toy on anything but a small, clean
tree. None of this existed in the Python version — it's the biggest gap
found when comparing against Beyond Compare / Araxis Merge, translated to
what actually matters for a CLI tool (not their GUI-only features like
image/Word diff, which are out of scope).

- **Respect `.gitignore` by default — ✅ DONE (2026-07-19).** Added
  `entry.Ignore` (`internal/entry/ignore.go`), a thin wrapper around
  `github.com/sabhiram/go-gitignore`. `LoadIgnore` reads a top-level
  `.gitignore` from each compared root, combines their patterns, and always
  adds a `.git/` rule so umerge never surfaces git's own internal object
  store as a difference — even with no `.gitignore` present. `BuildTree`
  gained `relPath`/`ig` parameters, threaded through the recursive merge so
  a matched entry (and its whole subtree, since it's skipped before
  recursing) is excluded before it's ever enumerated or compared, not just
  hidden after the fact. New `-I`/`--no-gitignore` flag disables filtering
  entirely (`ig = nil`, which `Ignore.Match` treats as "never matches").
  `Entry` gained a `RelPath` field so a later manual refresh
  (`ops.go`'s `rebuildChildren`, from Priority 4's `r` key) can re-test
  patterns with the correct root-relative path instead of assuming depth 0.

  **Deliberately scoped down, given the time available:** only the
  top-level `.gitignore` per root is read — real git's cascading,
  per-directory `.gitignore` support is a known follow-up, not implemented
  here, since a single repo-root file is the overwhelmingly common case.
  Wildcard/regex include-exclude filters (the other half of this section)
  are also still open — this pass covered gitignore only.

  **Gotcha found integrating the library:** its `MatchesPath` compiles a
  directory-only pattern (trailing `/`, e.g. `build/`) into a regexp that
  only matches a candidate string that itself ends in `/` — passing the
  bare directory name without a trailing slash silently fails to match.
  `Ignore.Match` takes an explicit `isDir bool` and appends the slash
  itself so callers can't get this wrong. Verified with a unit test
  (`TestIgnore_DirectoryOnlyPatternRequiresIsDirTrue`) that fails if that
  handling is removed.

  Verified with unit tests at three layers (`LoadIgnore`/`Match` directly,
  including negation and root-anchored-vs-nested `/dist` pattern
  correctness; `BuildPair` end-to-end showing an ignored directory's
  children never appear; `RelPath` stamping) plus a live-pty check against
  the real binary confirming `.gitignore`-matched entries and `.git` are
  hidden by default and reappear with `--no-gitignore`.
- **Include/exclude filters** — wildcard and/or regex, plus flags to ignore
  whitespace, case, and blank-line-only diffs when comparing file contents.
- **Fast short-circuit comparison + binary file detection — ✅ DONE
  (2026-07-19), built together.** `fileops.CompareTwoFiles`/
  `CompareThreeFiles` now: (1) stat + compare content in fixed-size chunks
  to *prove* equality without loading whole files into memory — if equal,
  return immediately without ever invoking `diff`/`diff3`; (2) if not
  equal, sniff the first ~8000 bytes of each file for a NUL byte (the same
  heuristic git uses); if any file is binary, return a `BinaryDifferent`
  result — again without invoking `diff`/`diff3`. Only text files that
  are genuinely different still shell out, for an accurate hunk count.
  Deliberately **not** using git's mtime-trusting shortcut (same
  size+mtime ⇒ assume unchanged): that has a real, if rare, false-negative
  risk, which is an acceptable trade for rsync's use case but not for a
  tool whose whole job is being trusted for vendor-drop/merge review
  decisions — correctness over the extra speed that heuristic would buy.
  Decided against shelling out to `file`/libmagic for binary detection
  too: that would just relocate the subprocess cost we're eliminating, for
  detection detail (specific format identification) we don't actually need
  for a binary/text yes-no decision.
  
  **Uncovered a real, pre-existing correctness bug while designing this:**
  two genuinely different binary files were silently rendered as `Same`.
  `diff` prints `Binary files ... differ` for differing binary content
  (verified empirically) rather than a normal hunk-formatted diff; the old
  hunk-counting logic (count lines starting with a digit) found none and
  reported 0 diffs, which `compareEntry` then read as "identical." Not
  something introduced this session — already present in shipped code.
  Also verified empirically: `diff3` doesn't degrade as gracefully — it
  fails outright (`diff3: diff failed: Binary files ... differ`, exit 2)
  the moment any pairwise diff inside it hits binary content, so there's
  no partial per-pair result to preserve for a mixed binary/text triple;
  the whole entry is marked `BinaryDifferent` uniformly in that case,
  matching diff3's own all-or-nothing behavior rather than inventing
  finer-grained information it doesn't give us.
  
  New `entry.CompareState` value `BinaryDifferent`: same "changed" blue
  color as a real text difference (including all three columns in 3-way,
  since LMDiffs/MRDiffs aren't meaningful here and `diffStyleForCol`
  special-cases this rather than falling through its normal per-pair
  logic), but shows `bin` instead of a numeric hunk count. Verified with
  unit tests — including setting `PATH` to an empty directory to *prove*
  `diff`/`diff3` are never invoked for the identical and binary cases,
  not just that the right answer comes back — plus a live-pty check
  against real random-content binary files confirming the fix (previously
  silently `=`/white, now correctly `bin`/blue). `umerge.1`'s "Diff
  counts" section documents the new `bin` marker.
- **Rename/move detection** (stretch) — a file moved within the tree
  shouldn't render as a delete+insert pair. Hard to get fully right; lower
  priority than the items above, but a strong "professional tool" signal.

---

## Priority 3 — External integrations

git already has a directory-diff mechanism this can plug straight into:
`git difftool --dir-diff` materializes two temp trees and invokes one
external command with two paths — the same calling convention umerge
already uses. This is likely near-zero implementation work, mostly
verification + docs.

**Verified empirically (throwaway repo + probe script, 2026-07-17):** the
side whose content matches the actual working tree is populated with
symlinks back into the real files, not copies — the other side gets real
copies. Practical implication for Priority 1's copy/delete: a delete on
the symlinked side will just unlink the symlink inside git's temp dir, not
touch the real working-tree file — worth being aware of so it doesn't
read as a silent no-op bug when this comes up. (The "only changed files,
hierarchically" shape itself is just standard `git difftool -d` behavior
for any tool, not something specific to umerge — not worth designing
around.)

- Verify umerge launches cleanly and exits without leftover terminal state
  when invoked non-interactively by `git difftool -d`.
- Document the `.gitconfig` snippet in the README, using `--read-only`
  (see the symlink-hazard note below for why):
  ```
  [difftool "umerge"]
      cmd = umerge --read-only "$LOCAL" "$REMOTE"
  [diff]
      tool = umerge
  ```
- The 3-way mode's real differentiated use case is comparing three
  arbitrary tree *snapshots* (three deploy configs, three `git worktree`
  checkouts of different branches) — not tied to an in-progress git merge.
  Worth calling out explicitly in docs/positioning, since it's a niche use
  case nobody else in the terminal space covers.

### Mercurial
`hg extdiff` uses the same mechanism as `git difftool -d`: materialize two
temp trees, invoke one external command once with two paths. Once the git
verification above is done, Mercurial support should be nearly the same
effort — verify + document the `extdiff` config snippet (`hg-git` users
get this for free either way, but plain Mercurial users are a real
audience distinct from git users).

### TUI file-manager hooks
The terminal-native equivalent of Araxis's Explorer "Queue for Comparison"
/ "Compare with Araxis Merge" shell integration: file managers like
`yazi`, `lf`, and `vifm` support binding a key to an external command with
the current selection or a bookmarked path. Document key-binding snippets
("mark this directory, navigate elsewhere, launch umerge against the
marked one") for at least yazi and vifm — vifm in particular already has
its own basic directory-compare mode, so its users are a natural audience
to convert. Low effort (docs only, no umerge code changes), same category
of value as the git/Mercurial config snippets above.

### Symlink hazard when invoked via `git difftool -d`
**The problem:** `git difftool -d` materializes `LOCAL`/`REMOTE` temp
directories, but whichever side's content matches the actual working tree
gets populated with *symlinks* back into the real working tree (git skips
copying bytes that are already on disk) — the other side gets real,
disposable copies. umerge can't tell these apart visually; they render
identically. But `delete` on the symlinked side just unlinks the symlink
(the real file is untouched — looks like it worked, didn't), and `copy`
*into* the symlinked side follows it and overwrites the real file (looks
like a harmless action inside a throwaway temp dir, actually mutates your
live working tree). Same keystroke, very different blast radius, no visual
distinction between the two cases.

**`--read-only` flag — ✅ DONE (2026-07-19).** New `-r`/`--read-only` flag:
`a`/`b`/`c`/`d` (and any future mutating command — Priority 7's `m`/`M`/`R`
once built) show a flash message explaining they're disabled instead of
acting. Everything non-mutating (navigate, collapse/expand, launch
vimdiff/emacs to *view*) still works. The recommended `git difftool`
`.gitconfig` snippet (Priority 3, git integration section above) now uses
`umerge --read-only "$LOCAL" "$REMOTE"`, making the git integration safe
by default without needing any symlink-detection logic at all. Documented
limitation: this only guarantees umerge's own commands don't mutate
anything — it can't stop you from opening a file in `vimdiff` and hitting
`:w` on the symlinked side, since that's a separate process with its own
write access. That's a much more deliberate action than a single umerge
keystroke, so treated as an acceptable, clearly-stated gap rather than
something to also solve here.

**Future, richer alternative (not yet built) — visual signaling instead of
blocking, for users who explicitly want full read-write in this mode:**
for each side of each entry, `os.Lstat` to check if it's a symlink, and if
so `os.Readlink` + resolve to an absolute path and check whether it falls
*outside* that side's root directory (a symlink that's part of the actual
project being compared is normal, not a hazard — only one escaping the
comparison root is). Mark the specific column(s) that are external
symlinks with a plain `~` appended after the filename (same position as
the existing diff-count suffix), in a color reserved exclusively for this
meaning (not yet used: magenta) — deliberately not another emoji/wide
glyph, given the ambiguous-width pain already hit twice this project (the
collapse arrows, CJK filenames); `~` is plain ASCII, ties to the existing
Unix "elsewhere" association, and needs no ascii/unicode fallback of its
own. Only the affected column gets marked — the other side keeps its
normal same/changed/absent styling. This would let `git difftool -d` users
opt out of `--read-only` and get the "resolve the diff by copying" power
feature back, with the hazard made visible instead of hidden, rather than
umerge choosing for them between "fully blocked" and "silently dangerous."

### Out of scope (considered, ruled out)
Keeping these here so they don't get re-litigated later.

- **Acting as a `git`/Mercurial `mergetool` backend.** That mechanism is
  inherently per-file (LOCAL/BASE/REMOTE/MERGED for one conflicted file at
  a time) — git/hg already dispatch that to vimdiff/ediff directly, which
  is what umerge's own 3-way per-file merge does. There's no role for a
  directory-level tool there.
- **Built-in FTP/SFTP/S3/WebDAV support** (Beyond Compare has this
  natively). The terminal ecosystem already solves "make a remote look
  like a local path" better than umerge could reinvent it: `sshfs` or
  `rclone mount`, then umerge just works on the result unmodified. Adding
  bespoke protocol clients would be real ongoing surface area for
  something already well-solved, and cuts against "does one thing well."
- **Deep VCS depot-browsing** (Araxis's File-System Plugins that let the
  diff tool browse Git/Hg/SVN/Perforce history directly, not just a
  working tree). That's effectively building a repo-history browser into
  umerge. The external-diff-tool hook above captures most of the
  practical value for a fraction of the effort and scope.
- **Windows Explorer / macOS Finder context-menu integration.** Wrong
  platform focus for a Linux/macOS terminal-first tool — the TUI
  file-manager hooks above are the honest equivalent.

---

## Priority 4 — Refresh / re-compare — ✅ DONE (2026-07-19)

Ported from Python.

- `r` — re-enumerate subtree rooted at current item, then re-compare it
  (in background, same goroutine+channel pattern already used for initial
  comparison)
- After vimdiff/ediff exits, automatically re-compare the edited entry
  (`toolDoneMsg` currently does nothing)

**Implementation notes:** both features turned out to reuse existing
machinery almost entirely rather than needing much new logic.
`toolDoneMsg` gained an `e *entry.Entry` field (the entry that was open in
the tool); its handler just calls the same `recompareSubtree` that
`copyEntry` already uses post-copy, synchronously — returning from a
full-screen external program already involves a redraw pause, so one more
`diff` call doesn't introduce a new stall. `r`'s handler (`beginRefresh`
in `ops.go`) calls the same `rebuildChildren`/`reflatten` pair `copyEntry`
uses for a directory, then starts a *background* compare via the existing
`startCompare`/`compareCh`/`comparing` machinery — unlike the tool-exit
case, a manual refresh isn't piggybacking on an already-blocking
operation, so a large subtree refreshing synchronously could cause a real
stall; TODO.md's original note calling for the background goroutine+
channel pattern was right to insist on it. Guarded against starting a
refresh while `comparing` is already true (shows a flash message instead)
since `compareCh`/`comparing` are shared with the initial scan — starting
a second concurrent compare would race on the same channel. Mirrors
Python's own `operation_thread` guard (`Model2.request_operation`), not a
new invention. Verified with unit tests (including driving the actual
Cmd/message loop the way Bubble Tea's runtime would, not just calling the
helpers directly) plus a live-pty check: externally edit a file to match
while umerge has it open showing a stale diff count, press `r`, confirm
the count updates to `=`. `--help`, README, and `umerge.1` updated for the
new `r` binding and the auto-recompare-on-tool-exit behavior.

---

## Priority 5 — Selection

Ported from Python.

- `s` — toggle selection on current item (propagates to all children)
- Selected items rendered in a distinct highlight color (Python: blue
  background `SELECTED` style)
- Bulk operations: `d` (delete) and copy commands operate on the selection
  when one exists, rather than just the cursor item

### 3-way only
- `S` — multi-step prompt: choose column (A/B/C), then choose feature
  (absent / unchanged / changed / inserted); selects matching items
- `x` sub-command of `S` — clear selection

**Bug found in the Python source, not to be ported as-is:**
`Match3.selection_matches(self, column, feature)` ignores both of its
parameters — it always just returns whether the item is "present in left,
absent in middle," regardless of which column/feature the user actually
chose in the `S` prompt. Looks like unfinished code, not an intentional
design choice. Implement this properly in Go: actually honor the chosen
column (A/B/C → left/right/middle) and feature (absent/unchanged/changed/
inserted) rather than reproducing the stub.

---

## Priority 6 — Hidden items

Ported from Python.

- `h` — toggle the user-hidden flag on the current item (and its subtree).
  This is a user-managed "ignore" state, unrelated to dot-files or the
  `.gitignore` filtering in Priority 2.
- `H` — toggle whether hidden items are rendered at all (`render_hidden` flag)
- Hidden items shown in a dimmed color when rendered; skipped entirely when
  `render_hidden` is false

---

## Priority 7 — 3-way merge workflow

Ported from Python.

### Resolution status markers
Each entry has a one-character resolution status prefix displayed at the
start of its row:

| Char | Meaning | Color |
|------|---------|-------|
| ` `  | unresolved (default) | green |
| `a`  | resolved, took left | green |
| `b`  | resolved, took right | green |
| `m`  | auto-merged | yellow |
| `r`  | manually resolved | yellow |
| `c`  | conflict | red |

- `R` — mark current item's tree as "resolved" (`r`)

### Auto-merge to center
- `m` — auto-merge current item into the middle (parent) directory using
  `diff3 -m`; marks result as merged (`m`) or conflict (`c`) if conflicts
  exist
- `M` — same, but for all selected items

Merge logic (mirrors Python `Model3.__merge_individual_item`):
- All three present, all files, no conflicts → run `diff3 -m`, write to
  middle, mark `m`
- Conflict detected (`diff3 -x` produces output) → mark `c`, leave for
  manual resolution
- One or both children absent → copy or delete middle as appropriate

---

## Priority 8 — External tool integration

Ported from Python. Diff color themes (vimdiff done, ediff open) added
2026-07-19 — not ported from Python, a new idea from comparing against
Araxis Merge.

- **Emacs/ediff support** — `FileMergeEmacs.py` exists but is unported.
  2-way: `emacs --eval "(ediff-files \"left\" \"right\")"`.
  3-way: `emacs --eval "(ediff3 \"left\" \"middle\" \"right\")"`.
- **`--merge` CLI flag** — choose between `vim` (default) and `emacs`
- **Generalize beyond hardcoded vim/emacs** — let `[merge]` in the
  Priority 9 TOML config accept an arbitrary external command template
  (with placeholders for the two/three paths), not just a `vim`/`emacs`
  enum. Lets anyone plug in neovim, helix, `code --diff`, or anything else
  diff-capable without umerge needing bespoke support for each one. Small
  change — the launch mechanism already exists for two tools, this is
  mostly a config-shape change in `mergetool.Command`.
- **Diff color themes — vimdiff done ✅ (2026-07-19), ediff still open.**
  vimdiff's default `DiffAdd`/`DiffChange`/`DiffDelete`/`DiffText` colors
  are unrelated to umerge's own tree palette, which made the jump from
  browsing the tree into a file feel like switching apps. Resolved the
  design question this entry used to leave open ("from memory, unverified
  — the user will supply actual RGB values from Araxis Merge"): rather
  than inventing a *separate* Araxis-flavored palette, `vimCommand`
  (`internal/mergetool/mergetool.go`) now reuses umerge's own directory
  colors directly — `DiffChange`/`DiffText` get `styleChanged`'s blue
  (`#a6caf0`), `DiffAdd` gets `styleUnique`'s green (`#c0dcc0`), matching
  what those hues already mean in the tree (changed vs. present-on-some-
  sides). `DiffDelete` (the filler for lines only the *other* buffer has)
  gets a plain neutral gray rather than a third color, since umerge itself
  never highlights an absent side — it's just left blank. This is the same
  principle Araxis Merge uses (one consistent palette across its directory
  and file views), just sourced from umerge's own colors instead of a
  separate imported palette. Applied via extra `-c "highlight ..."` args
  at launch time in `vimCommand`, not by editing the user's `.vimrc` — only
  affects umerge-launched sessions. `ctermbg` values are the closest
  xterm-256 approximations for terminals without true-color; `guibg`/
  `guifg` carry the exact hex. Verified by running the constructed
  `-c` flags against the real `vim` binary headlessly and reading back
  `:highlight` output to confirm each group landed exactly as intended, on
  top of unit tests in `mergetool_test.go` pinning the exact command-line
  built for one/two/three files.

  **Still open:** the same treatment for ediff
  (`ediff-current-diff-A/B/C`, `ediff-fine-diff-A/B/C`, the 3-way `-C`
  variants) — not done this pass, only vim was asked for. **Generalize
  beyond hardcoded vim/emacs** (below) and the config-file override in
  Priority 9 (`[theme.vimdiff]`/`[theme.ediff]`) are also both still open;
  the colors are hardcoded constants in `mergetool.go` for now, not yet
  user-overridable.

---

## Priority 9 — Configuration & theming

Command-line flags and the system/user config-file split are ported from
Python; the file format (TOML, not INI) and the theming scope are new
decisions for the Go version, made explicit here so Priority 8's and
Priority 10's color work has one real home instead of staying hardcoded.

### Command-line flags
| Flag | Description |
|------|-------------|
| `-c` / `--colors` | color depth: `auto`, `256`, `8`, `none` |
| `--merge` | merge tool: `vim` (default) or `emacs` |
| `-U` / `--unicode` | force Unicode tree symbols (`▶`/`▼`) — the default |
| `-A` / `--ascii` | fall back to ASCII tree symbols (`>`/`v`) |

**Default flip — ✅ DONE (2026-07-18):** brought back the Unicode
collapse/expand arrows (▶ U+25B6 / ▼ U+25BC) as the default — they look
noticeably better than the ASCII `>`/`v` fallback, and most terminals
(confirmed: WezTerm) render them at the correct width. `CLAUDE.md`
documented *why* umerge had switched to ASCII in the first place: those
characters have "Ambiguous" East Asian Width and some terminals render
them at the wrong column width, shifting text — the same underlying bug
class as the CJK-filename misalignment diagnosed the same day (confirmed
to be a COSMIC-terminal rendering bug, not a umerge one). Rather than
staying on ASCII everywhere to dodge a COSMIC-specific bug, defaulted to
the better-looking Unicode arrows with `-A`/`--ascii` as the escape hatch.

Implementation note: the byte-slicing in `renderCell` (added for the
arrow-color fix) assumed the arrow was always 2 *bytes* — true for ASCII
`"> "` but not for `"▶ "`, which is 4 bytes (▶ is a 3-byte UTF-8 sequence).
Had to make the byte-length arrow-agnostic (computed from which mode is
active) rather than hardcoded, or it would have corrupted the split for
every directory row once Unicode became the default. Caught before it
shipped — verified with new unit tests for both symbol sets plus a live
capture confirming clean rendering (no corruption) with real CJK directory
names. `--help`, README, and `umerge.1` (SYNOPSIS, OPTIONS, and "Tree
symbols") all updated to document `-A`/`-U`.

### Config file — `~/.umergerc.toml`
Deliberate departure from the Python version's INI format — TOML is a
better fit for structured/nested theme data and is idiomatic for Go CLI
tools. Rough shape below; the actual schema gets finalized at
implementation time, not frozen here:

```toml
[colors]
depth = "auto"          # auto | 256 | 8 | none

[merge]
tool = "vim"             # vim | emacs | a custom command template
                          # (see Priority 8's "generalize beyond
                          # hardcoded vim/emacs")

[theme.umerge]
# umerge's own tree-view palette: same/changed/inserted/removed,
# selection highlight, hidden-item dimming, resolution-status marker
# colors, separator coloring — see Priority 10 for the specific cases.

[theme.vimdiff]
# overrides for the built-in vimdiff palette (matches umerge's own tree colors — Priority 8).

[theme.ediff]
# overrides for the built-in ediff palette, once implemented (Priority 8).
```

- Optional system-wide defaults at `/etc/umerge.toml`; `~/.umergerc.toml`
  overrides those; CLI flags override both. System-wide config is
  low-priority given this is a personal/hobby-scale tool for now.
- The Araxis-style palettes from Priority 8 ship as the built-in defaults
  for `[theme.vimdiff]`/`[theme.ediff]` — the config file is how a user
  *overrides* them, not the only way to get them.

---

## Priority 10 — Coloring refinements

Ported from Python.

### Directory arrow color — ✅ DONE (2026-07-18)
**Bug found and fixed**, reported as "directory names are yellow, which
seems wrong." It was right — the whole directory row (arrow *and* name)
was rendered yellow via a flat `styleDir`. Checked against
`Settings.py`: Python's `dir_arrow_fg` is 226 (yellow) in *every* category
(normal/changed/inserted/removed), but `filename_fg` varies by category
independently — only the little collapse/expand arrow glyph is meant to
be yellow, as a consistent "this is a folder" cue; the name itself should
read the same color a file's name would in that category. Fixed:
`styleDir` renamed to `styleDirArrow` (arrow-only accent, not a whole-row
style); `rowCols` no longer special-cases `IsDir` for the base column
style (a present-everywhere directory now correctly falls through to
`styleNormal`, matching a file); a new `renderCell()` in `app.go` splits
a directory's fitted text into indent/arrow/rest and applies the yellow
foreground *only* to the arrow segment, preserving whatever background
the row's actual status style has (so green/blue/cursor rows keep their
background under the arrow too — this also matches Python, whose
`*_dir_arrow_bg` varies per category while `*_dir_arrow_fg` stays 226
throughout). Skipped for `CompareError` rows — an error should read as
one uniform red line, not a yellow arrow on red. Verified with unit tests
(forcing `lipgloss.SetColorProfile(termenv.ANSI256)` since `Render()`
strips color outside a real terminal) plus a live-pty smoke test
confirming the actual rendered SGR codes: yellow only around the arrow,
plain white around the name.

### 3-way partial presence (one or two sides absent)
The Python applies more nuanced colors than a blanket "green for absent":

| Sides present | Current Go | Python behavior |
|---------------|-----------|-----------------|
| Left + right only (no parent) | green both | green both (INSERTED) |
| Parent only (no children) | green parent | **magenta/purple** (REMOVED) |
| Parent + one child, same | green both | white (no difference) |
| Parent + one child, different | green both | **blue** (CHANGED) |

The last two cases require running a 2-way diff even when one side is
absent, which the current code skips.

### Separator coloring — ✅ DONE (2026-07-18), deliberately not matching Python
Implemented as `separatorStyle()` in `app.go`: a separator colors to match
its two adjacent columns only when **both** share the same background;
otherwise it stays the neutral gray default. **Deliberately different from
Python**, which always colors a separator to match the column on its
*right* regardless of whether the left side matches (`__fixed_compute_colors`
in `View3.py`: `new_colors[1] = colors[1]` — the left separator takes the
*middle* column's color unconditionally). That always-match-the-right-side
rule implies a boundary that isn't really there when the two sides
actually differ; matching only on agreement reads as more intentional.
Also fixes cursor rows as a side effect — previously even the
cursor-highlighted row had a flat gray separator breaking up the
highlight; now it blends since both sides are `styleCursor`. Verified with
unit tests (`separatorStyle`, comparing via `GetBackground()` since
`lipgloss.Style` isn't `==`-comparable and `Render()` strips color outside
a real terminal) plus a live-pty smoke test inspecting the raw ANSI bytes
directly to confirm the green/gray boundary lands exactly where expected.

**Follow-up bug found and fixed 2026-07-18: separators between two plain
rows rendered white instead of gray.** Root cause: `GetBackground()`
returns `lipgloss.NoColor{}` for a style with no background set (e.g.
`styleNormal`), and the original check only compared for equality — two
unset backgrounds compared equal, so ordinary/unstyled rows were treated
as "sharing a color" and the separator inherited `styleNormal`'s *white*
foreground. Two columns not having a color isn't the same as two columns
sharing one. Fixed by additionally requiring the shared background not be
`NoColor{}` before matching. Verified live: a "same everywhere" row's
separator is now the neutral gray `38;5;240` (was white `97`), while a
genuinely blue (changed) row's separator still correctly blends into the
blue.

**Also confirmed 2026-07-18 (not a umerge bug):** wide CJK filenames
looked misaligned in COSMIC terminal specifically. Checked `fit()`
directly against every demo filename (Chinese/Japanese/Korean/Cyrillic/
Greek/Arabic/German) — `runewidth.StringWidth()` and the padding it
produces are exactly correct in every case, confirmed by direct testing.
Confirmed fine in WezTerm — this is a COSMIC-terminal rendering quirk
(still a fairly new terminal), not something to fix in umerge.

---

## Priority 11 — Scriptability / launch polish

New — supports the "go-to tool" goal, not a Python-port item. Tools like
ripgrep and eza earn a place in people's default toolkits partly by being
useful both interactively *and* in scripts/CI, not just as a TUI.

- Non-interactive summary output mode (e.g. `--summary` or `--json`)
  listing changed/added/removed paths, for piping into scripts or CI —
  distinct from the interactive TUI.
- README positioning pass once Priorities 1–3 land: lead with "terminal
  native directory diff + `git difftool -d` backend," include a comparison
  table against Meld/Beyond Compare/Araxis, asciinema/GIF demo.

---

## Priority 12 — Robustness

Ported from Python (context/cancellation and lazy loading are new
observations from the Go rewrite, not present in the Python version).

- **Cancel background comparison on quit** — the comparison goroutine
  currently runs to completion even after the user presses `q`. Pass a
  `context.Context` so it stops promptly.
- **Error state display** — ✅ done (2026-07-19): `styleError` renders
  `CompareError` entries in red instead of white, copy/delete failures set
  it, and the compare-time path (e.g. permission denied reading a file
  during background comparison) is now covered by tests too:
  `TestCompareTwoFiles_PermissionDenied`/`TestCompareThreeFiles_PermissionDenied`
  (`fileops`), `TestCompareEntry_TwoWayPermissionDenied`/`_ThreeWay...`
  (`compareEntry`), and `TestUpdate_KeyR_PermissionDeniedFileBecomesCompareError`
  (full async goroutine/channel/`Update` pipeline via `drainCompare`). It's a
  static precondition (chmod before comparing), not a race — `filesEqual`'s
  `os.Stat` succeeds on an unreadable file but the subsequent `os.Open`
  fails deterministically, so no goroutine-timing coordination was needed.
  Scope was deliberately kept to "does a real failure get reported
  accurately," not hardening against adversarial/concurrent permission
  changes (TOCTOU races, symlink swaps mid-compare, etc.) — out of scope by
  design.
- **Lazy tree loading** — `BuildPair`/`BuildTriple` currently read the
  entire directory tree eagerly at startup. For very large trees a lazy
  approach (load children on expand) would improve startup time.
