package ui

import (
	"os"
	"path/filepath"
	"testing"

	"umerge/internal/entry"
)

func newTestModel(ways int, leftRoot, middleRoot, rightRoot string, entries []*entry.Entry) Model {
	m := Model{
		ways:       ways,
		leftRoot:   leftRoot,
		middleRoot: middleRoot,
		rightRoot:  rightRoot,
		entries:    entries,
	}
	m.flat = entry.Flatten(entries)
	return m
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestCopyLetterToSide(t *testing.T) {
	cases := map[byte]byte{'a': 'l', 'b': 'r', 'c': 'm'}
	for letter, want := range cases {
		if got := copyLetterToSide(letter); got != want {
			t.Errorf("copyLetterToSide(%q) = %q, want %q", letter, got, want)
		}
	}
}

func TestCopyEntry_TwoWay_NewDestination(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "hello\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.copyEntry(e, 'l', 'r')

	wantDest := filepath.Join(rightRoot, "file.txt")
	if e.Right == nil || *e.Right != wantDest {
		t.Fatalf("e.Right = %v, want %q", e.Right, wantDest)
	}
	got, err := os.ReadFile(wantDest)
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("copied content = %q, want %q", got, "hello\n")
	}
	if e.Compare != entry.Same {
		t.Errorf("Compare = %v, want Same", e.Compare)
	}
	if e.NumDiffs != 0 {
		t.Errorf("NumDiffs = %d, want 0", e.NumDiffs)
	}
}

func TestCopyEntry_TwoWay_OverwritesExistingDestFile(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "new content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "old content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.copyEntry(e, 'l', 'r')

	got, err := os.ReadFile(rightPath)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "new content\n" {
		t.Errorf("dest content = %q, want %q", got, "new content\n")
	}
	if e.Compare != entry.Same {
		t.Errorf("Compare = %v, want Same", e.Compare)
	}
}

func TestCopyEntry_Directory_RebuildsChildrenAndRecompares(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	writeFile(t, leftRoot, "sub/a.txt", "content\n")
	leftSub := filepath.Join(leftRoot, "sub")

	// Mirrors what BuildPair would have produced before the copy: the
	// directory and its child are known on the left only.
	child := &entry.Entry{
		Name:  "a.txt",
		Left:  strptr(filepath.Join(leftSub, "a.txt")),
		Depth: 1,
	}
	e := &entry.Entry{
		Name:     "sub",
		IsDir:    true,
		Left:     &leftSub,
		Depth:    0,
		Children: []*entry.Entry{child},
	}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.copyEntry(e, 'l', 'r')

	wantRightSub := filepath.Join(rightRoot, "sub")
	if e.Right == nil || *e.Right != wantRightSub {
		t.Fatalf("e.Right = %v, want %q", e.Right, wantRightSub)
	}
	if _, err := os.Stat(filepath.Join(wantRightSub, "a.txt")); err != nil {
		t.Fatalf("copied nested file missing: %v", err)
	}
	if len(e.Children) != 1 {
		t.Fatalf("rebuilt Children = %+v, want 1 entry", e.Children)
	}
	rebuiltChild := e.Children[0]
	if rebuiltChild.Depth != 1 {
		t.Errorf("rebuilt child Depth = %d, want 1", rebuiltChild.Depth)
	}
	if rebuiltChild.Left == nil || rebuiltChild.Right == nil {
		t.Fatalf("rebuilt child should have both sides present: %+v", rebuiltChild)
	}
	if rebuiltChild.Compare != entry.Same {
		t.Errorf("rebuilt child Compare = %v, want Same", rebuiltChild.Compare)
	}

	// Regression check for the bug reported 2026-07-18: m.flat is a
	// separately-maintained flattened cache, not re-derived automatically
	// just because e.Children changed underneath it. A stale m.flat here
	// would still hold the *old* child object (Right still nil), even
	// though len(m.flat) happens to match by coincidence in this fixture.
	if len(m.flat) != 2 {
		t.Fatalf("m.flat = %+v, want 2 entries (dir + rebuilt child)", m.flat)
	}
	if m.flat[1] != rebuiltChild {
		t.Errorf("m.flat[1] is a stale child object, not the rebuilt one")
	}
	if m.flat[1].Right == nil {
		t.Errorf("m.flat[1].Right is nil — UI is still showing the pre-copy tree")
	}
}

