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
