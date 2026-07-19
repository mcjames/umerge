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

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 0 {
		t.Errorf("got %d diffs, want 0", n)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareTwoFiles_SingleHunk(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "line1\nline2\nline3\n")
	right := writeFile(t, dir, "right.txt", "line1\nCHANGED\nline3\n")

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("got %d diffs, want 1", n)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareTwoFiles_MultipleHunks(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "a\nb\nc\nd\ne\nf\n")
	right := writeFile(t, dir, "right.txt", "CHANGED\nb\nc\nd\ne\nCHANGED\n")

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n != 2 {
		t.Errorf("got %d diffs, want 2", n)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareTwoFiles_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "content\n")

	_, _, err := CompareTwoFiles(left, filepath.Join(dir, "does-not-exist.txt"))
	if err == nil {
		t.Fatal("expected an error comparing against a nonexistent file, got nil")
	}
}

func TestCompareThreeFiles_AllIdentical(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm != 0 || mr != 0 {
		t.Errorf("got lm=%d mr=%d, want 0,0", lm, mr)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareThreeFiles_LeftDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "CHANGED\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 {
		t.Errorf("got lm=%d, want > 0", lm)
	}
	if mr != 0 {
		t.Errorf("got mr=%d, want 0", mr)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareThreeFiles_RightDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "CHANGED\n")

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm != 0 {
		t.Errorf("got lm=%d, want 0", lm)
	}
	if mr == 0 {
		t.Errorf("got mr=%d, want > 0", mr)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareThreeFiles_MiddleDiffersOnly(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "CHANGED\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 || mr == 0 {
		t.Errorf("got lm=%d mr=%d, want both > 0", lm, mr)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareThreeFiles_AllDiffer(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "AAA\n")
	middle := writeFile(t, dir, "middle.txt", "BBB\n")
	right := writeFile(t, dir, "right.txt", "CCC\n")

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v", err)
	}
	if lm == 0 || mr == 0 {
		t.Errorf("got lm=%d mr=%d, want both > 0", lm, mr)
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareThreeFiles_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "content\n")
	middle := writeFile(t, dir, "middle.txt", "content\n")

	_, _, _, err := CompareThreeFiles(left, middle, filepath.Join(dir, "does-not-exist.txt"))
	if err == nil {
		t.Fatal("expected an error comparing against a nonexistent file, got nil")
	}
}

func TestCopy_FileToNewDestination(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src.txt", "hello\n")
	dest := filepath.Join(dir, "dest.txt")

	if err := Copy(src, dest); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("dest content = %q, want %q", got, "hello\n")
	}
}

func TestCopy_FileOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src.txt", "new content\n")
	dest := writeFile(t, dir, "dest.txt", "old content\n")

	if err := Copy(src, dest); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "new content\n" {
		t.Errorf("dest content = %q, want %q", got, "new content\n")
	}
}

func TestCopy_DirectoryReplacesExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "src/sub/keep.txt", "keep\n")

	// dest already exists with different, unrelated contents.
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "dest/stale.txt", "should be gone after copy\n")

	if err := Copy(src, dest); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should have been removed by the directory replace, err=%v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "sub", "keep.txt"))
	if err != nil {
		t.Fatalf("reading copied nested file: %v", err)
	}
	if string(got) != "keep\n" {
		t.Errorf("nested content = %q, want %q", got, "keep\n")
	}
}

func TestCopy_CreatesMissingIntermediateDestinationDirectories(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src.txt", "content\n")
	// dest's parent ("missing/levels") doesn't exist at all yet — this is
	// the exact shape of the reported bug: a file present several levels
	// deep on one side where none of those intermediate directories were
	// ever created on the destination side.
	dest := filepath.Join(dir, "missing", "levels", "dest.txt")

	if err := Copy(src, dest); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != "content\n" {
		t.Errorf("dest content = %q, want %q", got, "content\n")
	}
}

func TestCopy_NonexistentSourceErrors(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "does-not-exist")
	dest := filepath.Join(dir, "dest.txt")

	if err := Copy(src, dest); err == nil {
		t.Fatal("expected an error copying a nonexistent source, got nil")
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "gone.txt", "bye\n")

	if err := Delete(p); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected file to be gone, err=%v", err)
	}
}

func TestDelete_RemovesDirectoryRecursively(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "target/sub/file.txt", "content\n")

	if err := Delete(target); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("expected directory to be gone, err=%v", err)
	}
}

func TestDelete_NonexistentPathIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "never-existed")

	if err := Delete(p); err != nil {
		t.Errorf("Delete of a nonexistent path should not error, got: %v", err)
	}
}

// The following cover the short-circuit comparison and binary-file
// detection added 2026-07-19: identical files should never invoke
// diff/diff3 at all, and files confirmed different but binary should
// never invoke them either (diff3 can't meaningfully hunk-count across a
// triple involving binary content anyway — verified empirically that it
// fails outright in that case).

func writeBinaryFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// withoutDiffToolsOnPath points PATH at an empty directory for the
// duration of the test, so diff/diff3 can't be found at all. This proves
// the short-circuit/binary-detection paths actually avoid invoking them,
// rather than just happening to return the right answer for some other
// reason — if the short-circuit weren't real, these tests would fail with
// "executable file not found in $PATH".
func withoutDiffToolsOnPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func TestCompareTwoFiles_IdenticalNeverInvokesDiff(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "line1\nline2\n")
	right := writeFile(t, dir, "right.txt", "line1\nline2\n")

	withoutDiffToolsOnPath(t)

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v (short-circuit should avoid needing diff at all)", err)
	}
	if n != 0 || binary {
		t.Errorf("got n=%d binary=%v, want 0, false", n, binary)
	}
}

