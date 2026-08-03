package ui

import (
	"os"
	"path/filepath"

	"github.com/mcjames/umerge/internal/entry"
	"github.com/mcjames/umerge/internal/fileops"
)

// mergeGuardOK reports whether a merge command (m/M/n) may proceed,
// flashing an explanatory message and returning false otherwise. Mirrors
// beginCopy's read-only and still-comparing guards — merge mutates the
// tree the same way copy does, so the same race with the background
// comparison goroutine applies.
func (m *Model) mergeGuardOK() bool {
	if m.readOnly {
		m.flash = "Read-only mode (--read-only): merge is disabled"
		return false
	}
	if m.comparing {
		m.flash = "Still comparing — please wait"
		return false
	}
	return true
}

// mergeCursorItem implements the `m` key: merge just the cursor item,
// regardless of any active selection — unlike a/b/d, `m`/`M` are two
// separate keys with two separate scopes (see TODO.md Priority 4), not
// one key that silently switches scope when a selection exists.
func (m *Model) mergeCursorItem() {
	if !m.mergeGuardOK() || len(m.flat) == 0 {
		return
	}
	m.mergeItem(m.flat[m.cursor])
	m.reflatten()
}

// mergeSelection implements the `M` key: merge every selected root.
func (m *Model) mergeSelection() {
	if !m.mergeGuardOK() {
		return
	}
	roots := selectedRoots(m.entries)
	if len(roots) == 0 {
		m.flash = "Nothing selected"
		return
	}
	for _, e := range roots {
		m.mergeItem(e)
	}
	m.reflatten()
}

// mergeAll implements the `n` key: merge the entire tree to center in one
// keystroke (Model3.merge_all applied to the root).
func (m *Model) mergeAll() {
	if !m.mergeGuardOK() {
		return
	}
	for _, e := range m.entries {
		m.mergeItem(e)
	}
	m.reflatten()
}

// mergeItem decides how to auto-merge e into middle, or marks it for
// manual resolution, per TODO.md Priority 4's classifier. It mirrors the
// Python reference's Model3.__merge_individual_item but deliberately
// diverges from it in two ways, both documented in TODO.md:
//   - a modify/delete mismatch (one side edited, the other deleted) is a
//     conflict, not a silent auto-resolve;
//   - a directory added independently on both sides (no common ancestor)
//     is recursed into per-child rather than force-conflicted as one
//     opaque unit — files deep inside that happen to be identical still
//     auto-resolve, matching how the file case already behaves.
func (m *Model) mergeItem(e *entry.Entry) {
	if e.IsDir {
		m.mergeDirItem(e)
	} else {
		m.mergeFileItem(e)
	}
}

func (m *Model) mergeFileItem(e *entry.Entry) {
	switch {
	case e.Left == nil && e.Middle != nil && e.Right == nil:
		m.mergeDeleteMiddle(e)

	case e.Left != nil && e.Right == nil:
		m.mergeOneSideAbsent(e, 'l', entry.ResolutionTookLeft)

	case e.Left == nil && e.Right != nil:
		m.mergeOneSideAbsent(e, 'r', entry.ResolutionTookRight)

	case e.Left != nil && e.Middle == nil && e.Right != nil:
		m.mergeNoCommonAncestor(e)

	case e.Left != nil && e.Middle != nil && e.Right != nil:
		m.mergeAllThreePresent(e)
	}
}

func (m *Model) mergeDirItem(e *entry.Entry) {
	switch {
	case e.Left == nil && e.Middle != nil && e.Right == nil:
		m.mergeDeleteMiddle(e)

	case e.Left != nil && e.Right == nil:
		m.mergeOneSideAbsentDir(e, 'l', entry.ResolutionTookLeft)

	case e.Left == nil && e.Right != nil:
		m.mergeOneSideAbsentDir(e, 'r', entry.ResolutionTookRight)

	case e.Left != nil && e.Middle == nil && e.Right != nil:
		// No common ancestor for this whole subtree, and — unlike a
		// file — no cheap way to prove two directory trees are
		// byte-identical. Create the middle directory and recurse,
		// letting each descendant resolve or conflict on its own
		// rather than forcing the whole subtree into one opaque
		// conflict (a deliberate divergence from Python here; see
		// TODO.md Priority 4).
		if !m.mergeCreateMiddleDir(e) {
			return
		}
		for _, c := range e.Children {
			m.mergeItem(c)
		}

	case e.Left != nil && e.Middle != nil && e.Right != nil:
		// No single diff3 call applies to a whole directory — recurse
		// and let each child resolve independently. e's own
		// Resolution is deliberately left unresolved: a mix of
		// resolved/conflicted children has no single honest
		// one-character summary (matches Python: it never calls
		// set_resolution_status_of_tree on this node in this branch).
		for _, c := range e.Children {
			m.mergeItem(c)
		}
	}
}

