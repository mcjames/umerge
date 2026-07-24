package mergetool

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mcjames/umerge/internal/entry"
)

func strptr(s string) *string { return &s }

func TestCommand_NilForDirectory(t *testing.T) {
	e := &entry.Entry{IsDir: true, Left: strptr("/some/dir")}
	if cmd := Command(e, "vim"); cmd != nil {
		t.Fatalf("expected nil for a directory entry, got %+v", cmd)
	}
}

func TestCommand_NilWhenNoPathsPresent(t *testing.T) {
	e := &entry.Entry{}
	if cmd := Command(e, "vim"); cmd != nil {
		t.Fatalf("expected nil when no sides are present, got %+v", cmd)
	}
}

func TestCommand_VimOneFile(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a")}
	cmd := Command(e, "vim")
	want := []string{"vim", "/a"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

// The following cover recoloring vimdiff's built-in diff highlight groups
// to match umerge's own directory-view palette, so a file opened from the
// tree doesn't switch to an unrelated set of colors — the same idea as
// Araxis Merge using one consistent scheme for both its directory and file
// comparison views.

func TestCommand_VimTwoFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}
	cmd := Command(e, "vim")
	want := []string{
		"vimdiff",
		"-c", "highlight DiffChange ctermbg=153 ctermfg=black guibg=#a6caf0 guifg=black",
		"-c", "highlight DiffText ctermbg=153 ctermfg=black cterm=bold guibg=#a6caf0 guifg=black gui=bold",
		"-c", "highlight DiffAdd ctermbg=151 ctermfg=black guibg=#c0dcc0 guifg=black",
		"-c", "highlight DiffDelete ctermbg=240 ctermfg=240 guibg=#444444 guifg=#444444",
		"/a", "/b",
	}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_VimThreeFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "vim")
	want := []string{
		"vimdiff",
		"-c", "highlight DiffChange ctermbg=153 ctermfg=black guibg=#a6caf0 guifg=black",
		"-c", "highlight DiffText ctermbg=153 ctermfg=black cterm=bold guibg=#a6caf0 guifg=black gui=bold",
		"-c", "highlight DiffAdd ctermbg=151 ctermfg=black guibg=#c0dcc0 guifg=black",
		"-c", "highlight DiffDelete ctermbg=240 ctermfg=240 guibg=#444444 guifg=#444444",
		"/a", "/m", "/b",
	}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_VimOneFile_NoHighlightArgs(t *testing.T) {
	// A single file isn't a diff, so there's nothing to recolor — vim
	// should launch plainly, not carry the -c flags meant for vimdiff.
	e := &entry.Entry{Left: strptr("/a")}
	cmd := Command(e, "vim")
	want := []string{"vim", "/a"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestDiffHighlightArgs_ChangedMatchesStyleChangedHex(t *testing.T) {
	args := diffHighlightArgs()
	for _, group := range []string{"DiffChange", "DiffText"} {
		found := false
		for _, a := range args {
			if strings.Contains(a, "highlight "+group+" ") && strings.Contains(a, changedHex) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s should be colored with changedHex (%s), matching app.go's styleChanged", group, changedHex)
		}
	}
}

func TestDiffHighlightArgs_AddMatchesStyleUniqueHex(t *testing.T) {
	args := diffHighlightArgs()
	found := false
	for _, a := range args {
		if strings.Contains(a, "highlight DiffAdd ") && strings.Contains(a, uniqueHex) {
			found = true
		}
	}
	if !found {
		t.Errorf("DiffAdd should be colored with uniqueHex (%s), matching app.go's styleUnique", uniqueHex)
	}
}

func TestDiffHighlightArgs_DeleteIsNeutralNotAThirdColor(t *testing.T) {
	// umerge itself never highlights an absent side — it's left blank.
	// DiffDelete (the filler for lines only the other buffer has) should
	// stay a plain neutral, not introduce a color umerge's own directory
	// view doesn't use for this concept.
	args := diffHighlightArgs()
	for _, a := range args {
		if strings.Contains(a, "highlight DiffDelete ") {
			if strings.Contains(a, changedHex) || strings.Contains(a, uniqueHex) {
				t.Errorf("DiffDelete should not reuse the changed/unique colors: %q", a)
			}
			return
		}
	}
	t.Error("expected a DiffDelete highlight command")
}

func TestCommand_EmacsOneFile(t *testing.T) {
	// -nw keeps emacs in the invoking terminal instead of popping a new
	// GUI frame — otherwise plain `emacs` opens a separate window
	// whenever a display is available, breaking the "suspend the TUI,
	// run the tool inline, resume" model vim already gets for free.
	e := &entry.Entry{Left: strptr("/a")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "-nw", "/a"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

// twoWayFaceArgs/threeWayFaceArgs pin the exact --eval sequence
// ediffFaceArgs must produce for A/B and A/B/C, hardcoded literally
// (not by calling ediffFaceArgs itself) for the same reason
// TestCommand_VimTwoFiles/ThreeFiles hardcode vim's -c flags rather than
// calling diffHighlightArgs: catches a regression in the generator
// itself, not just in how Command wires it in.
func twoWayFaceArgs() []string {
	return []string{
		"--eval", "(require 'ediff-init)",
		"--eval", `(set-face-attribute 'ediff-odd-diff-A nil :background "#a6caf0" :foreground "black")`,
		"--eval", `(set-face-attribute 'ediff-even-diff-A nil :background "#a6caf0" :foreground "black")`,
		"--eval", `(set-face-attribute 'ediff-odd-diff-B nil :background "#a6caf0" :foreground "black")`,
		"--eval", `(set-face-attribute 'ediff-even-diff-B nil :background "#a6caf0" :foreground "black")`,
	}
}

func threeWayFaceArgs() []string {
	args := twoWayFaceArgs()
	return append(args,
		"--eval", `(set-face-attribute 'ediff-odd-diff-C nil :background "#a6caf0" :foreground "black")`,
		"--eval", `(set-face-attribute 'ediff-even-diff-C nil :background "#a6caf0" :foreground "black")`,
	)
}

func TestCommand_EmacsTwoFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := append([]string{"emacs", "-nw"}, twoWayFaceArgs()...)
	want = append(want, "--eval", `(ediff-files "/a" "/b")`)
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsThreeFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := append([]string{"emacs", "-nw"}, threeWayFaceArgs()...)
	want = append(want, "--eval", `(ediff3 "/a" "/m" "/b")`)
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

// The following cover recoloring ediff's built-in diff faces to match
// umerge's own directory-view palette, the same treatment vimdiff already
// got — so switching merge tools doesn't also switch color languages.
// Only odd/even-diff are themed; see ediffFaceArgs' doc comment for why
// current-diff/fine-diff are deliberately left alone (verified not to
// render distinctly in the -nw sessions umerge actually launches).

func TestEdiffFaceArgs_RequiresEdiffInitFirst(t *testing.T) {
	args := ediffFaceArgs([]string{"A", "B"})
	if len(args) < 2 || args[0] != "--eval" || args[1] != "(require 'ediff-init)" {
		t.Fatalf("expected (require 'ediff-init) as the first --eval, got %v", args[:2])
	}
}

func TestEdiffFaceArgs_OddEvenUseChangedHex(t *testing.T) {
	args := ediffFaceArgs([]string{"A", "B"})
	for _, face := range []string{"ediff-odd-diff-A", "ediff-even-diff-A"} {
		found := false
		for _, a := range args {
			if strings.Contains(a, face) && strings.Contains(a, changedHex) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s should use changedHex (%s), matching app.go's styleChanged", face, changedHex)
		}
	}
}

func TestEdiffFaceArgs_DoesNotTouchCurrentOrFineDiff(t *testing.T) {
	// Verified empirically that neither face renders distinctly in a real
	// -nw session (ediff-highlight-diff is documented "Invoked for X
	// displays only"; fine-diff word-level highlighting showed the same
	// behavior under direct testing) — setting them would be dead code.
	args := ediffFaceArgs([]string{"A", "B", "C"})
	for _, a := range args {
		if strings.Contains(a, "ediff-current-diff-") || strings.Contains(a, "ediff-fine-diff-") {
			t.Errorf("should not set current-diff or fine-diff faces (verified not to render in -nw), got %q", a)
		}
	}
}

func TestEdiffFaceArgs_TwoLettersProducesNoThirdLetterFaces(t *testing.T) {
	// 2-way (ediff-files) only ever uses buffers A/B; a stray "C" face
	// setting would be harmless in practice (ediff-init defines the face
	// regardless) but is a sign the letter list was built wrong.
	args := ediffFaceArgs([]string{"A", "B"})
	for _, a := range args {
		if strings.Contains(a, "-diff-C ") {
			t.Errorf("2-way face args should not touch any -C face, got %q", a)
		}
	}
}

// The following cover a bug found during pre-release review: a filename
// containing a literal `"` broke out of the elisp string literal built by
// emacsCommand, turning the rest of the (attacker-controlled) filename
// into arbitrary Lisp for --eval to run — e.g. a path ending in
// `foo".png") (shell-command "...` would execute a shell command the
// moment the entry was opened with --merge emacs. Not hypothetical given
// umerge's own stated use case of comparing untrusted/vendor-dropped
// trees, where a crafted filename is exactly the kind of thing that could
// appear.

// wantTail asserts that got's final len(want) elements equal want —
// these escaping tests only care about the launch eval at the very end
// of cmd.Args, not the face-theming --eval args now prepended ahead of
// it (covered separately above), so checking the whole slice would make
// them fail every time face-args changes for reasons unrelated to
// escaping.
func wantTail(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("Args = %v, too short to contain want tail %v", got, want)
	}
	tail := got[len(got)-len(want):]
	if !reflect.DeepEqual(tail, want) {
		t.Errorf("Args tail = %v, want %v", tail, want)
	}
}

func TestCommand_EmacsTwoFiles_QuoteInFilenameIsEscaped(t *testing.T) {
	evil := `foo".png") (shell-command "touch PWNED") (ediff-files "bar`
	e := &entry.Entry{Left: strptr(evil), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	wantTail(t, cmd.Args, []string{"--eval",
		`(ediff-files "foo\".png\") (shell-command \"touch PWNED\") (ediff-files \"bar" "/b")`})
}

func TestCommand_EmacsThreeFiles_QuoteInFilenameIsEscaped(t *testing.T) {
	evil := `x"y`
	e := &entry.Entry{Left: strptr(evil), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	wantTail(t, cmd.Args, []string{"--eval", `(ediff3 "x\"y" "/m" "/b")`})
}

func TestCommand_EmacsTwoFiles_BackslashInFilenameIsEscaped(t *testing.T) {
	e := &entry.Entry{Left: strptr(`a\b`), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	wantTail(t, cmd.Args, []string{"--eval", `(ediff-files "a\\b" "/b")`})
}

func TestElispQuote_EscapesBackslashBeforeQuote(t *testing.T) {
	// Order matters: escaping `"` first would double-escape the backslash
	// just inserted in front of it. E.g. `\"` (backslash, quote) must
	// become `\\\"`, not `\\"`.
	got := elispQuote(`\"`)
	want := `\\\"`
	if got != want {
		t.Errorf("elispQuote(%q) = %q, want %q", `\"`, got, want)
	}
}

func TestPresentPaths_SkipsAbsentSidesInOrder(t *testing.T) {
	e := &entry.Entry{Left: strptr("/left"), Right: strptr("/right")} // Middle absent
	got := presentPaths(e)
	want := []string{"/left", "/right"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("presentPaths = %v, want %v", got, want)
	}
}

func TestPresentPaths_MiddleOnly(t *testing.T) {
	e := &entry.Entry{Middle: strptr("/mid")}
	got := presentPaths(e)
	want := []string{"/mid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("presentPaths = %v, want %v", got, want)
	}
}
