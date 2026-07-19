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

// The following cover BinaryDifferent rendering, added 2026-07-19 alongside
// the fast short-circuit comparison and binary-file detection: two
// genuinely different binary files must render distinctly from "Same"
// (the bug that motivated this work), get the same "changed" blue as a
// real text diff, and show a "bin" marker instead of a hunk count, which
// wouldn't mean anything for binary content.

func TestRowCols_BinaryDifferentTwoWayGetsChangedStyle(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := filepath.Join(leftRoot, "file.bin")
	rightPath := filepath.Join(rightRoot, "file.bin")
	if err := os.WriteFile(leftPath, []byte{0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rightPath, []byte{0x00, 0xFF}, 0o644); err != nil {
		t.Fatal(err)
	}
	e := &entry.Entry{Name: "file.bin", Left: &leftPath, Right: &rightPath, Compare: entry.BinaryDifferent}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	_, styles := m.rowCols(0, false)

	wantBG := styleChanged.GetBackground()
	for i, s := range styles {
		if s.GetBackground() != wantBG {
			t.Errorf("column %d background = %v, want styleChanged's %v", i, s.GetBackground(), wantBG)
		}
		if s.GetBackground() == styleNormal.GetBackground() {
			t.Errorf("column %d rendered as styleNormal — a BinaryDifferent entry must not look like Same", i)
		}
	}
}

func TestRowCols_BinaryDifferentThreeWayAllColumnsChanged(t *testing.T) {
	// BinaryDifferent entries never have LMDiffs/MRDiffs set, so
	// diffStyleForCol's normal per-pair logic (which would see zero
	// counts and wrongly conclude "unchanged") must be bypassed for this
	// state — all three columns should show as changed.
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath, middlePath, rightPath := filepath.Join(leftRoot, "f"), filepath.Join(middleRoot, "f"), filepath.Join(rightRoot, "f")
	e := &entry.Entry{
		Name: "f", Left: &leftPath, Middle: &middlePath, Right: &rightPath,
		Compare: entry.BinaryDifferent, LMDiffs: 0, MRDiffs: 0,
	}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	_, styles := m.rowCols(0, false)

	wantBG := styleChanged.GetBackground()
	for i, s := range styles {
		if s.GetBackground() != wantBG {
			t.Errorf("column %d background = %v, want styleChanged's %v (BinaryDifferent should mark every column, not rely on LM/MR counts)",
				i, s.GetBackground(), wantBG)
		}
	}
}

func TestDiffCounts_BinaryDifferentReturnsNoNumericCount(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	e := &entry.Entry{Name: "file.bin", Compare: entry.BinaryDifferent, NumDiffs: 0}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	counts := m.diffCounts(e)
	for i, c := range counts {
		if c != nil {
			t.Errorf("counts[%d] = %v, want nil — a hunk count doesn't apply to binary content", i, *c)
		}
	}
}

func TestEntryText_BinaryDifferentShowsBinMarker(t *testing.T) {
	path := "/left/file.bin"
	e := &entry.Entry{Name: "file.bin", Left: &path, Compare: entry.BinaryDifferent}

	got := entryText(e, &path, nil, false)
	want := "  file.bin bin"
	if got != want {
		t.Errorf("entryText = %q, want %q", got, want)
	}
}

// Default-flip decided 2026-07-18 (see TODO.md Priority 9, CLAUDE.md):
// Unicode tree symbols (▶/▼) are now the default, with -A/--ascii as the
// fallback for terminals that render the ambiguous-width glyphs
// incorrectly (confirmed: COSMIC terminal; confirmed fine: WezTerm).

func TestEntryText_UnicodeArrowsByDefault(t *testing.T) {
	path := "/left/sub"
	collapsed := &entry.Entry{Name: "sub", IsDir: true, Collapsed: true, Left: &path}
	expanded := &entry.Entry{Name: "sub", IsDir: true, Collapsed: false, Left: &path}

	if got := entryText(collapsed, &path, nil, false); got != "▶ sub" {
		t.Errorf("collapsed, unicode: got %q, want %q", got, "▶ sub")
	}
	if got := entryText(expanded, &path, nil, false); got != "▼ sub" {
		t.Errorf("expanded, unicode: got %q, want %q", got, "▼ sub")
	}
}

func TestEntryText_ASCIIArrowsWhenRequested(t *testing.T) {
	path := "/left/sub"
	collapsed := &entry.Entry{Name: "sub", IsDir: true, Collapsed: true, Left: &path}
	expanded := &entry.Entry{Name: "sub", IsDir: true, Collapsed: false, Left: &path}

	if got := entryText(collapsed, &path, nil, true); got != "> sub" {
		t.Errorf("collapsed, ascii: got %q, want %q", got, "> sub")
	}
	if got := entryText(expanded, &path, nil, true); got != "v sub" {
		t.Errorf("expanded, ascii: got %q, want %q", got, "v sub")
	}
}

func TestEntryText_FileNeverGetsAnArrowEitherMode(t *testing.T) {
	path := "/left/file.txt"
	e := &entry.Entry{Name: "file.txt", IsDir: false, Left: &path}

	if got := entryText(e, &path, nil, false); got != "  file.txt" {
		t.Errorf("unicode mode: got %q, want %q", got, "  file.txt")
	}
	if got := entryText(e, &path, nil, true); got != "  file.txt" {
		t.Errorf("ascii mode: got %q, want %q", got, "  file.txt")
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

func TestRenderCell_DirectoryArrowIsYellowButNameIsNot_ASCII(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Depth: 0}
	got := renderCell("v sub", 20, styleNormal, e, true)

	wantArrow := styleNormal.Foreground(lipgloss.Color("226")).Render("v ")
	wantRest := styleNormal.Render("sub" + strings.Repeat(" ", 15))
	want := styleNormal.Render("") + wantArrow + wantRest
	if got != want {
		t.Errorf("renderCell =\n%q\nwant\n%q", got, want)
	}
	assertArrowYellowNameNot(t, got, "sub")
}

// Regression coverage for switching the default to Unicode arrows
// (2026-07-18): "▶"/"▼" are 3-byte UTF-8 sequences, unlike the 2-byte
// ASCII "> "/"v " — renderCell's byte-offset slicing has to account for
// that or it corrupts the split (or the string) instead of just getting
// the wrong column width.
func TestRenderCell_DirectoryArrowIsYellowButNameIsNot_Unicode(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Depth: 0, Collapsed: false}
	got := renderCell("▼ sub", 20, styleNormal, e, false)

	wantArrow := styleNormal.Foreground(lipgloss.Color("226")).Render("▼ ")
	wantRest := styleNormal.Render("sub" + strings.Repeat(" ", 15))
	want := styleNormal.Render("") + wantArrow + wantRest
	if got != want {
		t.Errorf("renderCell =\n%q\nwant\n%q", got, want)
	}
	assertArrowYellowNameNot(t, got, "sub")
}

// TestRenderCell_DirectoryWithIndent_Unicode covers a nonzero Depth
// combined with the (wider, non-ASCII-byte-length) Unicode arrow, since
// the indent and arrow byte offsets are computed independently.
func TestRenderCell_DirectoryWithIndent_Unicode(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Depth: 2, Collapsed: true}
	got := renderCell("    ▶ sub", 20, styleNormal, e, false)

	wantIndent := styleNormal.Render("    ")
	wantArrow := styleNormal.Foreground(lipgloss.Color("226")).Render("▶ ")
	wantRest := styleNormal.Render("sub" + strings.Repeat(" ", 11))
	want := wantIndent + wantArrow + wantRest
	if got != want {
		t.Errorf("renderCell =\n%q\nwant\n%q", got, want)
	}
	assertArrowYellowNameNot(t, got, "sub")
}

