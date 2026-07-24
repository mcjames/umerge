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

See also **`POLISH.md`** — UX/trust/discoverability findings (in-app help,
search, delete confirmation, silent permission/symlink swallowing,
distribution) that don't map onto a numbered priority yet, split out
2026-07-23 to keep this file focused on the versioned roadmap.

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

**Reordered 2026-07-19** after re-scanning the Python source for
feature-parity gaps: Selection, Hidden items, and the 3-way merge workflow
were the only remaining Python-original features with zero implementation
(no key handling *and* no supporting fields on `entry.Entry`), so they're
promoted to Priority 2–4, ahead of the filtering/external-integration work
that was previously next in line. A few items below were also found to be
stale (already implemented, still listed as open) and corrected in place.

---

## 1.0 release scope (decided 2026-07-23)

Deliberate decision to scope a real 1.0 rather than let this stay a
perpetually-pre-1.0 hobby project. Bar: **match the Python original's
actual (working) functionality on a real, messy tree, plus what's already
been shipped beyond it** — not "every enhancement idea in this file."

**What's left before 1.0:**
- **Priority 2 — Selection, core only.** `s` toggle (propagate down the
  no-holes subtree), bulk `d`/`a`/`b`/3-way-copy acting on a selection.
  **`S` (bulk-select-by-rule) is explicitly deferred, not a parity loss**:
  checked Python's actual implementation (`Match3.selection_matches`
  ignores both its arguments and hardcodes one answer; the controller only
  wires up the `a`/absent branch, `u`/`c`/`i` just print a message and do
  nothing) — Python's own bulk-select-by-rule doesn't work, so skipping it
  ships the one part of Python's selection model that actually did.
- **Priority 4 — 3-way merge workflow**, in full: `m`/`M`/`n`/`R`,
  resolution-status markers, the `diff3 -m`/conflict classifier.

**Done, closing out this bar:**
- **`git difftool -d` end-to-end** — ✅ manually verified 2026-07-23 by
  hand against a real repo: launches cleanly, worked "wonderfully."
  Not exhaustively fuzzed, but confirmed sufficient for basic real use,
  which is the bar here — see Priority 6.
- Hidden items (Priority 3) — the third zero-implementation Python feature
  identified by the 2026-07-19 audit — shipped 2026-07-22.

**Explicitly deferred to post-1.0 (not gaps, decisions):** Priority 3b
(focus-on-diffs mode — new idea, not a Python feature), Priority 5's
include/exclude filters and rename/move detection, Priority 6's Mercurial
support and TUI file-manager hook docs, Priority 8's ediff color theming
and generalized merge-tool config, Priority 9's `--colors` depth flag and
the `~/.umergerc.toml` config file (Python had INI theming; not required
to match its actual working behavior), Priority 10's minor 3-way
partial-presence color nuance (blanket green vs. Python's couple of
edge-case shades — cosmetic, not a missing capability), Priority 11
(scriptability), and Priority 12's cancel-on-quit/lazy-loading (both
explicitly new-to-Go concerns, not things the single-threaded Python
version ever had to address).

---

## Priority 0 — Testing catchup — ✅ DONE (2026-07-18)

One-time backfill for code that predates the testing policy above. Ordered
by leverage: the tree-diff core first, since Priority 5 is about to modify
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
  conflict marking in Priority 4.

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
  cursor item) depends on Priority 2, not built yet — for now `d` always
  acts on just the cursor item.

### Delete
- `d` — delete current item (both/all sides); if a selection exists, delete
  all selected items instead

---

## Priority 2 — Selection

**Clean-room Go design — Python is inspiration only, not a spec.**
`entry.Entry` currently has no `Selected` field at all, so this is
unstarted at the data-model level. Promoted 2026-07-19 (re-scanned the
Python source for feature parity gaps); redesigned from scratch
2026-07-21 after finding real bugs in the Python reference (below);
revised again same day after further thought about the holes design (see
"Primary design" vs. "Fallback design" below) — the holes idea traded
away a whole feature (bulk copy) for a UX gesture ("select all but X")
judged less important than expected once actual usage patterns were
considered.

- `s` — toggle selection on current item; propagates the new value down
  to the whole subtree (selecting or deselecting a directory applies to
  all its current descendants). This part is common to both designs
  below.
- Selected items rendered in a distinct highlight color.
- Selection is **tree-wide, not cursor-scoped**: bulk operations act on
  every selected node anywhere in the tree, not just ones near the
  cursor.
- Bulk operations: `d` (delete) and bulk copy (`a`/`b`/3-way copy
  prompts) both operate on the selection when one exists, rather than
  just the cursor item.

### Primary design: no holes — partial deselect is blocked (decided 2026-07-21)