func TestCompareThreeFiles_AllIdenticalNeverInvokesDiff3(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")

	withoutDiffToolsOnPath(t)

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v (short-circuit should avoid needing diff3 at all)", err)
	}
	if lm != 0 || mr != 0 || binary {
		t.Errorf("got lm=%d mr=%d binary=%v, want 0, 0, false", lm, mr, binary)
	}
}

func TestCompareTwoFiles_DifferentSizesStillDetectsDifference(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "short\n")
	right := writeFile(t, dir, "right.txt", "a much longer line that is different\n")

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if n == 0 {
		t.Error("expected a nonzero diff count for files with different sizes/content")
	}
	if binary {
		t.Error("binary = true, want false")
	}
}

func TestCompareTwoFiles_DifferentBinaryFilesNeverInvokesDiff(t *testing.T) {
	dir := t.TempDir()
	left := writeBinaryFile(t, dir, "left.bin", []byte{0x00, 0x01, 0x02, 0x03})
	right := writeBinaryFile(t, dir, "right.bin", []byte{0x00, 0xFF, 0xFE, 0xFD})

	withoutDiffToolsOnPath(t)

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v (binary detection should avoid needing diff at all)", err)
	}
	if !binary {
		t.Error("binary = false, want true")
	}
	if n != 0 {
		t.Errorf("numDiffs = %d, want 0 (not meaningful for binary)", n)
	}
}

func TestCompareTwoFiles_IdenticalBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	left := writeBinaryFile(t, dir, "left.bin", content)
	right := writeBinaryFile(t, dir, "right.bin", content)

	n, binary, err := CompareTwoFiles(left, right)
	if err != nil {
		t.Fatalf("CompareTwoFiles: %v", err)
	}
	if binary {
		t.Error("binary = true, want false — identical files are Same regardless of content type")
	}
	if n != 0 {
		t.Errorf("numDiffs = %d, want 0", n)
	}
}

func TestCompareThreeFiles_AnyBinaryNeverInvokesDiff3(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeBinaryFile(t, dir, "right.bin", []byte{0x00, 0x01, 0x02})

	withoutDiffToolsOnPath(t)

	lm, mr, binary, err := CompareThreeFiles(left, middle, right)
	if err != nil {
		t.Fatalf("CompareThreeFiles: %v (binary detection should avoid needing diff3 at all)", err)
	}
	if !binary {
		t.Error("binary = false, want true")
	}
	if lm != 0 || mr != 0 {
		t.Errorf("got lm=%d mr=%d, want 0, 0 (not meaningful for binary)", lm, mr)
	}
}

func TestIsBinaryFile(t *testing.T) {
	dir := t.TempDir()
	textPath := writeFile(t, dir, "text.txt", "just plain text\n")
	binPath := writeBinaryFile(t, dir, "bin.dat", []byte{'h', 'i', 0x00, 'x'})

	if bin, err := isBinaryFile(textPath); err != nil || bin {
		t.Errorf("isBinaryFile(text) = %v, %v; want false, nil", bin, err)
	}
	if bin, err := isBinaryFile(binPath); err != nil || !bin {
		t.Errorf("isBinaryFile(binary) = %v, %v; want true, nil", bin, err)
	}
}

func TestFilesEqual(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "hello world\n")
	b := writeFile(t, dir, "b.txt", "hello world\n")
	c := writeFile(t, dir, "c.txt", "hello WORLD\n") // same length, different content
	d := writeFile(t, dir, "d.txt", "hello\n")       // different length

	if eq, err := filesEqual(a, b); err != nil || !eq {
		t.Errorf("filesEqual(a, b) = %v, %v; want true, nil", eq, err)
	}
	if eq, err := filesEqual(a, c); err != nil || eq {
		t.Errorf("filesEqual(a, c) = %v, %v; want false, nil (same size, different content)", eq, err)
	}
	if eq, err := filesEqual(a, d); err != nil || eq {
		t.Errorf("filesEqual(a, d) = %v, %v; want false, nil (different size)", eq, err)
	}
}

func TestFilesEqual_LargerThanChunkSize(t *testing.T) {
	// Exercises the chunked-read loop across multiple iterations, with a
	// difference near the very end — after several fully-matching chunks —
	// rather than in the first chunk.
	dir := t.TempDir()
	size := fileCompareChunkSize*2 + 100
	contentA := make([]byte, size)
	for i := range contentA {
		contentA[i] = byte(i % 251)
	}
	contentB := append([]byte(nil), contentA...)
	contentB[len(contentB)-1] ^= 0xFF // flip the very last byte

	a := writeBinaryFile(t, dir, "a.bin", contentA)
	b := writeBinaryFile(t, dir, "b.bin", contentA)
	c := writeBinaryFile(t, dir, "c.bin", contentB)

	if eq, err := filesEqual(a, b); err != nil || !eq {
		t.Errorf("filesEqual(a, b) = %v, %v; want true, nil", eq, err)
	}
	if eq, err := filesEqual(a, c); err != nil || eq {
		t.Errorf("filesEqual(a, c) = %v, %v; want false, nil (differ in the last byte, past several full chunks)", eq, err)
	}
}
