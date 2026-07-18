package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

func TestSeparatorStyle_MatchingColumnsUseTheirColor(t *testing.T) {
	// styleDirArrow deliberately excluded: it carries no background at
	// all (it's an accent color for the arrow glyph only, never a
	// whole-row style), so it doesn't represent a "real" shared highlight
	// the way these do.
	cases := []lipgloss.Style{styleUnique, styleChanged, styleError, styleCursor}
	for _, s := range cases {
		got := separatorStyle(s, s)
		if got.GetBackground() != s.GetBackground() {
			t.Errorf("separatorStyle(%v, %v) background = %v, want %v",
				s, s, got.GetBackground(), s.GetBackground())
		}
	}
}

func TestSeparatorStyle_MismatchedColumnsStayNeutral(t *testing.T) {
	cases := []struct{ left, right lipgloss.Style }{
		{styleUnique, styleChanged},
		{styleUnique, styleNormal},
		{styleChanged, styleError},
		{styleDirArrow, styleUnique},
	}
	for _, tc := range cases {
		got := separatorStyle(tc.left, tc.right)
		if got.GetBackground() != styleSep.GetBackground() {
			t.Errorf("separatorStyle(left=%v, right=%v) background = %v, want the neutral styleSep background %v",
				tc.left, tc.right, got.GetBackground(), styleSep.GetBackground())
		}
	}
}

// Regression test for the bug reported 2026-07-18: separators between two
// plain/unstyled columns (the ordinary case — most rows aren't
// green/blue/cursor/error) were rendering white instead of neutral gray,
// because two unset backgrounds compared equal under GetBackground() and
// were treated as "sharing a color." Two columns not having a color isn't
// the same as two columns sharing one.
func TestSeparatorStyle_BothNormalStaysNeutralNotWhite(t *testing.T) {
	got := separatorStyle(styleNormal, styleNormal)
	if got.GetForeground() != styleSep.GetForeground() {
		t.Errorf("separatorStyle(styleNormal, styleNormal) foreground = %v, want the neutral styleSep foreground %v (not styleNormal's white)",
			got.GetForeground(), styleSep.GetForeground())
	}
	if got.GetBackground() != styleSep.GetBackground() {
		t.Errorf("separatorStyle(styleNormal, styleNormal) background = %v, want styleSep's %v",
			got.GetBackground(), styleSep.GetBackground())
	}
}

// The following cover the bug reported 2026-07-18: an entire directory row
// rendered yellow, name included. Verified against Python source
// (Settings.py): dir_arrow_fg is always 226 (yellow) in every category,
// but filename_fg varies by category — only the arrow glyph should be
// yellow, never the name.

func TestRowCols_DirectoryPresentEverywhereUsesNormalBaseStyle(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub"))
	leftSub := filepath.Join(leftRoot, "sub")
	rightSub := filepath.Join(rightRoot, "sub")
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	_, styles := m.rowCols(0, false /* not the cursor row */)

	for i, s := range styles {
		if s.GetForeground() != styleNormal.GetForeground() || s.GetBackground() != styleNormal.GetBackground() {
			t.Errorf("column %d style = %v, want styleNormal (the directory arrow color is applied separately, in renderCell)", i, s)
		}
	}
}

// withForcedColorProfile forces lipgloss to actually emit ANSI codes for
// the duration of the test, restoring the original profile afterward.
// Needed because Render() silently strips color when go test isn't
// attached to a real terminal, which would make it impossible to inspect
// renderCell's output for the color split it's supposed to produce.
func withForcedColorProfile(t *testing.T) {
	t.Helper()
	original := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(original) })
}

func TestRenderCell_DirectoryArrowIsYellowButNameIsNot(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Depth: 0}
	got := renderCell("v sub", 20, styleNormal, e)

	wantArrow := styleNormal.Foreground(lipgloss.Color("226")).Render("v ")
	wantRest := styleNormal.Render("sub" + strings.Repeat(" ", 15))
	want := styleNormal.Render("") + wantArrow + wantRest
	if got != want {
		t.Errorf("renderCell =\n%q\nwant\n%q", got, want)
	}
	// Sanity check on the property this test is actually about: the
	// yellow SGR code (256-color 226) appears before "sub", and "sub"
	// itself is not wrapped in that color.
	yellowIdx := strings.Index(got, "38;5;226")
	subIdx := strings.Index(got, "sub")
	if yellowIdx == -1 {
		t.Fatalf("no yellow (38;5;226) SGR code found in %q", got)
	}
	if !(yellowIdx < subIdx) {
		t.Errorf("yellow code should appear before the name, got yellowIdx=%d subIdx=%d in %q", yellowIdx, subIdx, got)
	}
	// The segment rendering "sub" itself must not carry the yellow code —
	// find the reset immediately before "sub" and confirm no 226 between it
	// and "sub".
	resetBeforeSub := strings.LastIndex(got[:subIdx], "\x1b[0m")
	segment := got[resetBeforeSub:subIdx]
	if strings.Contains(segment, "226") {
		t.Errorf("the filename segment should not use the yellow arrow color, got %q", segment)
	}
}

func TestRenderCell_NonDirectoryUsesPlainStyle(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "file.txt", IsDir: false}
	got := renderCell("file.txt", 20, styleNormal, e)
	want := styleNormal.Render(fit("file.txt", 20))
	if got != want {
		t.Errorf("renderCell for a non-directory =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderCell_CompareErrorDirectorySkipsArrowOverride(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Compare: entry.CompareError}
	got := renderCell("v sub", 20, styleError, e)
	want := styleError.Render(fit("v sub", 20))
	if got != want {
		t.Errorf("renderCell for a CompareError directory =\n%q\nwant\n%q (should be one uniform error-colored block, no yellow arrow)",
			got, want)
	}
}