**Expected usage shape drove this decision.** Reconsidering the original
holes design below: the realistic way this tool gets used is *iterative
and interactive* — select a small piece of the tree, copy/delete/merge
it, look at the result, select the next piece, repeat — not "carefully
construct one big selection with several carved-out exceptions, then fire
a single irreversible bulk action and hope the reasoning was right." The
latter is exactly the shape where a mistake is most costly: delete and
copy are not undoable, and the more intricate the mental model needed to
predict what a bulk action will do, the higher the odds of getting it
wrong on a real tree. Optimize for the common iterative case, not the
rare big-batch one.

**Invariant:** a directory's `Selected` flag always means "100% of this
subtree is selected" — no exceptions, ever. Concretely: pressing `s` on a
node that is a descendant of an already-selected ancestor is a **no-op**
— flash "deselect the containing directory first" (reusing the existing
flash mechanism) instead of silently doing nothing or creating an
inconsistency. To exclude one item from a bulk selection, the user
deselects the ancestor and selects the other children individually — more
keystrokes for a wide directory, but zero new mental model and no risk of
an ancestor's selected state lying about what's actually underneath it.

**Why this is worth the extra keystrokes:**
- Bulk delete is trivial and needs no safety carve-out: walk the tree,
  `rm -Rf` any maximal selected subtree, stop and don't recurse beneath
  it. Nothing to check for holes because holes can't exist.
- Bulk copy is no longer punted — it's the identical maximal-subtree walk
  with `cp -R` instead of `rm -Rf`, fully symmetric with delete. This was
  blocked entirely under the holes design (see Fallback below); here it's
  not extra work at all, it falls out of the same algorithm.
- No hole-tracking bit, no incremental up-propagation needed for
  correctness. `Parent *Entry` is still worth adding for Priority 3's
  focus mode, but selection no longer depends on it for anything
  safety-critical.
- Optional, purely cosmetic: a tri-state indicator on collapsed
  directories (empty / partial / full — the standard VS Code/Explorer
  checkbox-tree convention) so a collapsed directory containing selected
  descendants doesn't look indistinguishable from an empty selection.
  Since nothing depends on this for correctness, it can be built later,
  or skipped, without affecting delete/copy safety either way.

### Fallback design: selection "holes" (parked, not primary)

Kept here in case the primary design's tedium (deselect-ancestor-then-
hand-pick-the-rest) turns out to bite in practice on wide directories, and
"select all but X" as a single gesture turns out to matter more than the
usage-pattern read above predicted.

Selecting a directory selects its whole subtree, but the user can
navigate into any descendant and press `s` there to deselect just that
one subtree — without disturbing the ancestor's own selected state. That
descendant is now a **hole**: the ancestor is still "selected" as far as
its own flag goes, but no longer uniformly so underneath. Bulk delete
must never delete a holed ancestor wholesale (an `rm -Rf` would destroy
the hole's contents too, which the user deliberately carved out to keep).

**Bulk delete algorithm:** walk the tree top-down. At each node: if it is
selected *and* no descendant anywhere beneath it is unselected (the whole
subtree is uniformly selected — no holes), delete it wholesale using the
existing Priority 1 per-item delete-and-splice, and stop — nothing left
to check underneath. Otherwise (node unselected, or selected-but-holed),
leave it alone and recurse into its children, applying the same test at
each level. The nodes actually deleted are the maximal fully-selected
subtrees; holes and anything outside the selection survive untouched.

**Rendering needs shared infrastructure with focus mode.** To show which
selected directories contain holes without expanding them (distinct
highlight color for "fully selected" vs. "selected with holes," not just
a single SELECTED color), a directory needs a persistent "has an
unselected descendant" bit, maintained incrementally (walking up from
whichever node was just toggled, via the same `Parent *Entry` pointer
Priority 3 already needs) rather than recomputed by a full subtree walk
every render frame.

**Bulk copy is materially harder here and would stay out of scope even if
this design were revived** ("this is a hobby project, not a CS thesis").
Delete's skip-and-recurse works because deleting the pieces independently
is correct. Copy isn't symmetric: preserving a hole on copy means the
destination directory has to be created and then *selectively* populated
with only the selected children — a filtered recursive merge,
structurally closer to Priority 4's 3-way merge walk than to Priority 1's
`cp -R`. A selection containing any hole would need to flash "Bulk copy
with a partial selection isn't supported yet" and do nothing, rather than
either building the filtered-merge logic or silently doing an unfiltered
`cp -R` that would wrongly bring the holes' contents along. (This is the
concrete cost that primarily disqualified this design as the default —
see "Primary design" above.)

### 3-way only
- `S` — multi-step prompt: choose column (A/B/C), then choose feature
  (absent / unchanged / changed / inserted); selects matching items.
- **Additive, not destructive** (decided 2026-07-21, diverges from
  Python): `S` sets `Selected = true` on every node matching the chosen
  column/feature and leaves every other node's selection state exactly as
  it was. It must never clear or overwrite an existing manual `s`
  selection — the whole point of manual picks plus pattern-matched
  bulk-adds is that they compose. Under the primary (no-holes) design,
  this composes the same way but is still subject to the no-partial-
  deselect rule elsewhere — `S` only ever sets flags to `true`, so it
  can't create a hole either way.
