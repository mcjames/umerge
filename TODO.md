# umerge — remaining Python features to port

This lists features present in the Python version that have not yet been
implemented in the Go/Bubble Tea version.  Roughly ordered by priority.

---

## File operations

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

## Refresh / re-compare

- `r` — re-enumerate subtree rooted at current item, then re-compare it
  (in background, same goroutine+channel pattern already used for initial
  comparison)
- After vimdiff/ediff exits, automatically re-compare the edited entry
  (`toolDoneMsg` currently does nothing)

---

## Selection

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

## Hidden items

- `h` — toggle the user-hidden flag on the current item (and its subtree).
  This is a user-managed "ignore" state, unrelated to dot-files.
- `H` — toggle whether hidden items are rendered at all (`render_hidden` flag)
- Hidden items shown in a dimmed color when rendered; skipped entirely when
  `render_hidden` is false

---

## 3-way merge workflow

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

## External tool integration

- **Emacs/ediff support** — `FileMergeEmacs.py` exists but is unported.
  2-way: `emacs --eval "(ediff-files \"left\" \"right\")"`.
  3-way: `emacs --eval "(ediff3 \"left\" \"middle\" \"right\")"`.
- **`--merge` CLI flag** — choose between `vim` (default) and `emacs`

---

## Configuration

### Command-line flags
| Flag | Description |
|------|-------------|
| `-c` / `--colors` | color depth: `auto`, `256`, `8`, `none` |
| `--merge` | merge tool: `vim` (default) or `emacs` |
| `-A` / `--ascii` | force ASCII tree symbols (already using ASCII; expose as flag) |
| `-U` / `--unicode` | force Unicode tree symbols |

### Config files
- `/etc/umerge.conf` — system-wide defaults (INI format)
- `~/.umergerc` — per-user overrides
- Config drives colors, merge tool choice, and symbol mode

---

## Coloring refinements

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

## Robustness

- **Cancel background comparison on quit** — the comparison goroutine
  currently runs to completion even after the user presses `q`.  Pass a
  `context.Context` so it stops promptly.
- **Error state display** — entries that fail to compare (e.g., permission
  denied) should render in an error color rather than staying white.
- **Lazy tree loading** — `BuildPair`/`BuildTriple` currently read the
  entire directory tree eagerly at startup.  For very large trees a lazy
  approach (load children on expand) would improve startup time.
