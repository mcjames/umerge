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

## Priority 1 — File operations (baseline usability)

Without these umerge is read-only. Ported from Python. Per the testing
policy: write `t.TempDir()`-based tests for each copy/delete function
before wiring its key-handling into `Update()`.

### Copy (2-way)
- `a` — copy current item left → right
- `b` — copy current item right → left

### Copy (3-way, multi-step prompt)
- `a` → sub-prompt "Copy from A (left) to:" → `b` or `c`
- `b` → sub-prompt "Copy from B (middle) to:" → `a` or `c`
- `c` → sub-prompt "Copy from C (right) to:" → `a` or `b`

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

- **Respect `.gitignore` by default** — skip `.git`, `node_modules`, build
  artifacts, etc. without being asked. This is the single highest-leverage
  item: it's both a general quality-of-life win and what makes umerge feel
  native when run near a git repo.
- **Include/exclude filters** — wildcard and/or regex, plus flags to ignore
  whitespace, case, and blank-line-only diffs when comparing file contents.
- **Fast short-circuit comparison** — use size+mtime (and optionally a
  checksum) to skip full content diff on files that are obviously
  unchanged, instead of always shelling out to `diff`. Needed for umerge to
  stay responsive on large trees.
- **Binary file detection** — report "differ (binary)" instead of running
  text diff/vimdiff against binary content.
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
- Document the `.gitconfig` snippet in the README:
  ```
  [difftool "umerge"]
      cmd = umerge "$LOCAL" "$REMOTE"
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

## Priority 4 — Refresh / re-compare

Ported from Python.

- `r` — re-enumerate subtree rooted at current item, then re-compare it
  (in background, same goroutine+channel pattern already used for initial
  comparison)
- After vimdiff/ediff exits, automatically re-compare the edited entry
  (`toolDoneMsg` currently does nothing)

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

Ported from Python.

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
- **Diff color themes (Araxis-style)** — vimdiff's and ediff's default
  colors (vim: `DiffAdd`/`DiffChange`/`DiffDelete`/`DiffText`; ediff:
  `ediff-current-diff-A/B/C`, `ediff-fine-diff-A/B/C`, and the 3-way `-C`
  variants) are jarringly saturated out of the box, which makes the jump
  from umerge's own muted directory-diff colors to the launched file-diff
  tool feel like a different app. Goal: a built-in palette closer to
  Araxis Merge's default — muted/pastel rather than saturated. **The exact
  palette (e.g. "pale green for insertions, pale red/pink for deletions,
  pale yellow/gold for changed lines, distinct muted shade for word-level
  differences") is from memory, unverified — the user will supply actual
  RGB values from Araxis Merge when this is implemented; use those rather
  than this description.** Apply it
  by injecting extra `-c "highlight ..."` args
  (vim) / extra `--eval` forms (emacs) at launch time in `vimCommand`/
  `emacsCommand` — not by editing the user's `.vimrc`/`.emacs` — so it
  only affects umerge-launched sessions. Ships as the built-in default for
  `[theme.vimdiff]`/`[theme.ediff]` in the Priority 9 config file, so a
  user who wants a different theme overrides it there rather than editing
  Go code.

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
| `-A` / `--ascii` | force ASCII tree symbols (already using ASCII; expose as flag) |
| `-U` / `--unicode` | force Unicode tree symbols |

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
# overrides for the built-in Araxis-style vimdiff palette (Priority 8).

[theme.ediff]
# overrides for the built-in Araxis-style ediff palette (Priority 8).
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

### Separator coloring
Python colors the `|` separators to match the content on their right side
(e.g., `CHANGED_VERTICAL_SEP` when the adjacent column is blue).
Currently a fixed gray.

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
- **Error state display** — entries that fail to compare (e.g., permission
  denied) should render in an error color rather than staying white.
- **Lazy tree loading** — `BuildPair`/`BuildTriple` currently read the
  entire directory tree eagerly at startup. For very large trees a lazy
  approach (load children on expand) would improve startup time.
