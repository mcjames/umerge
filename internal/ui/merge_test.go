package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcjames/umerge/internal/entry"
)

// ── mergeAllThreePresent (all sides present) ────────────────────────────────

func TestMergeItem_AllThreePresent_SameIsNoOp(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "same\n")
	middlePath := writeFile(t, middleRoot, "f.txt", "same\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "same\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Middle: &middlePath, Right: &rightPath, Compare: entry.Same}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionUnresolved {
		t.Errorf("Resolution = %q, want unresolved — nothing to merge when already Same", e.Resolution.Char())
	}
	got, _ := os.ReadFile(middlePath)
	if string(got) != "same\n" {
		t.Errorf("middle content = %q, want unchanged %q", got, "same\n")
	}
}

func TestMergeItem_AllThreePresent_NonOverlappingMergesCleanly(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "LEFT1\nline2\nline3\nline4\n")
	middlePath := writeFile(t, middleRoot, "f.txt", "line1\nline2\nline3\nline4\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "line1\nline2\nline3\nRIGHT4\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Middle: &middlePath, Right: &rightPath, Compare: entry.Different}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionMerged {
		t.Errorf("Resolution = %q, want 'm'", e.Resolution.Char())
	}
	got, _ := os.ReadFile(middlePath)
	want := "LEFT1\nline2\nline3\nRIGHT4\n"
	if string(got) != want {
		t.Errorf("middle content = %q, want %q", got, want)
	}
	// The merge combines both sides' independent changes, so the result
	// legitimately still differs from each input individually — recompare
	// should reflect that, not claim everything's now identical.
	if e.Compare != entry.Different {
		t.Errorf("Compare = %v, want Different (merged content differs from both left and right individually)", e.Compare)
	}
	if e.LMDiffs == 0 || e.MRDiffs == 0 {
		t.Errorf("LMDiffs=%d MRDiffs=%d, want both > 0 — recompareSubtree should have run after the merge", e.LMDiffs, e.MRDiffs)
	}
}

func TestMergeItem_AllThreePresent_OverlappingChangeIsConflictAndLeavesMiddleUntouched(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "line1\nLEFT-CHANGE\nline3\n")
	middlePath := writeFile(t, middleRoot, "f.txt", "line1\nline2\nline3\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "line1\nRIGHT-CHANGE\nline3\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Middle: &middlePath, Right: &rightPath, Compare: entry.Different}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c'", e.Resolution.Char())
	}
	got, _ := os.ReadFile(middlePath)
	if string(got) != "line1\nline2\nline3\n" {
		t.Errorf("middle should be left byte-for-byte untouched on conflict, got %q", got)
	}
}

func TestMergeItem_AllThreePresent_BinaryDifferentIsConflictWithoutInvokingDiff3(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeBinaryFile(t, leftRoot, "f.bin", []byte{0x00, 0x01})
	middlePath := writeBinaryFile(t, middleRoot, "f.bin", []byte{0x00, 0x02})
	rightPath := writeBinaryFile(t, rightRoot, "f.bin", []byte{0x00, 0x03})
	e := &entry.Entry{Name: "f.bin", Left: &leftPath, Middle: &middlePath, Right: &rightPath, Compare: entry.BinaryDifferent}

	t.Setenv("PATH", t.TempDir()) // proves diff3 is never invoked for this case

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c'", e.Resolution.Char())
	}
}

func TestMergeItem_AllThreePresent_SymlinkDifferentIsConflictWithoutInvokingDiff3(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := mkSymlink(t, leftRoot, "link", "target-a")
	middlePath := mkSymlink(t, middleRoot, "link", "target-b")
	rightPath := mkSymlink(t, rightRoot, "link", "target-c")
	e := &entry.Entry{Name: "link", Left: &leftPath, Middle: &middlePath, Right: &rightPath, Compare: entry.SymlinkDifferent}

	t.Setenv("PATH", t.TempDir()) // proves diff3 is never invoked for this case

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c'", e.Resolution.Char())
	}
}

// ── mergeOneSideAbsent (file present on one side, absent on the other) ──────

func TestMergeItem_OneSidedAdd_NoMiddle_CopiesAndMarks(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "new file\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	wantDest := filepath.Join(middleRoot, "f.txt")
	if e.Middle == nil || *e.Middle != wantDest {
		t.Fatalf("e.Middle = %v, want %q", e.Middle, wantDest)
	}
	if e.Resolution != entry.ResolutionTookLeft {
		t.Errorf("Resolution = %q, want 'a'", e.Resolution.Char())
	}
}