// Closer to the exact report: a directory present on both sides but empty
// on the destination-to-be side gains contents after copying from the
// side that has them. Before the fix, m.flat kept the old, empty flat
// list until an unrelated collapse/expand happened to call reflatten.
func TestCopyEntry_Directory_NewContentsVisibleInFlatWithoutCollapseExpand(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub")) // present on right too, but empty
	writeFile(t, leftRoot, "sub/a.txt", "content\n")
	leftSub := filepath.Join(leftRoot, "sub")
	rightSub := filepath.Join(rightRoot, "sub")

	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub} // no children yet
	sibling := &entry.Entry{Name: "zzz"}                                         // just to prove the rest of the tree is untouched

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e, sibling})
	if len(m.flat) != 2 {
		t.Fatalf("setup: m.flat = %+v, want 2 entries before the copy", m.flat)
	}

	m.copyEntry(e, 'l', 'r')

	if len(e.Children) != 1 {
		t.Fatalf("e.Children = %+v, want 1 entry (a.txt)", e.Children)
	}
	if len(m.flat) != 3 {
		t.Fatalf("m.flat = %+v, want 3 entries (dir + new child + sibling) without needing collapse/expand", m.flat)
	}
	if m.flat[1].Name != "a.txt" {
		t.Errorf("m.flat[1].Name = %q, want %q", m.flat[1].Name, "a.txt")
	}
	if m.flat[2] != sibling {
		t.Errorf("sibling should still be present and untouched at m.flat[2]")
	}
}

// Regression test for the bug reported 2026-07-18: a file nested two
// levels into a subdirectory that was never enumerated at all on the
// destination side (dirtest/a/a/ exists but is empty; dirtest/b and
// dirtest/c both have a/sub/file.txt) failed to copy, silently, because
// `cp -R` refuses to create missing intermediate destination directories.
func TestCopyEntry_CopyingIntoWhollyMissingIntermediateDirectory(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "a")) // present, but empty — no "sub" at all
	mkdirAll(t, filepath.Join(rightRoot, "a", "sub"))
	rightFile := writeFile(t, rightRoot, "a/sub/file.txt", "content\n")

	// Mirrors what BuildPair would produce: "a" present on both sides,
	// its child "sub" present on the right only, "file.txt" nested under
	// that, present on the right only.
	fileEntry := &entry.Entry{Name: "file.txt", Right: &rightFile, Depth: 2}
	subEntry := &entry.Entry{
		Name: "sub", IsDir: true, Depth: 1,
		Right:    strptr(filepath.Join(rightRoot, "a", "sub")),
		Children: []*entry.Entry{fileEntry},
	}
	aLeft := filepath.Join(leftRoot, "a")
	aRight := filepath.Join(rightRoot, "a")
	aEntry := &entry.Entry{
		Name: "a", IsDir: true, Depth: 0,
		Left: &aLeft, Right: &aRight,
		Children: []*entry.Entry{subEntry},
	}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{aEntry})
	m.copyEntry(fileEntry, 'r', 'l')

	wantDest := filepath.Join(leftRoot, "a", "sub", "file.txt")
	if fileEntry.Left == nil || *fileEntry.Left != wantDest {
		t.Fatalf("fileEntry.Left = %v, want %q (flash: %q)", fileEntry.Left, wantDest, m.flash)
	}
	got, err := os.ReadFile(wantDest)
	if err != nil {
		t.Fatalf("copied file missing: %v (flash: %q)", err, m.flash)
	}
	if string(got) != "content\n" {
		t.Errorf("copied content = %q, want %q", got, "content\n")
	}
	if fileEntry.Compare == entry.CompareError {
		t.Errorf("copy should have succeeded, not errored: flash=%q", m.flash)
	}
}

func TestCopyEntry_NoOpWhenSourceAbsent(t *testing.T) {
	rightRoot := t.TempDir()
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Right: &rightPath}

	m := newTestModel(2, t.TempDir(), "", rightRoot, []*entry.Entry{e})
	m.copyEntry(e, 'l', 'r') // left is absent — should no-op

	if e.Left != nil {
		t.Errorf("e.Left should remain nil, got %v", e.Left)
	}
	if e.Compare != entry.Uncompared {
		t.Errorf("Compare = %v, want Uncompared (untouched)", e.Compare)
	}
}

