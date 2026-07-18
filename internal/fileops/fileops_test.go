package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCompareTwoFiles_Identical(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "line1\nline2\n")
	right := writeFile(t, dir, "right.txt", "line1\nline2\n")

	n, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 0 {
		t.Errorf("got %d diffs, want 0", n)
	}
}

func TestCompareTwoFiles_SingleHunk(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "line1\nline2\nline3\n")
	right := writeFile(t, dir, "right.txt", "line1\nCHANGED\nline3\n")

	n, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("got %d diffs, want 1", n)
	}
}

func TestCompareTwoFiles_MultipleHunks(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "a\nb\nc\nd\ne\nf\n")
	right := writeFile(t, dir, "right.txt", "CHANGED\nb\nc\nd\ne\nCHANGED\n")

	n, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 2 {
		t.Errorf("got %d diffs, want 2", n)
	}
}

func TestCompareTwoFiles_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "content\n")

	_, err := CompareTwoFiles(left, filepath.Join(dir, "does-not-exist.txt"))
	if err == nil {
		t.Fatal("expected an error comparing against a nonexistent file, got nil")
	}
}

func TestCompareThreeFiles_AllIdentical(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm != 0 || mr != 0 {
		t.Errorf("got lm=%d mr=%d, want 0,0", lm, mr)
	}
}

func TestCompareThreeFiles_LeftDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "CHANGED\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 {
		t.Errorf("got lm=%d, want > 0", lm)
	}
	if mr != 0 {
		t.Errorf("got mr=%d, want 0", mr)
	}
}

func TestCompareThreeFiles_RightDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "CHANGED\n")

	lm, mr, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm != 0 {
		t.Errorf("got lm=%d, want 0", lm)
	}
	if mr == 0 {
		t.Errorf("got mr=%d, want > 0", mr)
	}
}

func TestCompareThreeFiles_MiddleDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "CHANGED\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 || mr == 0 {
		t.Errorf("got lm=%d mr=%d, want both > 0", lm, mr)
	}
}

func TestCompareThreeFiles_AllDiffer(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "AAA\n")
	middle := writeFile(t, dir, "middle.txt", "BBB\n")
	right := writeFile(t, dir, "right.txt", "CCC\n")

	lm, mr, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 || mr == 0 {
		t.Errorf("got lm=%d mr=%d, want both > 0", lm, mr)
	}
}

func TestCompareThreeFiles_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "content\n")
	middle := writeFile(t, dir, "middle.txt", "content\n")

	_, _, err := CompareThreeFiles(left, middle, filepath.Join(dir, "does-not-exist.txt"))
	if err == nil {
		t.Fatal("expected an error comparing against a nonexistent file, got nil")
	}
}