- `x` — clear the entire selection (this one *is* a full reset, not
  additive — that's its whole job).

### Bugs found in the Python source — confirms clean-room, not a port
Python's implementation should not be consulted as a reference for this
priority beyond the high-level UX shape (select/deselect, highlight, bulk
`d`, the `S` pattern-select idea). Specific bugs found, not to be
inherited:
- `Match3.selection_matches` ignores both of its parameters — always
  returns "present in left, absent in middle" regardless of the column/
  feature the user actually chose in the `S` prompt. Looks like
  unfinished code.
- `Model3.__delete_selected_aux` (`Model3.py:461-469`) recurses the tree
  and only assigns its `return_value` local inside the `if item.selected`
  branch, but unconditionally does `return return_value` at the end —
  an `UnboundLocalError` crash on essentially every real bulk-delete call,
  since the function is first invoked on the tree root, which is
  essentially never itself selected. It also separately re-deletes
  already-covered selected children after deleting their selected parent
  (harmless in practice only because `rm -f` is idempotent on missing
  paths, but sloppy). Neither behavior — nor the "holes get destroyed"
  semantics that would follow from a literal port — is worth carrying
  forward; both designs above replace this entirely.

---

## Priority 3 — Hidden items — ✅ DONE (2026-07-22)

Ported from Python. Promoted 2026-07-19 (re-scanned the Python source for
feature parity gaps) — `entry.Entry` had no `Hidden` field, unstarted at
the data-model level.

Split into two separate items 2026-07-22, at the point of implementation:
this priority is the core ported feature only; the "focus on diffs"
extension designed alongside it (2026-07-21) is its own item below,
**Priority 3b**, not yet built — it was never required to close this one
out, just designed in the same sitting because the two share rendering
machinery.

**Done:** `Entry.Hidden` (`internal/entry/entry.go`), `Entry.SetHidden`
(propagates down through `Children`, mirroring Python's
`toggle_hidden`/`set_hidden` — a descendant can later be un-hidden on its
own without disturbing an already-hidden ancestor's flag, since `Hidden`
is a plain per-entry bool, not aggregated). `h` toggles the cursor item's
subtree; `H` toggles `Model.renderHidden` (default `false`, matching
Python's `self.render_hidden = False`).

**`Flatten` gained a `skip func(*Entry) bool` parameter** (was
`Flatten(entries)`, now `Flatten(entries, skip)`) — the "exact same
skip-predicate mechanism" Priority 3b's design already assumed it would
reuse. Checked against the actual Python traversal
(`View3.py:__next_display_line_aux`/`__should_render_item`) before
implementing: hiding a node only omits that one display line, it does
**not** structurally stop recursion into its children — a directory's
subtree normally disappears as a whole only because `SetHidden` already
propagated the flag to every descendant, not because the traversal
special-cases a hidden ancestor. `Flatten` mirrors this: `skip` gates
whether an entry is appended to the output, recursion into `Children` is
governed only by `Collapsed`, same as before. This split matters for
Priority 3b too — its "clean subtree collapses to one dimmed line"
behavior is a *recursion*-level rule, not a line-level `skip` — so
`Flatten` deliberately keeps those as two different levers rather than
conflating them into one combined boolean, which is what let this land
without needing to redesign `Flatten` again once 3b is built.

**Rendering — category-preserving dim, decided over a flat override; two
attempts.** Checked Python's actual runtime color selection
(`View3.py:__compute_colors`) before choosing: its `hidden` check
short-circuits before the category checks, so in practice every hidden
entry there renders in one flat color regardless of category — even
though the file *also* defines a full set of per-category hidden pairs
(`normal_hidden`/`changed_hidden`/`inserted_hidden`/`removed_hidden` in
`View3.py`) that the color-selection logic never actually reaches. Decided
against replicating the flat-override behavior: a hidden-but-changed entry
is still worth knowing it changed.

First attempt (2026-07-22): `styles[i].Faint(true)` layered on top of
whatever category style was already picked — no new palette needed, but
reported back as hard to see in practice (the ANSI faint attribute is a
subtle effect, and some terminals barely render it differently at all).
Replaced same-day with explicit darker-but-same-hue background variants
plus solid medium-gray (256-color `245`) text: `styleHiddenNormal`
(no background, matching plain `styleNormal`'s own lack of one),
`styleHiddenUnique` (`#16261c`, a dark green), `styleHiddenChanged`
(`#17233d`, a dark navy), `styleHiddenError` (`#35141a`, a dark maroon) —
each a much-darkened version of its pastel counterpart
(`styleUnique`/`styleChanged`/`styleError`), not a single shared hidden
color.

**Priority order between Hidden and the cursor — reversed once, based on
feedback.** First implementation had `e.Hidden` checked before `isCursor`
in both `rowCols` and `diffStyleForCol`, on the reasoning that a hidden
entry should keep reading as hidden even under the cursor. User reported
back that this made the cursor itself hard to spot on a hidden row (no
visual difference between "cursor here" and "not cursor" once hidden).
Reversed same-day: `isCursor` now wins over `e.Hidden` in both functions
— **the cursor row looks identical whether or not the entry under it is
hidden**, matching a plain cursor row exactly; moving the cursor *off*
an entry is how you tell it's hidden, not a color change on the cursor
row itself. `renderCell`'s yellow directory-arrow accent follows the same
rule — gained an `isCursor` parameter so the arrow is only suppressed for
a hidden row when it's *not* the cursor row (a hidden cursor row keeps
its normal yellow arrow, same as any other cursor row on a directory).
Confirmed live via pty: the exact same SGR bytes
(`38;5;226;48;2;42;93;176`, i.e. `styleCursorChanged`) render whether the
cursor lands on `c.txt` before it's ever hidden, or after being hidden
and revealed via `H` — a non-cursor hidden row still shows the distinct
dark-navy/gray look in between.

**`Parent *Entry` added to `Entry` now, as prep for Priority 3b** — not
needed by Hidden items itself (propagation is downward through
`Children`), but explicitly the "structural prerequisite" 3b's own design
already called for, added here so that work doesn't have to touch
`BuildTree` again later. Wired in `BuildTree` (`entry.go`) for entries
created within its own recursion; `ops.go`'s `rebuildChildren` needed a
matching one-line fix — it calls `BuildTree` directly for a subtree
rebuild after a copy/refresh and reassigns the result to `e.Children`,
but `BuildTree` only wires `Parent` for descendants created *within* its
own recursion, not for the top-level slice it hands back to a caller, so
without this fix `Parent` would have gone stale (nil) after every
directory copy or `r` refresh.

**Interaction with the existing flat-index cursor model, found while
testing, not a bug:** hiding the entry the cursor is on removes it from
`m.flat` immediately (same as delete), so — per `CLAUDE.md`'s scroll
model, `cursor` is a flat-list index, not a reference to the entry itself
— there's nothing left for a second `h` press to reach and undo. Unlike
delete this is meant to be reversible, so this is worth knowing: reversing
requires `H` (reveal hidden entries) first, then `h` again on the now-
visible entry, not pressing `h` twice in place. Verified both the direct
propagate-down case and this reveal-then-unhide path with unit tests.

Verified with unit tests (`entry_test.go`: `Parent` wiring, `SetHidden`
propagation, an independently-un-hidden descendant, `Flatten`'s skip
predicate omitting a line without blocking recursion into its children;
`app_test.go`/`ops_test.go`: `h`/`H` key handling, the reveal-then-unhide
path above, the rebuilt-children `Parent` fix, and the dark-background/
gray-text rendering including the cursor-override case) plus live-pty
smoke tests against the real compiled binary confirming the full round
trip: `a.txt` drops out of view on `h`, reappears on `H` with the
captured raw SGR bytes showing medium-gray text (`38;5;245`) and, for a
changed entry, the dark navy background (`48;2;23;35;60`) rather than the
normal pastel blue — including with the cursor moved directly onto the
revealed row, confirming it stays dark/gray rather than reverting to the
cursor's yellow. `--help`, README, and `umerge.1` updated for `h`/`H` and
the dark-shade-plus-gray-text color note.

---

## Priority 3b — "Focus on diffs" mode (extension, not yet implemented)

New idea, not from the Python version — an enhancement discovered while
thinking about `git difftool -d` on a genuinely huge tree (Linux kernel,
Firefox), where only a handful of files/directories actually differ and
wading through the whole tree to find them defeats the point of the tool.
Designed alongside Priority 3 (Hidden items) because the two share most
of their rendering machinery (see "shared mechanism" below), even though
conceptually they're different: `Hidden` is a user-set fact; "has a diff"
is a computed aggregate. Split into its own numbered item 2026-07-22 —
see Priority 3's note — since it's materially more work than Hidden items
itself and was never a prerequisite for it.

**Data model.** Two independent booleans per conceptual axis — do not
conflate:
- `Hidden` (Priority 3) — user-managed, explicit, static once toggled.
- Subtree diff state — *derived*, bottom-up from the leaves, and
  **not fully known until comparison finishes**. Presence mismatches
  (file/dir absent on one side) are known instantly at tree-build time
  (`BuildTree` is eager/synchronous, per `CLAUDE.md`). Content mismatches
  only become known as `compareResultMsg`s trickle in from the background
  `walkAndCompare` goroutine (`internal/ui/compare.go:40-49`), one file at
  a time — on a kernel-sized tree that walk takes real time. So this is
  **not a plain bool**: model it as tri-state per subtree — diff-found /
  confirmed-clean / still-pending. A directory can't honestly claim
  "clean" until every comparable descendant has reported in.

**Structural prerequisite — ✅ already in place (done alongside Priority
3, 2026-07-22).** `Parent *Entry` now exists on `entry.Entry`, wired in
both `BuildTree` and `ops.go`'s `rebuildChildren`. Efficient up-propagation
still needs to be built on top of it: on each `compareResultMsg`, walk up
from the leaf via `Parent` OR-ing the dirty bit into each ancestor,
**stopping as soon as an ancestor is already marked dirty** (its ancestors
are transitively dirty too — free early exit). For "confirmed clean," each
directory also needs a pending-descendant counter, initialized at build
time (count of comparable leaves under it) and decremented as results
arrive up the parent chain; it flips to confirmed-clean the instant the
counter hits zero with no dirty bit set. Recomputing the aggregate
top-down from scratch on every message instead of maintaining it
incrementally would be O(n²) on a 70k-file tree — not viable at the scale
this feature exists for.

**Visibility rule — corrected 2026-07-22 against `Flatten`'s actual
implementation** (built for Priority 3, `entry.go:193`). The original note
here proposed one combined boolean, `visible = !(hidden && !renderHidden)
&& !(focusMode && subtreeClean)`, reused as a single `Flatten` skip
predicate. That's wrong for this feature specifically: Hidden's rule is a
**line-level** omission (skip just this entry, keep recursing into its
children — see Priority 3's done-note on why), but "clean directories
auto-collapse... instead of disappearing" (below) is a **recursion-level**
rule — the directory's own line must still render, only its children
should stop being visited. Those can't be the same predicate. `Flatten`
already anticipates this split: `skip` (line omission, Hidden's job) and
recursion (today gated only by `Collapsed`) are independent levers, so
this feature's job is to add a second recursion-gating condition
alongside `Collapsed` — e.g. `!e.Collapsed && !(focusMode &&
subtreeClean)` — not to fold a new bit into `skip`.

- **Explicit hide always wins** (decided 2026-07-21): if the user
  manually hid a subtree with `h` and it happens to contain a diff, focus
  mode does not override that — the user is in control.
- **Clean directories auto-collapse under focus mode rather than
  vanishing** (decided 2026-07-21): unlike `Hidden`, which removes entries
  from the flat list entirely, a subtree confirmed clean under focus mode
  collapses to one dimmed line instead of disappearing, to preserve a
  sense of tree structure on a huge comparison ("4,000 clean files lived
  under here") rather than silently missing chunks of the tree.
- **Pending subtrees default to visible** until proven clean. This makes
  focus mode's view shrink monotonically as the background scan confirms
  subtrees clean — never pops items *into* view, which would be the more
  surprising/worse direction. On a huge tree this produces a good emergent
  behavior for free: turn focus mode on and watch the visible tree
  progressively narrow down to just the files that actually differ as the
  scan completes.

**Keybinding.** `f` — toggles focus mode globally (decided 2026-07-21).
Considered and parked: a subtree-scoped "focus only within this
directory" variant. No clear, non-confusing way to signify
"focused-within-here" vs. "not-focused" at the boundary was found —
revisit only if a concrete need for it shows up later.

**Performance risk to remember at implementation time.** Today,
`compareResultMsg` handling (`app.go:155-160`) mutates the entry in place
and never calls `reflatten()` — visibility never depends on compare
results today, so it doesn't need to. Once focus mode exists, a compare
result *can* change visibility, so `Update` needs to reflatten while focus
mode is active — but `Flatten` rebuilds the whole visible slice via
recursive append (`entry.go:193`), so reflattening on every one of
(potentially) tens of thousands of streamed messages is the same O(n²)
trap as the propagation logic above. Needs throttling (e.g. only
reflatten on an ancestor's pending→resolved transition, or batch on a
short ticker) rather than reflattening unconditionally per message.

**Status line, designed alongside (also not yet implemented).** Since the
propagation logic above already has to track clean/pending/dirty counts
per subtree, the root's counts fall out for free — no extra bookkeeping.
Extends the existing status-bar priority chain
(`app.go:430-441`, currently prompt → flash → default hints line) with one
more layer:
- New `showCounts bool` on the model. Three write sites: set `true` the
  moment comparison starts; forced back to `false` the moment
  `compareDoneMsg` fires (unconditional reset — no need to track whether
  the user had manually overridden it); flipped by the toggle key at any
  time in either phase.
- Default while `showCounts` is true: `1,842 clean · 12 pending · 3
  differ` (the leading `%d/%d` cursor-position prefix stays as-is either
  way). The `pending` segment is dropped entirely once its count reaches
  zero, rather than showing "0 pending".
- Default while `showCounts` is false: today's hints line
  (`q quit  ←→/enter collapse  ...`), unchanged.
- `prompt`/`flash` still take priority over both — they're transient,
  higher-urgency signals; this new layer only governs the "nothing else
  going on" default slot.
- Toggle key: **`t`** (decided 2026-07-21).
- Hidden-but-dirty entries still count toward "differ": the counts
  describe the tree's actual state, independent of what's currently
  rendered, since explicit-hide-always-wins (above) means the rendered
  view can disagree with the true count.

---

## Priority 4 — 3-way merge workflow

Ported from Python. Promoted 2026-07-19 (re-scanned the Python source for
feature parity gaps) — `entry.Entry` currently has no resolution-status
field, unstarted at the data-model level.

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
- `n` — auto-merge the **entire tree** to center in one keystroke
  (`Model3.merge_all`, literally `m` applied to the root item). Missed in
  the original version of this TODO; found during the 2026-07-19
  feature-parity scan of `Controller.py`.

Merge logic (mirrors Python `Model3.__merge_individual_item`):
- All three present, all files, no conflicts → run `diff3 -m`, write to
  middle, mark `m`
- Conflict detected (`diff3 -x` produces output) → mark `c`, leave for
  manual resolution
- One or both children absent → copy or delete middle as appropriate

### How the mechanic actually works (recap, 2026-07-19 — write this down
because it's easy to forget between sessions)
Middle isn't just a display column — **it's the merge target.** `m`
doesn't create output somewhere new; a successful auto-merge overwrites
the middle file/directory in place, using `diff3 -m left ancestor right`.
Recursing into a directory just walks this same per-item decision down
the tree:
- One side added something the other doesn't have → copy it into middle,
  mark `a`/`b` (took-left/took-right). Green.
- Both sides deleted something that existed in middle → delete it from
  middle too, no marker needed.
- All three present, file, changes don't overlap → real `diff3 -m` merge,
  write to middle, mark `m` (auto-merged). Yellow.
- All three present and changes *do* overlap (checked via `diff3 -x`
  first, separately from the actual merge) → **do nothing to the file.**
  Mark `c` (conflict), leave it byte-for-byte as-is. Red.
- Same new file/dir appeared independently on both sides, no shared
  ancestor → also just `c` — can't tell "same idea" from "coincidence."

**Important: umerge never builds its own conflict-resolution UI.** A `c`
row has no in-app conflict-marker rendering or merge editor — the user
hits `Enter` (already built, Priority 7/8) to open all three files in
`vimdiff`/`ediff3`, resolves by hand, saves the middle buffer, and exits.
The classifier above (`a`/`b`/`m`/`c`) is the only new decision logic;
resolving a real conflict is entirely delegated to the external tool that
already exists. This is most of why this priority is smaller than it
first looks.

### Decided: delete/modify is a conflict, not a silent auto-resolve
(2026-07-19) Python: if middle+left both have a file but right deleted it
(or vice versa), it just copies left over middle and marks `a` — no
warning that one side intentionally deleted this. **Decided instead:**
delete-vs-modify is a `c` conflict, same as a real overlapping text
conflict — deletion and modification are both deliberate actions with
real intent behind them, and there's no line-level merge that reconciles
"delete this" with "here's a new version of it." Auto-picking either side
risks silently discarding real work. This is a long-settled question
elsewhere — git calls this exact shape a "modify/delete conflict" and
always stops rather than auto-resolving; Mercurial/Perforce/SVN do the
same. A deliberate divergence from Python (see `feedback_python_not_sacred`
in memory), matching the caution already applied elsewhere in this
project (the silent-copy-from-absent-source fix and the `cp`-parent-dir
fix in Priority 1 were both "don't silently do the convenient thing").

**Resolving it needs no new UI — it reuses what's already shipped.** The
item has `count() == 2` (one side + ancestor present, other side absent),
so the existing Priority 7/8 `Enter` logic already falls back to a 2-way
diff between the surviving side and the ancestor (not the 3-way merge
tool, since there's no third file to diff against) — which is exactly the
right view: it shows what the surviving side changed, so the human can
judge whether it's worth keeping over the deletion. To resolve: `a`/`b`
(whichever side is present) revives it into middle — "keep the edit,
undo the deletion" — or `d` deletes it everywhere — "honor the deletion."
Both are Priority 1 keys, already shipped and tested. Then `R` marks it
resolved (see the still-open question below). Also cheaper to detect than
a real text conflict: pure presence-check (left/middle/right nil-ness) at
compare time, no `diff3 -x`/`diff3 -m` subprocess needed for this case.

### Still open: resolution-status marker doesn't auto-clear after a manual fix
After hand-resolving a `c` in vimdiff and saving, Priority 7's
auto-recompare-on-exit updates the diff counts (0 diffs everywhere, looks
fine) but the status letter stays `c` (red) until `R` is pressed — that's
genuine Python behavior, not a bug, but worth deciding on purpose: keep
the explicit `R` requirement (matches Python, no magic), or auto-promote
to resolved when a post-edit recompare shows zero diffs on all three
pairs (friendlier, but "the tool decided it's fine" is a bigger claim
than "diffs are now zero"). No recommendation yet — flagging for a real
decision before implementation. Same category as the delete/modify
question above: don't silently replicate Python's behavior without
deciding on purpose (see `feedback_python_not_sacred` in memory).

---

## Priority 5 — Filtering & performance ("is this a real tool" bar)

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
  (`ops.go`'s `rebuildChildren`, from Priority 7's `r` key) can re-test
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
  **Design decision (2026-07-19, not yet implemented):** default to
  whatever `diff`/`diff3` themselves default to — exact, case-sensitive,
  whitespace-significant comparison, the Unix way — and make any
  whitespace/case-insensitive behavior an explicit opt-in flag, never the
  default. Considered and rejected: quietly configuring the *launched merge
  tool's* (vimdiff/ediff) own whitespace handling (e.g. `diffopt+=iwhite`)
  — that's the editor's own settings to own, not something umerge should
  impose on someone else's `.vimrc`/`.emacs`; this item is specifically
  about umerge's *own* content comparison (`fileops.CompareTwoFiles`/
  `CompareThreeFiles`), analogous to `diff`'s own `-w`/`-b`/`-i` flags.
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

## Priority 6 — External integrations

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
  when invoked non-interactively by `git difftool -d` — ✅ done (manually
  verified 2026-07-23 against a real repo; not exhaustively fuzzed, but
  confirmed sufficient for basic real use).
- The 3-way mode's real differentiated use case is comparing three
  arbitrary tree *snapshots* (three deploy configs, three `git worktree`
  checkouts of different branches) — not tied to an in-progress git merge.
  Worth calling out explicitly in docs/positioning, since it's a niche use
  case nobody else in the terminal space covers.

**Stale item removed 2026-07-19:** the README already documents the
`.gitconfig` snippet (`--read-only` `git difftool` backend, see below) —
this used to be listed here as still-open, but it was done alongside the
`--read-only` flag itself.

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
`a`/`b`/`c`/`d` (and any future mutating command — Priority 4's `m`/`M`/`R`
once built) show a flash message explaining they're disabled instead of
acting. Everything non-mutating (navigate, collapse/expand, launch
vimdiff/emacs to *view*) still works. The recommended `git difftool`
`.gitconfig` snippet (Priority 6, git integration section above) now uses
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

## Priority 7 — Refresh / re-compare — ✅ DONE (2026-07-19)

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

## Priority 8 — External tool integration

Ported from Python. Diff color themes (vimdiff done, ediff open) added
2026-07-19 — not ported from Python, a new idea from comparing against
Araxis Merge.

- **Emacs/ediff support — ✅ done.** `internal/mergetool/mergetool.go`'s
  `emacsCommand` implements both `(ediff-files "left" "right")` (2-way)
  and `(ediff3 "left" "middle" "right")` (3-way), matching
  `FileMergeEmacs.py`. **Stale item corrected 2026-07-19**: this used to
  say "unported" — it wasn't, that note was just out of date.
- **`--merge` CLI flag — ✅ done.** `main.go` accepts `-m`/`--merge vim|emacs`
  (default `vim`), validated against those two values. **Stale item
  corrected 2026-07-19** — also already implemented.
- **Generalize beyond hardcoded vim/emacs** — let `[merge]` in the
  Priority 9 TOML config accept an arbitrary external command template
  (with placeholders for the two/three paths), not just a `vim`/`emacs`
  enum. Lets anyone plug in neovim, helix, `code --diff`, or anything else
  diff-capable without umerge needing bespoke support for each one. Small
  change — the launch mechanism already exists for two tools, this is
  mostly a config-shape change in `mergetool.Command`.
- **Diff color themes — vimdiff done ✅ (2026-07-19), ediff done ✅ (2026-07-23).**
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

  **ediff done 2026-07-23**, prompted by a friend about to try umerge
  with `--merge emacs` for outside feedback. Verified the actual face
  list against a real installed Emacs (29.3) rather than assuming — batch
  `mapatoms` over every `ediff-`-prefixed face confirmed exactly
  `current`/`odd`/`even`/`fine`-diff × `A`/`B`/`C`/`Ancestor` exist, and
  nothing else (no separate `Ancestor` faces needed since umerge only
  ever calls plain `ediff3`, never the ancestor-merge variant). Important
  finding from that check: **ediff has no equivalent of vim's `DiffAdd`**
  — every diff region, whether an edit or a one-sided insertion, uses the
  same faces, so there's no separate green "unique" treatment to carry
  over; every region gets the blue "changed" hue.

  **Revised same day after the friend flagged a separate but related
  problem: plain `emacs` was popping a full-screen GUI frame** (their
  setup is Spacemacs, whose GUI frame defaults to fullscreen) instead of
  staying in the terminal umerge was already running in — breaking the
  "suspend the display, run the tool, resume" model the man page
  describes, which vim never has a problem with (it stays in its
  invoking terminal by default; a separate GUI frame is a real, unrelated
  binary, `gvim`, that has to be explicitly requested). Fixed by adding
  `-nw` to every `emacs` invocation in `emacsCommand`.

  **That fix, in turn, invalidated part of the color work done earlier
  the same day and caught before it shipped.** The first pass went beyond
  a flat vim-style port: `ediff-current-diff-<letter>` (the hunk the
  cursor is on) got a saturated blue matching the tree's own cursor-row
  treatment, and `ediff-fine-diff-<letter>` (word-level highlighting)
  got the same blue bolded, mirroring `DiffText`. Both were verified
  working — in `--batch` mode, which doesn't render to a real display or
  terminal at all, so it never actually exercises `-nw` rendering.
  Once `-nw` became the real launch mode, live-pty verification against
  an actual terminal session (constructing test cases specifically to
  make the current-diff and fine-diff regions unambiguous — a single
  isolated diff, then a single-word change in an otherwise identical
  line) showed **neither face renders distinctly in a real terminal
  session** — the differing region always just shows the plain
  odd/even background color, even after explicitly forcing
  `ediff-use-faces`/`ediff-force-faces`. Root-caused via ediff's own
  source: `ediff-highlight-diff` (`ediff-util.el`, the function that
  applies the current-diff overlay) is directly docstring'd "Invoked for
  X displays only" — a real, longstanding Emacs/ediff limitation, not an
  umerge bug. Simplified `ediffFaceArgs` to theme only
  `ediff-odd-diff-<letter>`/`ediff-even-diff-<letter>` (verified to
  render as real 24-bit truecolor in an actual `-nw` session) and removed
  the current-diff/fine-diff theming entirely — setting faces that
  provably never render in the mode umerge actually launches in would
  just be dead code implying an effect that doesn't happen. The
  `cursorChangedHex` constant this introduced was removed along with it.

  Implementation: `ediffFaceArgs` in `mergetool.go` builds one `--eval
  (set-face-attribute ...)` per face, prepended before the final
  `ediff-files`/`ediff3` launch eval — same "extra flags at launch time,
  never touches the user's init file" principle as vim's `-c` args.
  Unlike vim (which needs a separate `ctermbg` for terminals without
  true-color), Emacs approximates a hex color to the terminal's actual
  palette itself — confirmed via real `-nw` pty sessions rendering true
  24-bit color escape sequences (`\x1b[48;2;166;202;240m`) matching
  `changedHex` exactly, not just in `--batch` mode. Unit tests in
  `mergetool_test.go` pin the exact `--eval` sequence for both 2-way and
  3-way (including `-nw` itself), plus a dedicated test asserting
  current-diff/fine-diff are *not* touched, so a future well-intentioned
  attempt to "finish" this theming doesn't silently reintroduce dead
  code.

  **Generalize beyond hardcoded vim/emacs** (above) and the config-file
  override in Priority 9 (`[theme.vimdiff]`/`[theme.ediff]`) are still
  open — the colors are hardcoded constants in `mergetool.go` for both
  tools now, not yet user-overridable.
- **`diffopt+=linematch:60` for vimdiff (discussed 2026-07-23, not yet
  implemented).** Came up while weighing whether umerge should adopt a
  modern structural/TUI diff tool (difftastic, delta) as an alternative to
  vimdiff/ediff for viewing. Decided against that: those tools are
  view-only (they format and page diff output, they can't edit), which
  breaks down for exactly the case umerge's own 3-way conflict handling
  depends on — Priority 4 deliberately builds no in-app conflict UI and
  hands real conflicts to vimdiff/ediff for editing, something a pure
  viewer can't do at all. Adding a second default tool just for the
  read-only case would also cut against the muscle-memory concern that
  started this conversation — the user explicitly prefers one tool used
  consistently over several. The better-of-both-worlds alternative: Vim/
  Neovim's `diffopt+=linematch:60` does much smarter intra-hunk line
  matching (closer to difftastic's structural diffing quality) while
  staying the same editor, same keybindings, same edit capability.
  `vimCommand` (`internal/mergetool/mergetool.go`) already injects
  `-c "highlight ..."` flags for color-matching (see above) — adding
  `-c "set diffopt+=...,linematch:60"` alongside them is the same
  mechanism, just one more flag. For anyone who genuinely wants
  difftastic/delta anyway, "generalize beyond hardcoded vim/emacs" (above)
  is the right escape hatch — an opt-in the user configures themselves,
  not a second umerge default.

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

**Still open:** `-c`/`--colors` (color-depth override) is not implemented
yet — umerge currently always renders in the terminal's native color
depth.

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
- README positioning pass once Priorities 5–6 land: lead with "terminal
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
