package entry

import (
	"os"
	"path/filepath"
	"testing"
)

func writeGitignore(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIgnore_NilMatchesNothing(t *testing.T) {
	var ig *Ignore
	if ig.Match("anything", false) {
		t.Error("nil *Ignore should never match")
	}
	if ig.Match("anything", true) {
		t.Error("nil *Ignore should never match, even for a directory")
	}
}

func TestIgnore_DotGitAlwaysIgnoredWithNoGitignoreFile(t *testing.T) {
	root := t.TempDir() // no .gitignore written at all
	ig := LoadIgnore(&root)

	if !ig.Match(".git", true) {
		t.Error(".git should always be ignored, even with no .gitignore present")
	}
	if ig.Match("src", true) {
		t.Error("unrelated directory should not be ignored")
	}
}

func TestIgnore_SimpleWildcardPattern(t *testing.T) {
	root := t.TempDir()
	writeGitignore(t, root, "*.log\n")
	ig := LoadIgnore(&root)

	if !ig.Match("debug.log", false) {
		t.Error("debug.log should match *.log")
	}
	if ig.Match("debug.txt", false) {
		t.Error("debug.txt should not match *.log")
	}
}

// TestIgnore_DirectoryOnlyPatternRequiresIsDirTrue is the trickiest bit of
// integrating this library: a trailing-slash pattern like "build/" compiles
// to a regexp that only matches a candidate string that itself contains a
// trailing slash. Passing isDir=false for a directory would silently fail
// to filter it out.
func TestIgnore_DirectoryOnlyPatternRequiresIsDirTrue(t *testing.T) {
	root := t.TempDir()
	writeGitignore(t, root, "build/\n")
	ig := LoadIgnore(&root)

	if !ig.Match("build", true) {
		t.Error("build/ pattern should match the directory when isDir=true")
	}
	if ig.Match("build", false) {
		t.Error("build/ pattern should never match a same-named file (isDir=false)")
	}
}

func TestIgnore_RootAnchoredPatternDoesNotMatchNestedSameName(t *testing.T) {
	root := t.TempDir()
	writeGitignore(t, root, "/dist\n") // anchored: only the top-level "dist"
	ig := LoadIgnore(&root)

	if !ig.Match("dist", true) {
		t.Error("top-level dist should match /dist")
	}
	if ig.Match("sub/dist", true) {
		t.Error("nested sub/dist should NOT match the root-anchored /dist pattern")
	}
}

func TestIgnore_CombinesPatternsFromMultipleRoots(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeGitignore(t, left, "*.left-only\n")
	writeGitignore(t, right, "*.right-only\n")
	ig := LoadIgnore(&left, &right)

	if !ig.Match("x.left-only", false) {
		t.Error("pattern from left's .gitignore should apply")
	}
	if !ig.Match("x.right-only", false) {
		t.Error("pattern from right's .gitignore should apply")
	}
}

func TestIgnore_NegationReincludesFile(t *testing.T) {
	root := t.TempDir()
	writeGitignore(t, root, "*.log\n!important.log\n")
	ig := LoadIgnore(&root)

	if !ig.Match("debug.log", false) {
		t.Error("debug.log should still be ignored")
	}
	if ig.Match("important.log", false) {
		t.Error("important.log should be re-included by the negated pattern")
	}
}

// TestBuildPair_RespectsGitignore is the end-to-end check: a file/directory
// matched by .gitignore should be absent from the built tree entirely
// (including never recursing into an ignored directory's children), while
// .git is filtered even with no .gitignore file at all.
func TestBuildPair_RespectsGitignore(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeGitignore(t, left, "*.log\nbuild/\n")
	mkfile(t, left, "keep.txt")
	mkfile(t, right, "keep.txt")
	mkfile(t, left, "debug.log")
	mkfile(t, left, "build/output.bin")
	mkdirOnly(t, left, ".git")
	mkfile(t, left, ".git/HEAD")

	ig := LoadIgnore(&left, &right)
	entries, err := BuildPair(left, right, ig)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}

	if findByName(entries, "keep.txt") == nil {
		t.Error("keep.txt should be present")
	}
	if e := findByName(entries, "debug.log"); e != nil {
		t.Errorf("debug.log should be filtered out by *.log, got %+v", e)
	}
	if e := findByName(entries, "build"); e != nil {
		t.Errorf("build/ should be filtered out entirely (including its children), got %+v", e)
	}
	if e := findByName(entries, ".git"); e != nil {
		t.Errorf(".git should always be filtered out, got %+v", e)
	}
}

// TestBuildPair_NilIgnoreFiltersNothing confirms passing nil (the
// --no-gitignore case) preserves the pre-filtering behavior exactly,
// including showing .git itself.
func TestBuildPair_NilIgnoreFiltersNothing(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	writeGitignore(t, left, "*.log\n")
	mkfile(t, left, "debug.log")
	mkdirOnly(t, left, ".git")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}

	if findByName(entries, "debug.log") == nil {
		t.Error("debug.log should be present when ignore is nil")
	}
	if findByName(entries, ".git") == nil {
		t.Error(".git should be present when ignore is nil")
	}
}

func TestBuildPair_RelPathIsStampedOnEntries(t *testing.T) {
	left, right := t.TempDir(), t.TempDir()
	mkfile(t, left, "sub/nested.txt")
	mkfile(t, right, "sub/nested.txt")

	entries, err := BuildPair(left, right, nil)
	if err != nil {
		t.Fatalf("BuildPair: %v", err)
	}

	sub := findByName(entries, "sub")
	if sub == nil {
		t.Fatal("expected a sub entry")
	}
	if sub.RelPath != "sub" {
		t.Errorf("sub.RelPath = %q, want %q", sub.RelPath, "sub")
	}
	nested := findByName(sub.Children, "nested.txt")
	if nested == nil {
		t.Fatal("expected a nested.txt entry under sub")
	}
	if nested.RelPath != "sub/nested.txt" {
		t.Errorf("nested.RelPath = %q, want %q", nested.RelPath, "sub/nested.txt")
	}
}