// assertArrowYellowNameNot checks that the yellow (256-color 226) SGR
// code appears before name in the rendered output, and that the segment
// rendering name itself doesn't carry that color.
func assertArrowYellowNameNot(t *testing.T, rendered, name string) {
	t.Helper()
	yellowIdx := strings.Index(rendered, "38;5;226")
	nameIdx := strings.Index(rendered, name)
	if yellowIdx == -1 {
		t.Fatalf("no yellow (38;5;226) SGR code found in %q", rendered)
	}
	if !(yellowIdx < nameIdx) {
		t.Errorf("yellow code should appear before the name, got yellowIdx=%d nameIdx=%d in %q", yellowIdx, nameIdx, rendered)
	}
	resetBeforeName := strings.LastIndex(rendered[:nameIdx], "\x1b[0m")
	segment := rendered[resetBeforeName:nameIdx]
	if strings.Contains(segment, "226") {
		t.Errorf("the filename segment should not use the yellow arrow color, got %q", segment)
	}
}

func TestRenderCell_NonDirectoryUsesPlainStyle(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "file.txt", IsDir: false}
	got := renderCell("file.txt", 20, styleNormal, e, false)
	want := styleNormal.Render(fit("file.txt", 20))
	if got != want {
		t.Errorf("renderCell for a non-directory =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderCell_CompareErrorDirectorySkipsArrowOverride(t *testing.T) {
	withForcedColorProfile(t)

	e := &entry.Entry{Name: "sub", IsDir: true, Compare: entry.CompareError}
	got := renderCell("v sub", 20, styleError, e, true)
	want := styleError.Render(fit("v sub", 20))
	if got != want {
		t.Errorf("renderCell for a CompareError directory =\n%q\nwant\n%q (should be one uniform error-colored block, no yellow arrow)",
			got, want)
	}
}

// The following cover --read-only (added 2026-07-19): copy/delete must be
// fully disabled, with a clear flash message rather than silently doing
// nothing — motivated by the symlink hazard when umerge is invoked as a
// `git difftool -d` backend (see TODO.md Priority 3).

func TestUpdate_ReadOnly_TwoWayCopyDisabled(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.readOnly = true
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	if e.Right != nil {
		t.Errorf("no copy should have happened in read-only mode, e.Right = %v", e.Right)
	}
	if m.flash == "" {
		t.Error("flash should explain that copy is disabled in read-only mode")
	}
}

func TestUpdate_ReadOnly_ThreeWayCopyPromptNeverStarts(t *testing.T) {
	leftRoot, middleRoot, rightRoot := t.TempDir(), t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(3, leftRoot, middleRoot, rightRoot, []*entry.Entry{e})
	m.readOnly = true
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	if m.pendingCopyFrom != 0 || m.prompt != "" {
		t.Errorf("the copy-destination prompt should never start in read-only mode, got pendingCopyFrom=%q prompt=%q",
			m.pendingCopyFrom, m.prompt)
	}
	if m.flash == "" {
		t.Error("flash should explain that copy is disabled in read-only mode")
	}
}

func TestUpdate_ReadOnly_DeleteDisabled(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.readOnly = true
	updated, _ := m.Update(keyMsg('d'))
	m = updated.(Model)

	if _, err := os.Stat(leftPath); err != nil {
		t.Errorf("left file should still exist in read-only mode, stat err = %v", err)
	}
	if len(m.entries) != 1 {
		t.Errorf("entry should not be spliced out in read-only mode, got %+v", m.entries)
	}
	if m.flash == "" {
		t.Error("flash should explain that delete is disabled in read-only mode")
	}
}

func TestUpdate_NotReadOnly_CopyAndDeleteStillWork(t *testing.T) {
	// Regression guard: make sure the read-only gate doesn't accidentally
	// block the default (non-read-only) case.
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, _ := m.Update(keyMsg('a'))
	m = updated.(Model)

	if e.Right == nil {
		t.Fatal("copy should have run when readOnly is false")
	}
}

// The following cover Priority 4 (refresh / re-compare), added 2026-07-19:
// the "r" key re-enumerates and re-compares the entry at the cursor in the
// background, and returning from vimdiff/ediff automatically re-compares
// the entry that was open.

// drainCompare drives m's Update loop with the Cmds a background compare
// produces (compareResultMsg, ..., compareDoneMsg) until m.comparing goes
// false, the same way the real Bubble Tea runtime would as messages
// arrive — up to a generous iteration cap as a safety net against a bug
// turning this into an infinite loop.
func drainCompare(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for i := 0; i < 50 && m.comparing; i++ {
		if cmd == nil {
			t.Fatal("comparing is true but no Cmd to listen for the next message")
		}
		msg := cmd()
		updated, next := m.Update(msg)
		m = updated.(Model)
		cmd = next
	}
	if m.comparing {
		t.Fatal("comparing never became false — drainCompare's iteration cap was hit")
	}
	return m
}

func TestUpdate_KeyR_RefreshesFileEntry(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "same\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "same\n")
	// Deliberately stale/wrong state, as if set before some external
	// change — refresh should correct it from what's actually on disk.
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath, Compare: entry.Different, NumDiffs: 5}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	updated, cmd := m.Update(keyMsg('r'))
	m = updated.(Model)

	if !m.comparing {
		t.Fatal("comparing should be true right after starting a refresh")
	}
	if cmd == nil {
		t.Fatal("expected a non-nil Cmd to listen for the compare result")
	}

	m = drainCompare(t, m, cmd)

	if e.Compare != entry.Same {
		t.Errorf("Compare = %v, want Same (refresh should correct the stale Different state)", e.Compare)
	}
	if e.NumDiffs != 0 {
		t.Errorf("NumDiffs = %d, want 0", e.NumDiffs)
	}
}

