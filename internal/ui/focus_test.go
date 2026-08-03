package ui

import (
	"testing"

	"github.com/mcjames/umerge/internal/entry"
)

// ── initDiffCounts ───────────────────────────────────────────────────────────

func TestInitDiffCounts_PresenceMismatchIsImmediatelyDirty(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a")} // right absent — presence mismatch
	initDiffCounts(e, 2)

	if e.PendingCount != 0 || e.DirtyCount != 1 || e.CleanCount != 0 {
		t.Errorf("got pending=%d dirty=%d clean=%d, want 0,1,0", e.PendingCount, e.DirtyCount, e.CleanCount)
	}
}

func TestInitDiffCounts_ComparableFileIsPendingUntilCompared(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Uncompared}
	initDiffCounts(e, 2)

	if e.PendingCount != 1 || e.DirtyCount != 0 || e.CleanCount != 0 {
		t.Errorf("got pending=%d dirty=%d clean=%d, want 1,0,0", e.PendingCount, e.DirtyCount, e.CleanCount)
	}
}

func TestInitDiffCounts_AlreadyKnownSameOrDifferent(t *testing.T) {
	same := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same}
	initDiffCounts(same, 2)
	if same.PendingCount != 0 || same.DirtyCount != 0 || same.CleanCount != 1 {
		t.Errorf("Same: got pending=%d dirty=%d clean=%d, want 0,0,1", same.PendingCount, same.DirtyCount, same.CleanCount)
	}

	diff := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Different}
	initDiffCounts(diff, 2)
	if diff.PendingCount != 0 || diff.DirtyCount != 1 || diff.CleanCount != 0 {
		t.Errorf("Different: got pending=%d dirty=%d clean=%d, want 0,1,0", diff.PendingCount, diff.DirtyCount, diff.CleanCount)
	}
}

func TestInitDiffCounts_DirectoryAggregatesChildren(t *testing.T) {
	clean := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same}
	pending := &entry.Entry{Left: strptr("/c"), Right: strptr("/d"), Compare: entry.Uncompared}
	dirty := &entry.Entry{Left: strptr("/e")} // presence mismatch
	dir := &entry.Entry{IsDir: true, Children: []*entry.Entry{clean, pending, dirty}}

	initDiffCounts(dir, 2)

	if dir.CleanCount != 1 || dir.PendingCount != 1 || dir.DirtyCount != 1 {
		t.Errorf("got clean=%d pending=%d dirty=%d, want 1,1,1", dir.CleanCount, dir.PendingCount, dir.DirtyCount)
	}
}

// ── recordCompareResult ──────────────────────────────────────────────────────

func TestRecordCompareResult_SamePropagatesToCleanAndMayChangeVisibility(t *testing.T) {
	root := &entry.Entry{IsDir: true, PendingCount: 1, DirtyCount: 0, CleanCount: 0}
	leaf := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same, PendingCount: 1, Parent: root}

	mayChange := recordCompareResult(leaf)

	if leaf.PendingCount != 0 || leaf.CleanCount != 1 {
		t.Errorf("leaf: pending=%d clean=%d, want 0,1", leaf.PendingCount, leaf.CleanCount)
	}
	if root.PendingCount != 0 || root.CleanCount != 1 || root.DirtyCount != 0 {
		t.Errorf("root: pending=%d clean=%d dirty=%d, want 0,1,0", root.PendingCount, root.CleanCount, root.DirtyCount)
	}
	if !mayChange {
		t.Error("mayChangeFocusVisibility = false, want true — leaf resolved Same, so it just became hideable")
	}
}

func TestRecordCompareResult_DifferentNeverChangesVisibility(t *testing.T) {
	root := &entry.Entry{IsDir: true, PendingCount: 1}
	leaf := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Different, PendingCount: 1, Parent: root}

	mayChange := recordCompareResult(leaf)

	if root.DirtyCount != 1 || root.PendingCount != 0 {
		t.Errorf("root: dirty=%d pending=%d, want 1,0", root.DirtyCount, root.PendingCount)
	}
	if mayChange {
		t.Error("mayChangeFocusVisibility = true, want false — a Different result never makes anything newly hideable")
	}
}

