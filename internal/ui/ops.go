package ui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"umerge/internal/entry"
	"umerge/internal/fileops"
)

// copyLetterToSide maps the Python version's column letters — verified
// against Match3.letter_to_subpart and the Controller's own prompt text —
// to our internal side bytes. In 3-way mode: a=left, b=right, c=middle.
func copyLetterToSide(letter byte) byte {
	switch letter {
	case 'a':
		return 'l'
	case 'b':
		return 'r'
	case 'c':
		return 'm'
	}
	return 0
}

func getSide(e *entry.Entry, side byte) *string {
	switch side {
	case 'l':
		return e.Left
	case 'm':
		return e.Middle
	case 'r':
		return e.Right
	}
	return nil
}

func setSide(e *entry.Entry, side byte, path string) {
	switch side {
	case 'l':
		e.Left = &path
	case 'm':
		e.Middle = &path
	case 'r':
		e.Right = &path
	}
}

func (m Model) rootFor(side byte) string {
	switch side {
	case 'l':
		return m.leftRoot
	case 'm':
		return m.middleRoot
	case 'r':
		return m.rightRoot
	}
	return ""
}

// setSubtreeError marks e and all of its descendants as failed to
// compare, mirroring the Python version's set_state_of_tree(ERROR).
func setSubtreeError(e *entry.Entry) {
	e.Compare = entry.CompareError
	for _, c := range e.Children {
		setSubtreeError(c)
	}
}

// addDepth recursively adjusts Depth on entries and their descendants by
// delta, used after rebuilding a subtree via entry.BuildTree (which always
// starts counting from 0) to match the depth of the directory it replaces.
func addDepth(entries []*entry.Entry, delta int) {
	for _, e := range entries {
		e.Depth += delta
		addDepth(e.Children, delta)
	}
}

// recompareSubtree synchronously re-derives comparison state for e (or,
// if e is a directory, every file beneath it that now has every required
// side present). It's synchronous because it only ever runs right after a
// copy of a single item/subtree — the copy itself already blocked the UI
// briefly, so this doesn't introduce a new stall on top of that.
func (m Model) recompareSubtree(e *entry.Entry) {
	if !e.IsDir {
		if allSidesPresent(e, m.ways) {
			msg := compareEntry(e, m.ways)
			e.Compare = msg.state
			e.NumDiffs = msg.numDiffs
			e.LMDiffs = msg.lmDiffs
			e.MRDiffs = msg.mrDiffs
		}
		return
	}
	for _, c := range e.Children {
		m.recompareSubtree(c)
	}
}

// rebuildChildren re-enumerates e's subtree from disk after a directory
// copy, reusing entry.BuildTree instead of duplicating the tree-merge
// logic, then adjusts the result's Depth to match e's position in the
// tree (BuildTree always starts counting from 0).
func (m *Model) rebuildChildren(e *entry.Entry) {
	var mid *string
	if e.Middle != nil {
		mid = e.Middle
	}
	children, err := entry.BuildTree(e.Left, mid, e.Right, 0, e.RelPath, m.ignore)
	if err != nil {
		setSubtreeError(e)
		return
	}
	addDepth(children, e.Depth+1)
	e.Children = children
}

// beginRefresh re-enumerates e's subtree from disk (if e is a directory —
// files have nothing to enumerate) and starts a background re-compare of
// it, reusing the same goroutine+channel pattern as the initial
// comparison. Unlike copyEntry's post-copy recompare, this isn't
// synchronous: a manual refresh isn't piggybacking on an operation that
// already blocked the UI, so a large subtree could otherwise cause a
// noticeable stall. The caller (Update()'s "r" handler) guards against
// starting a refresh while another comparison is already running —
// compareCh/comparing are shared with the initial scan, so overlapping
// runs would race on the same channel — mirroring Python's own
// operation_thread guard (Model2.request_operation).
func (m *Model) beginRefresh(e *entry.Entry) tea.Cmd {
	if e.IsDir {
		m.rebuildChildren(e)
		m.reflatten()
	}
	m.compareCh = startCompare([]*entry.Entry{e}, m.ways)
	m.comparing = true
	return listenForCompare(m.compareCh)
}

// copyEntry copies e from side "from" to side "to" ('l', 'm', or 'r'),
// mirroring FileOpsPOSIX's cp -R semantics from the Python version. It is
// a no-op if the source side is absent (matches Model2's
// "if item.left is None: return" guard). On success it updates e's
// pointer for "to", re-enumerates e's subtree if it's a directory, and
// re-derives comparison state for whatever changed.
func (m *Model) copyEntry(e *entry.Entry, from, to byte) {
	src := getSide(e, from)
	if src == nil {
		return
	}

	var destPath string
	if dest := getSide(e, to); dest != nil {
		destPath = *dest
	} else {
		rel, err := filepath.Rel(m.rootFor(from), *src)
		if err != nil {
			setSubtreeError(e)
			m.flash = "Copy failed: " + err.Error()
			return
		}
		destPath = filepath.Join(m.rootFor(to), rel)
	}

	if err := fileops.Copy(*src, destPath); err != nil {
		setSubtreeError(e)
		m.flash = "Copy failed: " + err.Error()
		return
	}
	setSide(e, to, destPath)

	if e.IsDir {
		m.rebuildChildren(e)
		// m.flat is a separately-maintained flattened cache of the tree —
		// it isn't re-derived automatically just because e.Children changed
		// underneath it. Without this, a directory copy's new contents are
		// invisible until something else (e.g. collapse/expand) happens to
		// call reflatten for its own reasons.
		m.reflatten()
	}
	m.recompareSubtree(e)
}

// deleteEntry removes every present side of e from disk. On success, e is
// spliced out of the tree entirely. On failure, e's subtree is marked as
// an error and left in place — matching Model2's __delete_item, which
// returns False (don't reset the cursor) when a delete fails.
func (m *Model) deleteEntry(e *entry.Entry) {
	var firstErr error
	for _, p := range []*string{e.Left, e.Middle, e.Right} {
		if p == nil {
			continue
		}
		if err := fileops.Delete(*p); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		setSubtreeError(e)
		m.flash = "Delete failed: " + firstErr.Error()
		return
	}
	m.entries = removeEntry(m.entries, e)
	m.reflatten()
}

// removeEntry returns entries with target spliced out, searching
// recursively into children. Entry has no parent pointer, so this walks
// the tree rather than following a back-reference.
func removeEntry(entries []*entry.Entry, target *entry.Entry) []*entry.Entry {
	out := entries[:0:0]
	for _, e := range entries {
		if e == target {
			continue
		}
		e.Children = removeEntry(e.Children, target)
		out = append(out, e)
	}
	return out
}
