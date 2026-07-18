package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"umerge/internal/entry"
)

func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestUpdate_TwoWay_KeyACopiesLeftToRight(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "hello\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	wantDest := filepath.Join(rightRoot, "file.txt")
	if e.Right == nil || *e.Right != wantDest {
		t.Fatalf("e.Right = %v, want %q", e.Right, wantDest)
	}
	if _, err := os.Stat(wantDest); err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
}

func TestUpdate_TwoWay_KeyBCopiesRightToLeft(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	rightPath := writeFile(t, rightRoot, "file.txt", "hello\n")
	e := &entry.Entry{Name: "file.txt", Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('b'))
	m = updated.(Model)

	wantDest := filepath.Join(leftRoot, "file.txt")
	if e.Left == nil || *e.Left != wantDest {
		t.Fatalf("e.Left = %v, want %q", e.Left, wantDest)
	}
}

func TestUpdate_KeyDDeletesCurrentItem(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('d'))
	m = updated.(Model)

	if _, err := os.Stat(leftPath); !os.IsNotExist(err) {
		t.Errorf("left file should be gone, err=%v", err)
	}
	if len(m.entries) != 0 {
		t.Errorf("entry should be spliced out, got %+v", m.entries)
	}
}

func TestUpdate_ThreeWay_KeyASetsPendingPrompt(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	if m.pendingCopyFrom != 'a' {
		t.Errorf("pendingCopyFrom = %q, want 'a'", m.pendingCopyFrom)
	}
	if m.prompt == "" {
		t.Error("prompt should be set while awaiting a destination choice")
	}
	// No copy should have happened yet — it's a two-step prompt.
	if e.Right != nil || e.Middle != nil {
		t.Errorf("copy should not run until the destination is chosen: %+v", e)
	}
}

func TestUpdate_ThreeWay_PendingThenValidDestinationCopies(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a')) // "Copy from A (left) to:"
	m = updated.(Model)
	updated, _ = m.Update(keyMsg('c')) // destination: middle
	m = updated.(Model)

	if m.pendingCopyFrom != 0 || m.prompt != "" {
		t.Errorf("prompt state should be cleared after resolving, got pendingCopyFrom=%q prompt=%q",
			m.pendingCopyFrom, m.prompt)
	}
	wantDest := filepath.Join(middleRoot, "file.txt")
	if e.Middle == nil || *e.Middle != wantDest {
		t.Fatalf("e.Middle = %v, want %q", e.Middle, wantDest)
	}
}

func TestUpdate_ThreeWay_PendingThenInvalidKeyCancels(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)
	updated, _ = m.Update(keyMsg('a')) // same column as source — invalid
	m = updated.(Model)

	if m.pendingCopyFrom != 0 || m.prompt != "" {
		t.Errorf("prompt state should be cleared even on an invalid choice, got pendingCopyFrom=%q prompt=%q",
			m.pendingCopyFrom, m.prompt)
	}
	if e.Middle != nil || e.Right != nil {
		t.Errorf("no copy should happen on an invalid destination choice: %+v", e)
	}
	if m.flash != "Invalid choice" {
		t.Errorf("flash = %q, want %q", m.flash, "Invalid choice")
	}

	// Normal key handling should resume afterward.
	updated, cmd := m.Update(keyMsg('q'))
	m = updated.(Model)
	if cmd == nil {
		t.Error("q should resume normal handling and return the quit command")
	}
}

func TestUpdate_ThreeWay_PendingIgnoresOtherKeys(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.cursor = 0
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)
	if m.pendingCopyFrom != 'a' {
		t.Fatalf("setup: pendingCopyFrom = %q, want 'a' (left is present, so the prompt should start)", m.pendingCopyFrom)
	}

	// While pending, movement keys must not fall through to normal handling.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.pendingCopyFrom != 0 {
		t.Errorf("pendingCopyFrom should be cleared (any key resolves or cancels the prompt), got %q", m.pendingCopyFrom)
	}
}