// TestRecordCompareResult_SameMayChangeVisibilityEvenWithSiblingStillPending
// is the regression case for the original bug report: a leaf resolving
// Same must be treated as possibly visibility-changing regardless of
// whether some ancestor directory also reaches confirmed-clean at the
// same time — the leaf itself needs to vanish under focus mode
// (focusSkip) independent of any ancestor's aggregate state, since
// individual clean files hide on their own, not just whole clean
// directories.
func TestRecordCompareResult_SameMayChangeVisibilityEvenWithSiblingStillPending(t *testing.T) {
	root := &entry.Entry{IsDir: true, PendingCount: 2}
	leaf := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same, PendingCount: 1, Parent: root}

	mayChange := recordCompareResult(leaf)

	if root.PendingCount != 1 {
		t.Errorf("root.PendingCount = %d, want 1 (one sibling still pending)", root.PendingCount)
	}
	if !mayChange {
		t.Error("mayChangeFocusVisibility = false, want true — the leaf itself resolved Same and must be able to vanish, regardless of siblings")
	}
}

func TestRecordCompareResult_MultiLevelAncestorsAllUpdated(t *testing.T) {
	grandparent := &entry.Entry{IsDir: true, PendingCount: 1}
	parent := &entry.Entry{IsDir: true, PendingCount: 1, Parent: grandparent}
	leaf := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same, PendingCount: 1, Parent: parent}

	recordCompareResult(leaf)

	if parent.CleanCount != 1 || parent.PendingCount != 0 {
		t.Errorf("parent: clean=%d pending=%d, want 1,0", parent.CleanCount, parent.PendingCount)
	}
	if grandparent.CleanCount != 1 || grandparent.PendingCount != 0 {
		t.Errorf("grandparent: clean=%d pending=%d, want 1,0", grandparent.CleanCount, grandparent.PendingCount)
	}
}

// ── updateDiffCounts / removeDiffCounts ──────────────────────────────────────

func TestUpdateDiffCounts_PropagatesDeltaToAncestors(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "content\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath} // right absent — dirty=1 initially
	initDiffCounts(e, 2)
	root := &entry.Entry{IsDir: true, PendingCount: e.PendingCount, DirtyCount: e.DirtyCount, CleanCount: e.CleanCount}
	e.Parent = root

	// Simulate a copy that fills in the right side, making it comparable
	// and (since content matches whatever the copy produced) Same.
	rightPath := writeFile(t, rightRoot, "f.txt", "content\n")
	e.Right = &rightPath
	e.Compare = entry.Same

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.updateDiffCounts(e)

	if e.DirtyCount != 0 || e.CleanCount != 1 {
		t.Errorf("e: dirty=%d clean=%d, want 0,1", e.DirtyCount, e.CleanCount)
	}
	if root.DirtyCount != 0 || root.CleanCount != 1 {
		t.Errorf("root: dirty=%d clean=%d, want 0,1 (delta should have propagated up)", root.DirtyCount, root.CleanCount)
	}
}

func TestUpdateDiffCounts_NoOpWhenNothingChanged(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b"), Compare: entry.Same}
	initDiffCounts(e, 2)
	root := &entry.Entry{IsDir: true, CleanCount: 1}
	e.Parent = root

	m := newTestModel(2, "", "", "", nil)
	m.updateDiffCounts(e)

	if root.CleanCount != 1 || root.DirtyCount != 0 || root.PendingCount != 0 {
		t.Errorf("root should be unchanged: clean=%d dirty=%d pending=%d", root.CleanCount, root.DirtyCount, root.PendingCount)
	}
}

func TestRemoveDiffCounts_SubtractsFromAncestors(t *testing.T) {
	root := &entry.Entry{IsDir: true, DirtyCount: 1}
	e := &entry.Entry{DirtyCount: 1, Parent: root}

	removeDiffCounts(e)

	if root.DirtyCount != 0 {
		t.Errorf("root.DirtyCount = %d, want 0", root.DirtyCount)
	}
}

// ── focusStopRecursion ───────────────────────────────────────────────────────

