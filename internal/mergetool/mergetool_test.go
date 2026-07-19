package mergetool

import (
	"reflect"
	"testing"

	"umerge/internal/entry"
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

func TestCommand_VimTwoFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}
	cmd := Command(e, "vim")
	want := []string{"vimdiff", "/a", "/b"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_VimThreeFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "vim")
	want := []string{"vimdiff", "/a", "/m", "/b"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsOneFile(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "/a"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsTwoFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "--eval", `(ediff-files "/a" "/b")`}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsThreeFiles(t *testing.T) {
	e := &entry.Entry{Left: strptr("/a"), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "--eval", `(ediff3 "/a" "/m" "/b")`}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
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

func TestCommand_EmacsTwoFiles_QuoteInFilenameIsEscaped(t *testing.T) {
	evil := `foo".png") (shell-command "touch PWNED") (ediff-files "bar`
	e := &entry.Entry{Left: strptr(evil), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "--eval",
		`(ediff-files "foo\".png\") (shell-command \"touch PWNED\") (ediff-files \"bar" "/b")`}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsThreeFiles_QuoteInFilenameIsEscaped(t *testing.T) {
	evil := `x"y`
	e := &entry.Entry{Left: strptr(evil), Middle: strptr("/m"), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "--eval", `(ediff3 "x\"y" "/m" "/b")`}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
}

func TestCommand_EmacsTwoFiles_BackslashInFilenameIsEscaped(t *testing.T) {
	e := &entry.Entry{Left: strptr(`a\b`), Right: strptr("/b")}
	cmd := Command(e, "emacs")
	want := []string{"emacs", "--eval", `(ediff-files "a\\b" "/b")`}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("Args = %v, want %v", cmd.Args, want)
	}
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