// mergeOneSideAbsent handles a file present on side `present` ('l' or
// 'r') and absent on the other. If middle is also absent, this is a
// genuine one-sided add — copy it in and mark took. If middle is
// present, the other side deleted a file middle still has; this only
// auto-resolves (deleting middle to honor that deletion) when the
// surviving side is byte-identical to middle — nothing of its own was
// lost. If it differs, that's a real modify/delete conflict (TODO.md
// Priority 4: "delete/modify is a conflict, not a silent auto-resolve")
// and is left for manual resolution instead of Python's unconditional
// auto-copy here.
func (m *Model) mergeOneSideAbsent(e *entry.Entry, present byte, took entry.ResolutionStatus) {
	src := getSide(e, present)
	if e.Middle == nil {
		m.mergeCopyToMiddle(e, present, took)
		return
	}
	equal, err := contentEqual(*src, *e.Middle)
	if err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return
	}
	if equal {
		m.mergeDeleteMiddle(e)
		return
	}
	e.Resolution = entry.ResolutionConflict
}

// mergeOneSideAbsentDir is mergeOneSideAbsent's directory counterpart.
// There's no cheap way to prove two whole directory subtrees are
// identical, so the modify/delete case is unconditionally a conflict
// here rather than attempting a recursive equality check.
func (m *Model) mergeOneSideAbsentDir(e *entry.Entry, present byte, took entry.ResolutionStatus) {
	if e.Middle == nil {
		m.mergeCopyToMiddle(e, present, took)
		return
	}
	e.Resolution = entry.ResolutionConflict
}

// mergeNoCommonAncestor handles a file present on both left and right
// with no middle counterpart at all. Auto-resolves when left and right
// are byte-identical (added independently, no real conflict of intent);
// otherwise marks a conflict — matches the Python reference's actual
// behavior here (checked directly against Model3.py, not just TODO.md's
// summary, which had this wrong).
func (m *Model) mergeNoCommonAncestor(e *entry.Entry) {
	equal, err := contentEqual(*e.Left, *e.Right)
	if err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return
	}
	if equal {
		m.mergeCopyToMiddle(e, 'r', entry.ResolutionTookRight)
		return
	}
	e.Resolution = entry.ResolutionConflict
}

// mergeAllThreePresent classifies a file present on every side: no-op if
// already identical everywhere, an immediate conflict for binary content
// (diff3 can't merge it — see fileops.CompareThreeFiles), otherwise a
// real diff3 -m merge, writing the result to middle on success or
// flagging a conflict (leaving middle untouched) on overlap.
func (m *Model) mergeAllThreePresent(e *entry.Entry) {
	if e.Compare == entry.Same {
		return
	}
	if e.Compare == entry.BinaryDifferent {
		e.Resolution = entry.ResolutionConflict
		return
	}
	merged, conflict, err := fileops.MergeThreeFiles(*e.Left, *e.Middle, *e.Right)
	if err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return
	}
	if conflict {
		e.Resolution = entry.ResolutionConflict
		return
	}
	if err := os.WriteFile(*e.Middle, merged, 0o644); err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return
	}
	e.Resolution = entry.ResolutionMerged
	m.recompareSubtree(e)
}

// mergeCopyToMiddle copies e from side "from" to middle (reusing the
// existing Priority 1 copyEntry, which already handles missing parent
// directories, subtree rebuild, and recompare) and marks the result with
// took — but only on success; copyEntry already flashes its own error
// and marks the entry CompareError on failure, so this doesn't overwrite
// that with a resolution status that would claim success.
func (m *Model) mergeCopyToMiddle(e *entry.Entry, from byte, took entry.ResolutionStatus) {
	m.copyEntry(e, from, 'm')
	if e.Compare != entry.CompareError {
		e.SetResolution(took)
	}
}

// mergeDeleteMiddle deletes e's middle side (honoring a deletion the
// other side already made) and re-derives the tree around it: if e is a
// directory, its children are re-enumerated from disk (rebuildChildren)
// so every descendant's stale Middle pointer is cleared too, not just
// e's own; if nothing remains on any side, e is spliced out of the tree
// entirely — the same "both sides deleted it" shape as a plain delete.
func (m *Model) mergeDeleteMiddle(e *entry.Entry) {
	if e.Middle == nil {
		return
	}
	if err := fileops.Delete(*e.Middle); err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return
	}
	e.Middle = nil
	if e.IsDir {
		m.rebuildChildren(e)
	}
	if e.Left == nil && e.Right == nil {
		m.entries = removeEntry(m.entries, e)
	}
	m.reflatten()
}

// mergeCreateMiddleDir creates the middle-side directory for e (whose
// path is derived the same way copyEntry derives a destination for an
// absent side) and wires it onto e, returning false — after flashing an
// error and marking the subtree failed — if either step fails.
func (m *Model) mergeCreateMiddleDir(e *entry.Entry) bool {
	rel, err := filepath.Rel(m.rootFor('l'), *e.Left)
	if err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return false
	}
	dest := filepath.Join(m.rootFor('m'), rel)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		setSubtreeError(e)
		m.flash = "Merge failed: " + err.Error()
		return false
	}
	e.Middle = &dest
	return true
}

// contentEqual reports whether a and b are byte-identical, reusing
// fileops.CompareTwoFiles (which already short-circuits via a size+chunk
// comparison before ever invoking diff) rather than adding a separate
// exported equality primitive to fileops for this one call site.
func contentEqual(a, b string) (bool, error) {
	n, binary, err := fileops.CompareTwoFiles(a, b)
	if err != nil {
		return false, err
	}
	return n == 0 && !binary, nil
}