func TestUpdate_KeyR_RefreshesDirectoryEntry(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	mkdirAll(t, filepath.Join(leftRoot, "sub"))
	mkdirAll(t, filepath.Join(rightRoot, "sub"))
	leftSub := filepath.Join(leftRoot, "sub")
	rightSub := filepath.Join(rightRoot, "sub")
	e := &entry.Entry{Name: "sub", IsDir: true, Left: &leftSub, Right: &rightSub} // no children yet

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	if len(m.flat) != 1 {
		t.Fatalf("setup: expected 1 entry before refresh, got %d", len(m.flat))
	}

	// A file appears on disk after the tree was built — e.g. edited
	// outside umerge — which a stale in-memory tree wouldn't know about
	// until refreshed.
	writeFile(t, leftRoot, "sub/new.txt", "hello\n")
	writeFile(t, rightRoot, "sub/new.txt", "hello\n")

	updated, cmd := m.Update(keyMsg('r'))
	m = updated.(Model)

	if len(e.Children) != 1 || e.Children[0].Name != "new.txt" {
		t.Fatalf("rebuildChildren should have picked up the new file immediately, got %+v", e.Children)
	}
	if len(m.flat) != 2 {
		t.Fatalf("m.flat should include the newly-enumerated child without needing collapse/expand, got %+v", m.flat)
	}

	m = drainCompare(t, m, cmd)

	if e.Children[0].Compare != entry.Same {
		t.Errorf("new.txt Compare = %v, want Same", e.Children[0].Compare)
	}
}