func TestFocusStopRecursion_TrueOnlyWhenFocusModeOnAndConfirmedClean(t *testing.T) {
	clean := &entry.Entry{IsDir: true, PendingCount: 0, DirtyCount: 0}
	dirty := &entry.Entry{IsDir: true, PendingCount: 0, DirtyCount: 1}
	pending := &entry.Entry{IsDir: true, PendingCount: 1, DirtyCount: 0}

	stopOn := focusStopRecursion(true)
	if !stopOn(clean) {
		t.Error("focus on + confirmed clean should stop recursion")
	}
	if stopOn(dirty) {
		t.Error("focus on + dirty should not stop recursion")
	}
	if stopOn(pending) {
		t.Error("focus on + pending should not stop recursion")
	}

	stopOff := focusStopRecursion(false)
	if stopOff(clean) {
		t.Error("focus off should never stop recursion, even if confirmed clean")
	}
}

// ── focusSkip / lineSkip ─────────────────────────────────────────────────────

func TestFocusSkip_TrueOnlyForCleanFilesUnderFocusMode(t *testing.T) {
	cleanFile := &entry.Entry{PendingCount: 0, DirtyCount: 0}
	dirtyFile := &entry.Entry{PendingCount: 0, DirtyCount: 1}
	cleanDir := &entry.Entry{IsDir: true, PendingCount: 0, DirtyCount: 0}

	skipOn := focusSkip(true)
	if !skipOn(cleanFile) {
		t.Error("a clean file should be skipped (vanish) under focus mode")
	}
	if skipOn(dirtyFile) {
		t.Error("a dirty file should never be skipped")
	}
	if skipOn(cleanDir) {
		t.Error("a clean DIRECTORY should not be skipped — it dims via focusStopRecursion instead, it doesn't vanish")
	}

	skipOff := focusSkip(false)
	if skipOff(cleanFile) {
		t.Error("focus off should never skip anything, even a clean file")
	}
}

func TestLineSkip_CombinesHiddenAndFocusClean(t *testing.T) {
	hiddenDirty := &entry.Entry{Hidden: true, DirtyCount: 1}
	cleanFile := &entry.Entry{DirtyCount: 0, PendingCount: 0}
	dirtyFile := &entry.Entry{DirtyCount: 1}

	skip := lineSkip(false /* renderHidden */, true /* focusMode */)
	if !skip(hiddenDirty) {
		t.Error("a hidden entry should be skipped regardless of its dirty state")
	}
	if !skip(cleanFile) {
		t.Error("a clean file should be skipped under focus mode")
	}
	if skip(dirtyFile) {
		t.Error("a dirty, non-hidden file should not be skipped")
	}
}

// TestFlatten_FocusMode_CleanFileVanishesEvenAsDirtySiblingStays is the
// regression test for the bug reported live: comparing sample directories,
// files that are equal on both/all sides didn't disappear when toggling
// focus mode. Root cause was that only whole clean DIRECTORIES were ever
// hidden (via focusStopRecursion); an individual clean FILE sitting next
// to a dirty sibling — or with no wrapping directory at all — never
// vanished on its own. Exercises the exact reported shape: loose
// top-level files, some identical, one different, no directory at all.
func TestFlatten_FocusMode_CleanFileVanishesEvenAsDirtySiblingStays(t *testing.T) {
	clean := &entry.Entry{Name: "a.txt", PendingCount: 0, DirtyCount: 0, CleanCount: 1}
	dirty := &entry.Entry{Name: "b.txt", PendingCount: 0, DirtyCount: 1}
	entries := []*entry.Entry{clean, dirty}

	flat := entry.Flatten(entries, lineSkip(false, true), focusStopRecursion(true))

	if len(flat) != 1 || flat[0].Name != "b.txt" {
		t.Fatalf("got %+v, want just [b.txt] — the clean loose file should vanish under focus mode", flat)
	}
}

// ── key wiring ────────────────────────────────────────────────────────────────