// The following cover the bug reported 2026-07-18: copying from an absent
// source silently did nothing, in both directions ("b then c" when right
// was absent, and "c then a" when middle was absent). Fixed by validating
// the source the moment the first letter is pressed, in both 2-way and
// 3-way mode, with a visible flash message instead of a silent no-op.

func TestUpdate_TwoWay_KeyANoOpWhenLeftAbsent(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Right: &rightPath} // Left absent

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	if e.Left != nil {
		t.Errorf("no copy should have happened, e.Left = %v", e.Left)
	}
	if m.flash == "" {
		t.Error("flash should explain why nothing happened")
	}
}

func TestUpdate_ThreeWay_KeyBNoOpWhenRightAbsent(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath} // Right absent

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('b'))
	m = updated.(Model)

	if m.pendingCopyFrom != 0 {
		t.Errorf("should not enter the destination prompt when the source is absent, got pendingCopyFrom=%q", m.pendingCopyFrom)
	}
	if m.prompt != "" {
		t.Errorf("prompt should not be set, got %q", m.prompt)
	}
	if m.flash != "Nothing to copy: right is absent" {
		t.Errorf("flash = %q, want %q", m.flash, "Nothing to copy: right is absent")
	}
}

func TestUpdate_ThreeWay_KeyCNoOpWhenMiddleAbsent(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath} // Middle absent

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('c'))
	m = updated.(Model)

	if m.pendingCopyFrom != 0 {
		t.Errorf("should not enter the destination prompt when the source is absent, got pendingCopyFrom=%q", m.pendingCopyFrom)
	}
	if m.flash != "Nothing to copy: middle is absent" {
		t.Errorf("flash = %q, want %q", m.flash, "Nothing to copy: middle is absent")
	}

	// e.g. pressing 'a' next (a valid, present source) should work normally —
	// the earlier no-op must not have left the model in a broken state.
	updated, _ = m.Update(keyMsg('a'))
	m = updated.(Model)
	if m.pendingCopyFrom != 'a' {
		t.Errorf("subsequent valid source key should still work, got pendingCopyFrom=%q", m.pendingCopyFrom)
	}
}

func TestUpdate_ThreeWay_KeyCWorksWhenMiddlePresent(t *testing.T) {
	// Regression check for the exact user report: entry present only in
	// the middle/parent, copy from C (middle, present) to A (left, absent)
	// should succeed, not no-op.
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	middlePath := writeFile(t, middleRoot, "file.txt", "middle only content\n")
	e := &entry.Entry{Name: "file.txt", Middle: &middlePath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('c'))
	m = updated.(Model)
	if m.pendingCopyFrom != 'c' {
		t.Fatalf("pendingCopyFrom = %q, want 'c'", m.pendingCopyFrom)
	}
	updated, _ = m.Update(keyMsg('a'))
	m = updated.(Model)

	wantDest := filepath.Join(leftRoot, "file.txt")
	if e.Left == nil || *e.Left != wantDest {
		t.Fatalf("e.Left = %v, want %q", e.Left, wantDest)
	}
	got, err := os.ReadFile(wantDest)
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != "middle only content\n" {
		t.Errorf("copied content = %q, want %q", got, "middle only content\n")
	}
}

func TestRowCols_CompareErrorRendersDistinctlyFromNormal(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath, Compare: entry.CompareError}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	_, styles := m.rowCols(0, false /* not the cursor row */)

	// lipgloss.Style isn't comparable with == and Render() strips color
	// when go test isn't attached to a real terminal (no color profile
	// detected), so compare the configured background color directly.
	wantBG := styleError.GetBackground()
	normalBG := styleNormal.GetBackground()
	for i, s := range styles {
		got := s.GetBackground()
		if got == normalBG {
			t.Errorf("column %d has styleNormal's background, want a distinct error style", i)
		}
		if got != wantBG {
			t.Errorf("column %d background = %v, want styleError's %v", i, got, wantBG)
		}
	}
}
