package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcjames/umerge/internal/entry"
	"github.com/mcjames/umerge/internal/fileops"
)

// compareResultMsg carries the result of one file comparison back to Update.
type compareResultMsg struct {
	e        *entry.Entry
	state    entry.CompareState
	numDiffs int // 2-way
	lmDiffs  int // 3-way left↔middle
	mrDiffs  int // 3-way middle↔right
}

// compareDoneMsg is sent when all comparisons have finished.
type compareDoneMsg struct{}

// listenForCompare returns a Cmd that blocks until one message arrives on ch.
func listenForCompare(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// startCompare launches the comparison goroutine and returns the channel to
// listen on.  The goroutine closes the channel after sending compareDoneMsg.
func startCompare(entries []*entry.Entry, ways int) <-chan tea.Msg {
	ch := make(chan tea.Msg)
	go func() {
		walkAndCompare(entries, ways, ch)
		ch <- compareDoneMsg{}
		close(ch)
	}()
	return ch
}

// walkAndCompare walks the tree depth-first and sends one compareResultMsg per
// file that should be compared (both/all sides present, no directories).
func walkAndCompare(entries []*entry.Entry, ways int, ch chan<- tea.Msg) {
	for _, e := range entries {
		if !e.IsDir && allSidesPresent(e, ways) {
			ch <- compareEntry(e, ways)
		}
		walkAndCompare(e.Children, ways, ch)
	}
}

func allSidesPresent(e *entry.Entry, ways int) bool {
	if ways == 2 {
		return e.Left != nil && e.Right != nil
	}
	return e.Left != nil && e.Middle != nil && e.Right != nil
}

// compareSymlinks classifies a comparison by symlink target instead of
// file content when at least one of paths is a symbolic link — reading
// through a symlink to a directory fails outright ("is a directory"),
// and even for a symlink to a regular file, reading through it conflates
// "did the link itself change" with "did the target's content change".
// handled is false when none of paths is a symlink, meaning the caller
// should fall through to its normal file-content comparison.
func compareSymlinks(paths []string) (state entry.CompareState, handled bool) {
	targets := make([]string, len(paths))
	isLink := make([]bool, len(paths))
	anyLink := false
	for i, p := range paths {
		target, link, err := fileops.SymlinkTarget(p)
		if err != nil {
			return entry.CompareError, true
		}
		targets[i], isLink[i] = target, link
		anyLink = anyLink || link
	}
	if !anyLink {
		return 0, false
	}
	for i := 1; i < len(paths); i++ {
		if isLink[i] != isLink[0] || targets[i] != targets[0] {
			return entry.SymlinkDifferent, true
		}
	}
	return entry.Same, true
}

func compareEntry(e *entry.Entry, ways int) compareResultMsg {
	msg := compareResultMsg{e: e}

	if ways == 2 {
		if state, handled := compareSymlinks([]string{*e.Left, *e.Right}); handled {
			msg.state = state
			return msg
		}
		n, binary, err := fileops.CompareTwoFiles(*e.Left, *e.Right)
		if err != nil {
			msg.state = entry.CompareError
			return msg
		}
		if binary {
			msg.state = entry.BinaryDifferent
			return msg
		}
		msg.numDiffs = n
		if n == 0 {
			msg.state = entry.Same
		} else {
			msg.state = entry.Different
		}
		return msg
	}

	// 3-way
	if state, handled := compareSymlinks([]string{*e.Left, *e.Middle, *e.Right}); handled {
		msg.state = state
		return msg
	}
	lm, mr, binary, err := fileops.CompareThreeFiles(*e.Left, *e.Middle, *e.Right)
	if err != nil {
		msg.state = entry.CompareError
		return msg
	}
	if binary {
		msg.state = entry.BinaryDifferent
		return msg
	}
	msg.lmDiffs = lm
	msg.mrDiffs = mr
	if lm == 0 && mr == 0 {
		msg.state = entry.Same
	} else {
		msg.state = entry.Different
	}
	return msg
}

// ── focus mode (TODO.md Priority 3b): Pending/Dirty/Clean bookkeeping ──────────
//
// Three different update paths, each suited to how often it runs:
//   - initDiffCounts: a full bottom-up recompute of a subtree. Used once
//     up front (New) and after anything that changes what a subtree
//     looks like out from under the running totals — a copy, delete,
//     merge, or manual refresh (see updateDiffCounts/removeDiffCounts).
//   - recordCompareResult: the O(depth) incremental path for the one
//     leaf a single compareResultMsg is about, walking up through its
//     ancestors. This is the one called on every message during a scan
//     (potentially tens of thousands on a large tree), which is exactly
//     why it must not be an O(n) full-tree recompute.
//   - updateDiffCounts/removeDiffCounts: wrap initDiffCounts for the
//     mutation call sites, propagating the delta between an entry's old
//     and new counts up through its ancestors (or, for a deletion,
//     simply subtracting its current contribution).

// initDiffCounts computes e's Pending/Dirty/Clean leaf counts, bottom-up.
// A file contributes to exactly one of the three: Pending if a compare
// result for it hasn't arrived yet, Dirty immediately (no result is ever
// coming) if it's a presence mismatch — absent on at least one required
// side, known the instant the tree is built, per CLAUDE.md's eager/
// synchronous BuildTree — or Dirty/Clean once its Compare state is
// already known (Same is Clean; Different/BinaryDifferent/CompareError
// are all Dirty — anything that isn't a confirmed-fine Same should keep
// an entry visible under focus mode). A directory's counts are the sum
// of its children's.
func initDiffCounts(e *entry.Entry, ways int) {
	if e.IsDir {
		e.PendingCount, e.DirtyCount, e.CleanCount = 0, 0, 0
		for _, c := range e.Children {
			initDiffCounts(c, ways)
			e.PendingCount += c.PendingCount
			e.DirtyCount += c.DirtyCount
			e.CleanCount += c.CleanCount
		}
		return
	}
	if !allSidesPresent(e, ways) {
		e.PendingCount, e.DirtyCount, e.CleanCount = 0, 1, 0
		return
	}
	switch e.Compare {
	case entry.Same:
		e.PendingCount, e.DirtyCount, e.CleanCount = 0, 0, 1
	case entry.Uncompared:
		e.PendingCount, e.DirtyCount, e.CleanCount = 1, 0, 0
	default: // Different, BinaryDifferent, CompareError
		e.PendingCount, e.DirtyCount, e.CleanCount = 0, 1, 0
	}
}

// recordCompareResult updates the Pending/Dirty/Clean counts after e's
// Compare state was just set by a compareResultMsg, walking up through
// e's ancestors. Depth-bounded, not a full-tree walk — safe to call
// unconditionally on every message a scan sends. Returns whether this
// result can change what's visible under focus mode — true exactly when
// e resolved Same, since that's the only kind of transition that does:
// e itself becomes hideable (focusSkip), independent of whether any
// ancestor directory's own aggregate happens to reach confirmed-clean at
// the same time — a Different/error/binary result never makes anything
// newly hideable, so it's always false there. This is the caller's cue
// for whether a reflatten is actually needed: reflattening unconditionally
// on every message would be the same O(n²) trap as recomputing counts
// from scratch each time (see TODO.md Priority 3b's "performance risk"
// note) — reflatten() itself recomputes the whole flat list fresh, so it
// doesn't matter here whether it was e itself or some ancestor whose
// visibility actually changed as a result.
func recordCompareResult(e *entry.Entry) (mayChangeFocusVisibility bool) {
	dirty := e.Compare != entry.Same
	e.PendingCount = 0
	if dirty {
		e.DirtyCount = 1
	} else {
		e.CleanCount = 1
	}
	for p := e.Parent; p != nil; p = p.Parent {
		p.PendingCount--
		if dirty {
			p.DirtyCount++
		} else {
			p.CleanCount++
		}
	}
	return !dirty
}

// updateDiffCounts recomputes e's own Pending/Dirty/Clean counts (and,
// if e is a directory, its whole subtree's, via initDiffCounts) and
// propagates the resulting delta up through e's ancestors. Call this
// after anything that changes what initDiffCounts already baked into
// e's ancestors' running totals without going through the
// compareResultMsg path: a copy or merge that changes which sides are
// present, a synchronous recompare that changes e.Compare, or a
// directory subtree rebuilt from disk.
func (m Model) updateDiffCounts(e *entry.Entry) {
	oldP, oldD, oldC := e.PendingCount, e.DirtyCount, e.CleanCount
	initDiffCounts(e, m.ways)
	dp, dd, dc := e.PendingCount-oldP, e.DirtyCount-oldD, e.CleanCount-oldC
	if dp == 0 && dd == 0 && dc == 0 {
		return
	}
	for p := e.Parent; p != nil; p = p.Parent {
		p.PendingCount += dp
		p.DirtyCount += dd
		p.CleanCount += dc
	}
}

// removeDiffCounts subtracts e's current Pending/Dirty/Clean
// contribution from every ancestor — call this just before e is spliced
// out of the tree entirely (a delete, or a merge that honors a
// deletion), since e's own fields stop mattering once it's gone but its
// ancestors' running totals must no longer include it.
func removeDiffCounts(e *entry.Entry) {
	for p := e.Parent; p != nil; p = p.Parent {
		p.PendingCount -= e.PendingCount
		p.DirtyCount -= e.DirtyCount
		p.CleanCount -= e.CleanCount
	}
}