func TestUpdate_KeyFTogglesFocusModeAndReflattens(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "same.txt", "x\n")
	rightPath := writeFile(t, rightRoot, "same.txt", "x\n")
	child := &entry.Entry{Name: "same.txt", Left: &leftPath, Right: &rightPath, Compare: entry.Same}
	dir := &entry.Entry{Name: "dir", IsDir: true, Children: []*entry.Entry{child}}
	child.Parent = dir
	initDiffCounts(dir, 2) // its one child is already known Same -> confirmed clean

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{dir})
	if len(m.flat) != 2 {
		t.Fatalf("setup: got %d flat entries, want 2 (dir + child)", len(m.flat))
	}

	updated, _ := m.Update(keyMsg('f'))
	m = updated.(Model)

	if !m.focusMode {
		t.Fatal("focusMode should be true after pressing f")
	}
	if len(m.flat) != 1 || m.flat[0].Name != "dir" {
		t.Errorf("got %+v, want just [dir] — confirmed-clean dir should stop recursing into its child", m.flat)
	}

	updated, _ = m.Update(keyMsg('f'))
	m = updated.(Model)
	if m.focusMode {
		t.Error("focusMode should be false after pressing f again")
	}
	if len(m.flat) != 2 {
		t.Errorf("got %d flat entries, want 2 after turning focus back off", len(m.flat))
	}
}

func TestUpdate_KeyTTogglesShowCounts(t *testing.T) {
	m := newTestModel(2, "", "", "", nil)
	m.showCounts = false

	updated, _ := m.Update(keyMsg('t'))
	m = updated.(Model)
	if !m.showCounts {
		t.Error("showCounts should be true after pressing t")
	}

	updated, _ = m.Update(keyMsg('t'))
	m = updated.(Model)
	if m.showCounts {
		t.Error("showCounts should be false after pressing t again")
	}
}

func TestUpdate_CompareDoneMsg_ResetsShowCounts(t *testing.T) {
	m := newTestModel(2, "", "", "", nil)
	m.showCounts = true

	updated, _ := m.Update(compareDoneMsg{})
	m = updated.(Model)

	if m.showCounts {
		t.Error("showCounts should be unconditionally reset to false when comparison finishes")
	}
}

func TestUpdate_CompareResultMsg_ReflattensOnlyWhenFocusModeAndJustConfirmedClean(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "x\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "x\n")
	leaf := &entry.Entry{Name: "f.txt", Left: &leftPath, Right: &rightPath, Depth: 1}
	dir := &entry.Entry{Name: "dir", IsDir: true, Children: []*entry.Entry{leaf}}
	leaf.Parent = dir
	initDiffCounts(dir, 2) // leaf is pending

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{dir})
	m.focusMode = true
	if len(m.flat) != 2 {
		t.Fatalf("setup: got %d flat entries, want 2 (dir + pending leaf, still visible)", len(m.flat))
	}

	updated, _ := m.Update(compareResultMsg{e: leaf, state: entry.Same})
	m = updated.(Model)

	if dir.PendingCount != 0 || dir.DirtyCount != 0 {
		t.Errorf("dir: pending=%d dirty=%d, want 0,0", dir.PendingCount, dir.DirtyCount)
	}
	if len(m.flat) != 1 || m.flat[0].Name != "dir" {
		t.Errorf("got %+v, want just [dir] — dir just became confirmed clean, should auto-collapse", m.flat)
	}
}

func TestUpdate_CompareResultMsg_NoReflattenWhenFocusModeOff(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "x\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "x\n")
	leaf := &entry.Entry{Name: "f.txt", Left: &leftPath, Right: &rightPath, Depth: 1}
	dir := &entry.Entry{Name: "dir", IsDir: true, Children: []*entry.Entry{leaf}}
	leaf.Parent = dir
	initDiffCounts(dir, 2)

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{dir})
	// focusMode left false (default)

	updated, _ := m.Update(compareResultMsg{e: leaf, state: entry.Same})
	m = updated.(Model)

	if len(m.flat) != 2 {
		t.Errorf("got %d flat entries, want 2 — focus mode is off, nothing should auto-collapse", len(m.flat))
	}
}

// ── status line ──────────────────────────────────────────────────────────────

func TestDiffCountsSummary_DropsPendingSegmentWhenZero(t *testing.T) {
	clean := &entry.Entry{CleanCount: 5}
	dirty := &entry.Entry{DirtyCount: 2}
	m := newTestModel(2, "", "", "", []*entry.Entry{clean, dirty})

	got := m.diffCountsSummary()
	want := "5 clean · 2 differ"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDiffCountsSummary_IncludesPendingWhenNonzero(t *testing.T) {
	e := &entry.Entry{CleanCount: 3, PendingCount: 4, DirtyCount: 1}
	m := newTestModel(2, "", "", "", []*entry.Entry{e})

	got := m.diffCountsSummary()
	want := "3 clean · 4 pending · 1 differ"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