func TestMergeItem_RightDeleted_LeftUnchanged_HonorsDeletionSilently(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "unchanged\n")
	middlePath := writeFile(t, middleRoot, "f.txt", "unchanged\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Middle: &middlePath} // Right deleted it

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Middle != nil {
		t.Errorf("e.Middle = %v, want nil — right's deletion should be honored since left never touched it", e.Middle)
	}
	if _, err := os.Stat(middlePath); !os.IsNotExist(err) {
		t.Errorf("middle file should be deleted from disk, err=%v", err)
	}
	if e.Resolution != entry.ResolutionUnresolved {
		t.Errorf("Resolution = %q, want unresolved — uncontested deletions get no marker", e.Resolution.Char())
	}
	if e.Left == nil {
		t.Error("e.Left should be untouched — the surviving side's own file is not deleted")
	}
}

func TestMergeItem_RightDeleted_LeftModified_IsConflictNotAutoResolved(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "MODIFIED\n")
	middlePath := writeFile(t, middleRoot, "f.txt", "original\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Middle: &middlePath} // Right deleted it

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c' — this is a modify/delete conflict, not a silent auto-resolve", e.Resolution.Char())
	}
	if e.Middle == nil {
		t.Error("e.Middle should be untouched (still present) — conflicts leave state as-is")
	}
	got, _ := os.ReadFile(middlePath)
	if string(got) != "original\n" {
		t.Errorf("middle content = %q, want unchanged", got)
	}
}

func TestMergeItem_BothSidesDeleted_DeletesMiddleAndSplices(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	middlePath := writeFile(t, middleRoot, "f.txt", "content\n")
	e := &entry.Entry{Name: "f.txt", Middle: &middlePath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if _, err := os.Stat(middlePath); !os.IsNotExist(err) {
		t.Errorf("middle file should be deleted, err=%v", err)
	}
	if len(m.entries) != 0 {
		t.Errorf("entry should be spliced out once nothing remains on any side, got %+v", m.entries)
	}
}

// ── mergeNoCommonAncestor (file present on both left/right, no middle) ──────

func TestMergeItem_NoCommonAncestor_IdenticalContentAutoResolves(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "same everywhere\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "same everywhere\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionTookRight {
		t.Errorf("Resolution = %q, want 'b' (matches Python's real, tested-here behavior of arbitrarily preferring right)", e.Resolution.Char())
	}
	wantDest := filepath.Join(middleRoot, "f.txt")
	if e.Middle == nil || *e.Middle != wantDest {
		t.Fatalf("e.Middle = %v, want %q", e.Middle, wantDest)
	}
}

func TestMergeItem_NoCommonAncestor_DifferentContentConflicts(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "left version\n")
	rightPath := writeFile(t, rightRoot, "f.txt", "right version\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c' — independently added, differing content, can't tell intent", e.Resolution.Char())
	}
	if e.Middle != nil {
		t.Error("e.Middle should stay nil on a conflict — nothing gets copied")
	}
}

// TestMergeItem_NoCommonAncestor_SymlinkToDirectory is a regression case:
// mergeNoCommonAncestor uses contentEqual (fileops.CompareTwoFiles under
// the hood), which fails outright ("is a directory") if handed a symlink
// to a directory — contentEqual must check compareSymlinks first.
func TestMergeItem_NoCommonAncestor_SymlinkToDirectory(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := mkSymlink(t, leftRoot, "link", "somewhere")
	rightPath := mkSymlink(t, rightRoot, "link", "somewhere")
	e := &entry.Entry{Name: "link", Left: &leftPath, Right: &rightPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionTookRight {
		t.Errorf("Resolution = %q, want 'b' — identical symlink targets should auto-resolve, not error", e.Resolution.Char())
	}
}

// ── Directory variants ───────────────────────────────────────────────────────

func TestMergeItem_Directory_OneSidedAdd_CopiesWholeSubtreeAndMarksThroughout(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	writeFile(t, leftRoot, "sub/a.txt", "content\n")
	leftSub := filepath.Join(leftRoot, "sub")
	child := &entry.Entry{Name: "a.txt", Left: strptr(filepath.Join(leftSub, "a.txt")), Depth: 1}
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Children: []*entry.Entry{child}}
	child.Parent = e

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionTookLeft {
		t.Errorf("dir Resolution = %q, want 'a'", e.Resolution.Char())
	}
	if _, err := os.Stat(filepath.Join(middleRoot, "sub", "a.txt")); err != nil {
		t.Fatalf("copied nested file missing: %v", err)
	}
	if len(e.Children) != 1 || e.Children[0].Resolution != entry.ResolutionTookLeft {
		t.Errorf("rebuilt child should also be marked 'a' (whole-subtree copy propagates), got %+v", e.Children)
	}
}

func TestMergeItem_Directory_ModifyDeleteIsConflict(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(middleRoot, "sub"))
	leftSub, middleSub := filepath.Join(leftRoot, "sub"), filepath.Join(middleRoot, "sub")
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Middle: &middleSub} // Right deleted it

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionConflict {
		t.Errorf("Resolution = %q, want 'c' — directories get no identical-content bypass", e.Resolution.Char())
	}
	if _, err := os.Stat(middleSub); err != nil {
		t.Errorf("middle directory should be untouched on conflict: %v", err)
	}
}

