package entry

import (
	"os"
	"path/filepath"
	"testing"
)

func mkfile(t *testing.T, root, rel string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("content of "+rel), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdirOnly(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		t.Fatal(err)
	}
}

// findByName returns the entry with the given name, or nil.
func findByName(entries []*Entry, name string) *Entry {
	for _, e := range entries {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func TestBuildPair_OneSideOnly(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	mkfile(t, left, "a.txt")
	mkfile(t, right, "b.txt")
	mkfile(t, left, "common.txt")
	mkfile(t, right, "common.txt")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	a := findByName(entries, "a.txt")
	if a == nil || a.Left == nil || a.Right != nil {
		t.Errorf("a.txt: want Left set, Right nil; got %+v", a)
	}

	b := findByName(entries, "b.txt")
	if b == nil || b.Left != nil || b.Right == nil {
		t.Errorf("b.txt: want Left nil, Right set; got %+v", b)
	}

	c := findByName(entries, "common.txt")
	if c == nil || c.Left == nil || c.Right == nil {
		t.Errorf("common.txt: want both sides set; got %+v", c)
	}
}

func TestBuildPair_InterleavedSortOrder(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	mkfile(t, left, "b")
	mkfile(t, left, "d")
	mkfile(t, right, "a")
	mkfile(t, right, "c")
	mkfile(t, right, "d")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}

	wantOrder := []string{"a", "b", "c", "d"}
	if len(entries) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d", len(entries), len(wantOrder))
	}
	for i, name := range wantOrder {
		if entries[i].Name != name {
			t.Errorf("entries[%d].Name = %q, want %q", i, entries[i].Name, name)
		}
	}

	// Spot-check sidedness for the interleaved names.
	if e := findByName(entries, "a"); e.Left != nil || e.Right == nil {
		t.Errorf("a: want right-only, got %+v", e)
	}
	if e := findByName(entries, "d"); e.Left == nil || e.Right == nil {
		t.Errorf("d: want both sides, got %+v", e)
	}
}

func TestBuildPair_NestedDirsAndDepth(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	mkfile(t, left, "sub/nested.txt")
	mkfile(t, right, "sub/nested.txt")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d top-level entries, want 1", len(entries))
	}

	sub := entries[0]
	if sub.Name != "sub" || !sub.IsDir || sub.Depth != 0 {
		t.Fatalf("sub entry wrong: %+v", sub)
	}
	if len(sub.Children) != 1 {
		t.Fatalf("sub has %d children, want 1", len(sub.Children))
	}
	nested := sub.Children[0]
	if nested.Name != "nested.txt" || nested.IsDir || nested.Depth != 1 {
		t.Fatalf("nested entry wrong: %+v", nested)
	}
	if nested.Left == nil || nested.Right == nil {
		t.Fatalf("nested entry should have both sides set: %+v", nested)
	}
}

func TestBuildPair_MissingRoot(t *testing.T) {
	right := t.TempDir()
	mkfile(t, right, "only.txt")

	entries, err := BuildPair(filepath.Join(right, "does-not-exist"), right, nil)
	if err != nil {
		t.Fatalf("BuildPair should not error on a missing root, got: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "only.txt" {
		t.Fatalf("got %+v, want single only.txt entry", entries)
	}
	if entries[0].Left != nil {
		t.Errorf("Left should be nil when the left root doesn't exist")
	}
}

func TestBuildPair_EmptyDirs(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestBuildTriple_AllThreePresent(t *testing.T) {
	left, middle, right := t.TempDir(), t.TempDir(), t.TempDir()
	mkfile(t, left, "f.txt")
	mkfile(t, middle, "f.txt")
	mkfile(t, right, "f.txt")

	entries, err := BuildTriple(left, middle, right, nil)
	if err != nil {
		t.Fatalf("BuildTriple: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Left == nil || e.Middle == nil || e.Right == nil {
		t.Fatalf("expected all three sides set: %+v", e)
	}
}

func TestBuildTriple_RemovedAndInserted(t *testing.T) {
	left, middle, right := t.TempDir(), t.TempDir(), t.TempDir()
	// "removed.txt" only in the parent/middle — removed from both children.
	mkfile(t, middle, "removed.txt")
	// "inserted.txt" only in left+right — absent from the parent/middle.
	mkfile(t, left, "inserted.txt")
	mkfile(t, right, "inserted.txt")

	entries, err := BuildTriple(left, middle, right, nil)
	if err != nil {
		t.Fatalf("BuildTriple: %v", err)
	}

	removed := findByName(entries, "removed.txt")
	if removed == nil || removed.Middle == nil || removed.Left != nil || removed.Right != nil {
		t.Errorf("removed.txt: want middle-only, got %+v", removed)
	}

	inserted := findByName(entries, "inserted.txt")
	if inserted == nil || inserted.Middle != nil || inserted.Left == nil || inserted.Right == nil {
		t.Errorf("inserted.txt: want left+right only, got %+v", inserted)
	}
}

func TestBuildTriple_InterleavedSortOrder(t *testing.T) {
	left, middle, right := t.TempDir(), t.TempDir(), t.TempDir()
	mkfile(t, left, "x")
	mkfile(t, middle, "y")
	mkfile(t, right, "z")

	entries, err := BuildTriple(left, middle, right, nil)
	if err != nil {
		t.Fatalf("BuildTriple: %v", err)
	}
	wantOrder := []string{"x", "y", "z"}
	if len(entries) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d", len(entries), len(wantOrder))
	}
	for i, name := range wantOrder {
		if entries[i].Name != name {
			t.Errorf("entries[%d].Name = %q, want %q", i, entries[i].Name, name)
		}
	}
}

func TestFlatten_FlatSiblingsOnly(t *testing.T) {
	entries := []*Entry{
		{Name: "a"},
		{Name: "b"},
	}
	flat := Flatten(entries, nil)
	if len(flat) != 2 || flat[0].Name != "a" || flat[1].Name != "b" {
		t.Fatalf("got %+v", flat)
	}
}

func TestFlatten_ExpandedDirIncludesChildren(t *testing.T) {
	entries := []*Entry{
		{
			Name:  "dir",
			IsDir: true,
			Children: []*Entry{
				{Name: "child1"},
				{Name: "child2"},
			},
		},
		{Name: "z"},
	}
	flat := Flatten(entries, nil)
	wantOrder := []string{"dir", "child1", "child2", "z"}
	if len(flat) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d: %+v", len(flat), len(wantOrder), flat)
	}
	for i, name := range wantOrder {
		if flat[i].Name != name {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, name)
		}
	}
}