func TestCopyEntry_ErrorMarksSubtreeError(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	// Left points at a path that doesn't actually exist on disk, simulating
	// a stale entry — cp should fail.
	stalePath := filepath.Join(leftRoot, "does-not-exist")
	child := &entry.Entry{Name: "child"}
	e := &entry.Entry{Name: "does-not-exist", Left: &stalePath, Children: []*entry.Entry{child}}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.copyEntry(e, 'l', 'r')

	if e.Compare != entry.CompareError {
		t.Errorf("Compare = %v, want CompareError", e.Compare)
	}
	if child.Compare != entry.CompareError {
		t.Errorf("child Compare = %v, want CompareError (propagated)", child.Compare)
	}
	if e.Right != nil {
		t.Errorf("e.Right should remain nil after a failed copy, got %v", e.Right)
	}
	if m.flash == "" {
		t.Error("flash should explain that the copy failed")
	}
}

func TestDeleteEntry_RemovesFileAndSplicesFromTree(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.deleteEntry(e)

	if _, err := os.Stat(leftPath); !os.IsNotExist(err) {
		t.Errorf("left file should be gone, err=%v", err)
	}
	if _, err := os.Stat(rightPath); !os.IsNotExist(err) {
		t.Errorf("right file should be gone, err=%v", err)
	}
	if len(m.entries) != 0 {
		t.Errorf("entries should be empty after delete, got %+v", m.entries)
	}
	if len(m.flat) != 0 {
		t.Errorf("flat should be empty after delete, got %+v", m.flat)
	}
}

func TestDeleteEntry_RemovesDirectoryRecursively(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub"))
	writeFile(t, leftRoot, "sub/a.txt", "content\n")
	writeFile(t, rightRoot, "sub/a.txt", "content\n")
	leftSub := filepath.Join(leftRoot, "sub")
	rightSub := filepath.Join(rightRoot, "sub")
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.deleteEntry(e)

	if _, err := os.Stat(leftSub); !os.IsNotExist(err) {
		t.Errorf("left dir should be gone, err=%v", err)
	}
	if _, err := os.Stat(rightSub); !os.IsNotExist(err) {
		t.Errorf("right dir should be gone, err=%v", err)
	}
	if len(m.entries) != 0 {
		t.Errorf("entries should be empty after delete, got %+v", m.entries)
	}
}

func TestDeleteEntry_OneSideAbsent(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath} // Right absent

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.deleteEntry(e)

	if _, err := os.Stat(leftPath); !os.IsNotExist(err) {
		t.Errorf("left file should be gone, err=%v", err)
	}
	if len(m.entries) != 0 {
		t.Errorf("entries should be empty after delete, got %+v", m.entries)
	}
}

func TestDeleteEntry_NestedEntry_SplicedFromParentChildren(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub"))
	leftChildPath := writeFile(t, leftRoot, "sub/a.txt", "content\n")
	rightChildPath := writeFile(t, rightRoot, "sub/a.txt", "content\n")
	leftSub := filepath.Join(leftRoot, "sub")
	rightSub := filepath.Join(rightRoot, "sub")

	child := &entry.Entry{Name: "a.txt", Left: &leftChildPath, Right: &rightChildPath}
	parent := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub, Children: []*entry.Entry{child}}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{parent})
	m.deleteEntry(child)

	if len(m.entries) != 1 {
		t.Fatalf("parent should remain, got %+v", m.entries)
	}
	if len(m.entries[0].Children) != 0 {
		t.Errorf("child should be spliced from parent.Children, got %+v", m.entries[0].Children)
	}
	if _, err := os.Stat(leftSub); err != nil {
		t.Errorf("parent dir should still exist: %v", err)
	}
}

func TestRemoveEntry_TopLevel(t *testing.T) {
	a := &entry.Entry{Name: "a"}
	b := &entry.Entry{Name: "b"}
	result := removeEntry([]*entry.Entry{a, b}, a)
	if len(result) != 1 || result[0] != b {
		t.Fatalf("got %+v, want just b", result)
	}
}

func TestRemoveEntry_NestedChild(t *testing.T) {
	child := &entry.Entry{Name: "child"}
	parent := &entry.Entry{Name: "parent", Children: []*entry.Entry{child}}
	sibling := &entry.Entry{Name: "sibling"}
	entries := []*entry.Entry{parent, sibling}

	result := removeEntry(entries, child)
	if len(result) != 2 {
		t.Fatalf("top-level entries changed unexpectedly: %+v", result)
	}
	if len(result[0].Children) != 0 {
		t.Errorf("child was not removed from parent.Children: %+v", result[0].Children)
	}
}