func TestMergeItem_Directory_NoCommonAncestor_RecursesPerChild(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub"))
	leftSub, rightSub := filepath.Join(leftRoot, "sub"), filepath.Join(rightRoot, "sub")

	sameLeft := writeFile(t, leftRoot, "sub/same.txt", "identical\n")
	sameRight := writeFile(t, rightRoot, "sub/same.txt", "identical\n")
	diffLeft := writeFile(t, leftRoot, "sub/diff.txt", "left version\n")
	diffRight := writeFile(t, rightRoot, "sub/diff.txt", "right version\n")

	same := &entry.Entry{Name: "same.txt", Left: &sameLeft, Right: &sameRight, Depth: 1}
	diff := &entry.Entry{Name: "diff.txt", Left: &diffLeft, Right: &diffRight, Depth: 1}
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub, Children: []*entry.Entry{same, diff}}
	same.Parent, diff.Parent = e, e

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.mergeItem(e)

	if e.Resolution != entry.ResolutionUnresolved {
		t.Errorf("directory's own Resolution = %q, want unresolved — mixed children have no single summary", e.Resolution.Char())
	}
	if e.Middle == nil {
		t.Fatal("e.Middle should be created so children have somewhere to land")
	}
	if same.Resolution != entry.ResolutionTookRight || same.Middle == nil {
		t.Errorf("same.txt should auto-resolve (identical content): resolution=%q middle=%v", same.Resolution.Char(), same.Middle)
	}
	if diff.Resolution != entry.ResolutionConflict || diff.Middle != nil {
		t.Errorf("diff.txt should conflict (differing content): resolution=%q middle=%v", diff.Resolution.Char(), diff.Middle)
	}
}

// ── R key, read-only guard, comparing guard ─────────────────────────────────

func TestUpdate_ThreeWay_KeyRMarksResolvedTree(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "content\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('R'))
	m = updated.(Model)

	if e.Resolution != entry.ResolutionManual {
		t.Errorf("Resolution = %q, want 'r'", e.Resolution.Char())
	}
}

func TestMergeGuard_ReadOnlyDisablesMerge(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "content\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.readOnly = true
	m.mergeCursorItem()

	if e.Middle != nil {
		t.Error("merge should not have run in read-only mode")
	}
	if m.flash == "" {
		t.Error("flash should explain that merge is disabled")
	}
}

func TestMergeGuard_StillComparingBlocksMerge(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "f.txt", "content\n")
	e := &entry.Entry{Name: "f.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.comparing = true
	m.mergeCursorItem()

	if e.Middle != nil {
		t.Error("merge should not have run while still comparing")
	}
	if m.flash == "" {
		t.Error("flash should explain to wait")
	}
}

func TestMergeSelection_ActsOnEverySelectedRoot(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	aPath := writeFile(t, leftRoot, "a.txt", "a\n")
	bPath := writeFile(t, leftRoot, "b.txt", "b\n")
	a := &entry.Entry{Name: "a.txt", Left: &aPath, Selected: true}
	b := &entry.Entry{Name: "b.txt", Left: &bPath, Selected: true}
	c := &entry.Entry{Name: "c.txt"} // not selected — should be untouched

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{a, b, c})
	m.mergeSelection()

	if a.Resolution != entry.ResolutionTookLeft || b.Resolution != entry.ResolutionTookLeft {
		t.Errorf("both selected items should merge: a=%q b=%q", a.Resolution.Char(), b.Resolution.Char())
	}
	if c.Resolution != entry.ResolutionUnresolved {
		t.Error("unselected item should be untouched")
	}
}

func TestMergeSelection_FlashesWhenNothingSelected(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	m := newTestModel(3, leftRoot, middleRoot, rightRoot, nil)
	m.mergeSelection()

	if m.flash != "Nothing selected" {
		t.Errorf("flash = %q, want %q", m.flash, "Nothing selected")
	}
}

func TestMergeAll_MergesEveryTopLevelEntry(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	aPath := writeFile(t, leftRoot, "a.txt", "a\n")
	bPath := writeFile(t, leftRoot, "b.txt", "b\n")
	a := &entry.Entry{Name: "a.txt", Left: &aPath}
	b := &entry.Entry{Name: "b.txt", Left: &bPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{a, b})
	m.mergeAll()

	if a.Resolution != entry.ResolutionTookLeft || b.Resolution != entry.ResolutionTookLeft {
		t.Errorf("both top-level entries should merge: a=%q b=%q", a.Resolution.Char(), b.Resolution.Char())
	}
}