func TestFlatten_CollapsedDirSkipsChildren(t *testing.T) {
	entries := []*Entry{
		{
			Name:      "dir",
			IsDir:     true,
			Collapsed: true,
			Children: []*Entry{
				{Name: "child1"},
			},
		},
		{Name: "z"},
	}
	flat := Flatten(entries, nil)
	wantOrder := []string{"dir", "z"}
	if len(flat) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d: %+v", len(flat), len(wantOrder), flat)
	}
	for i, name := range wantOrder {
		if flat[i].Name != name {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, name)
		}
	}
}

func TestFlatten_MixedCollapsedAndExpandedSiblings(t *testing.T) {
	entries := []*Entry{
		{
			Name:      "collapsed",
			IsDir:     true,
			Collapsed: true,
			Children:  []*Entry{{Name: "hidden-child"}},
		},
		{
			Name:  "expanded",
			IsDir: true,
			Children: []*Entry{
				{Name: "shown-child"},
			},
		},
	}
	flat := Flatten(entries, nil)
	wantOrder := []string{"collapsed", "expanded", "shown-child"}
	if len(flat) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d: %+v", len(flat), len(wantOrder), flat)
	}
	for i, name := range wantOrder {
		if flat[i].Name != name {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, name)
		}
	}
}

func TestBuildPair_ParentPointerSetForNestedEntries(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	mkfile(t, left, "sub/nested.txt")
	mkfile(t, right, "sub/nested.txt")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}

	sub := entries[0]
	if sub.Parent != nil {
		t.Fatalf("top-level entry's Parent = %+v, want nil", sub.Parent)
	}
	nested := sub.Children[0]
	if nested.Parent != sub {
		t.Fatalf("nested.Parent = %p, want sub (%p)", nested.Parent, sub)
	}
}

func TestSetHidden_PropagatesToWholeSubtree(t *testing.T) {
	child := &Entry{Name: "child"}
	grandchild := &Entry{Name: "grandchild"}
	child.Children = []*Entry{grandchild}
	root := &Entry{Name: "root", IsDir: true, Children: []*Entry{child}}

	root.SetHidden(true)

	if !root.Hidden || !child.Hidden || !grandchild.Hidden {
		t.Fatalf("SetHidden(true) didn't reach whole subtree: root=%v child=%v grandchild=%v",
			root.Hidden, child.Hidden, grandchild.Hidden)
	}
}

func TestSetHidden_DescendantCanBeUnhiddenIndependently(t *testing.T) {
	child := &Entry{Name: "child"}
	root := &Entry{Name: "root", IsDir: true, Children: []*Entry{child}}

	root.SetHidden(true)
	child.SetHidden(false)

	if !root.Hidden {
		t.Fatalf("root.Hidden = false, want true (unhiding a child shouldn't touch its ancestor)")
	}
	if child.Hidden {
		t.Fatalf("child.Hidden = true, want false")
	}
}

func TestFlatten_SkipOmitsEntryButStillDescendsIntoChildren(t *testing.T) {
	// A directory whose Hidden flag wasn't propagated to one child (as if
	// that child was independently un-hidden after the fact): the
	// directory's own line is skipped, but its non-hidden child still
	// renders on its own, matching the reference implementation's
	// per-line (not per-subtree) filtering.
	child := &Entry{Name: "child"}
	dir := &Entry{Name: "dir", IsDir: true, Hidden: true, Children: []*Entry{child}}
	entries := []*Entry{dir, {Name: "z"}}

	skip := func(e *Entry) bool { return e.Hidden }
	flat := Flatten(entries, skip)

	wantOrder := []string{"child", "z"}
	if len(flat) != len(wantOrder) {
		t.Fatalf("got %d entries, want %d: %+v", len(flat), len(wantOrder), flat)
	}
	for i, name := range wantOrder {
		if flat[i].Name != name {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, name)
		}
	}
}

func TestFlatten_NilSkipKeepsEveryEntry(t *testing.T) {
	entries := []*Entry{{Name: "a", Hidden: true}, {Name: "b"}}
	flat := Flatten(entries, nil)
	if len(flat) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(flat), flat)
	}
}

func TestBuildPair_EmptyDirOnOneSide(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	mkdirOnly(t, left, "emptydir")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Name != "emptydir" || !e.IsDir {
		t.Fatalf("emptydir entry wrong: %+v", e)
	}
	if e.Left == nil || e.Right != nil {
		t.Errorf("emptydir: want Left set, Right nil; got %+v", e)
	}
	if len(e.Children) != 0 {
		t.Errorf("emptydir should have no children, got %+v", e.Children)
	}
}