func TestUpdate_KeyR_BlockedWhileComparing(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "content\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "content\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})
	m.comparing = true // simulate an in-progress comparison
	originalCh := m.compareCh

	updated, cmd := m.Update(keyMsg('r'))
	m = updated.(Model)

	if m.flash == "" {
		t.Error("flash should explain that a refresh can't start while already comparing")
	}
	if cmd != nil {
		t.Error("no new Cmd should be started while a comparison is already running")
	}
	if m.compareCh != originalCh {
		t.Error("compareCh should be untouched — starting a second compare would race with the one already running")
	}
}

func TestUpdate_ToolDoneMsg_RecomparesTheEditedEntry(t *testing.T) {
	leftRoot, rightRoot := t.TempDir(), t.TempDir()
	leftPath := writeFile(t, leftRoot, "file.txt", "one\n")
	rightPath := writeFile(t, rightRoot, "file.txt", "two\n")
	e := &entry.Entry{Name: "file.txt", Left: &leftPath, Right: &rightPath, Compare: entry.Different, NumDiffs: 1}

	m := newTestModel(2, leftRoot, "", rightRoot, []*entry.Entry{e})

	// Simulate the user having made the file match while it was open in
	// vimdiff.
	if err := os.WriteFile(rightPath, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	updated, _ := m.Update(toolDoneMsg{e: e})
	m = updated.(Model)

	if e.Compare != entry.Same {
		t.Errorf("Compare = %v, want Same (returning from the merge tool should trigger a recompare)", e.Compare)
	}
	if e.NumDiffs != 0 {
		t.Errorf("NumDiffs = %d, want 0", e.NumDiffs)
	}
}

func TestUpdate_ToolDoneMsg_NilEntryIsSafe(t *testing.T) {
	m := Model{}
	updated, cmd := m.Update(toolDoneMsg{})
	if _, ok := updated.(Model); !ok {
		t.Fatal("Update should still return a Model")
	}
	if cmd != nil {
		t.Errorf("expected a nil Cmd, got %v", cmd)
	}
}
