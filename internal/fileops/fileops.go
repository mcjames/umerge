package fileops

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Copy copies src onto dest, recursively. If dest already exists as a
// directory, it is removed first so the result exactly mirrors src rather
// than nesting src inside the existing directory. If dest exists as a
// file, it is overwritten directly. Any missing intermediate directories
// in dest's path are created first — without this, copying a file whose
// destination side is missing multiple levels deep (e.g. the immediate
// parent directory was never enumerated on that side at all) fails with
// "cp: cannot create ... No such file or directory", since plain `cp -R`
// requires the destination's parent to already exist. Mirrors the Python
// version's FileOpsPOSIX primitive ("cp -R") plus the pre-deleting of a
// directory target, but goes beyond it on the missing-parent case, which
// Python's `cp -R` would fail on identically.
func Copy(src, dest string) error {
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return exec.Command("cp", "-R", src, dest).Run()
}

// Delete removes path and everything under it. It is not an error if
// path does not exist.
func Delete(path string) error {
	return os.RemoveAll(path)
}

// fileCompareChunkSize is the buffer size used by filesEqual's chunked
// content comparison — chosen so comparing even large files doesn't load
// either one fully into memory.
const fileCompareChunkSize = 64 * 1024

// filesEqual reports whether a and b have byte-identical content. It
// checks sizes first — a mismatch proves inequality without reading
// either file — then compares content in fixed-size chunks rather than
// loading whole files into memory. This is the short-circuit that lets
// CompareTwoFiles/CompareThreeFiles skip invoking diff/diff3 entirely for
// the common case of unchanged files.
func filesEqual(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()
	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()

	bufA := make([]byte, fileCompareChunkSize)
	bufB := make([]byte, fileCompareChunkSize)
	for {
		na, errA := io.ReadFull(fa, bufA)
		if errA != nil && errA != io.EOF && errA != io.ErrUnexpectedEOF {
			return false, errA
		}
		nb, errB := io.ReadFull(fb, bufB)
		if errB != nil && errB != io.EOF && errB != io.ErrUnexpectedEOF {
			return false, errB
		}
		if !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false, nil
		}
		// Sizes already matched above, and both sides read the same
		// chunk size each iteration, so reaching EOF/ErrUnexpectedEOF on
		// one side implies the other did too at the same point.
		if errA == io.EOF || errA == io.ErrUnexpectedEOF {
			return true, nil
		}
	}
}

// binarySniffBytes is how many leading bytes to check for a NUL byte when
// deciding whether a file is binary — the same heuristic git uses for the
// same decision, and consistent with what GNU diff/diff3 evidently do
// internally (verified empirically: they report "Binary files ... differ"
// for files containing this kind of content rather than attempting a
// line-based diff).
const binarySniffBytes = 8000

// isBinaryFile reports whether path's leading bytes contain a NUL byte.
func isBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, binarySniffBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	return bytes.IndexByte(buf[:n], 0) != -1, nil
}

// anyBinary reports whether any of paths is binary, stopping at the first
// one found (or the first error).
func anyBinary(paths ...string) (bool, error) {
	for _, p := range paths {
		bin, err := isBinaryFile(p)
		if err != nil {
			return false, err
		}
		if bin {
			return true, nil
		}
	}
	return false, nil
}

// CompareTwoFiles compares left and right. If they're byte-identical, it
// returns (0, false, nil) without invoking diff at all — the common case
// for a real directory comparison, where most files are unchanged. If
// they differ and either is binary, hunk-counting doesn't apply, so it
// returns (0, true, nil) — diff is not invoked in that case either. Only
// text files that actually differ result in diff being run, for an
// accurate hunk count (lines in diff output that start with a digit, e.g.
// "1,3c1,5").
func CompareTwoFiles(left, right string) (numDiffs int, binary bool, err error) {
	equal, err := filesEqual(left, right)
	if err != nil {
		return 0, false, err
	}
	if equal {
		return 0, false, nil
	}

	bin, err := anyBinary(left, right)
	if err != nil {
		return 0, false, err
	}
	if bin {
		return 0, true, nil
	}

	out, execErr := exec.Command("diff", left, right).Output()
	if execErr != nil {
		if exit, ok := execErr.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			// exit 1 means files differ — not an error
			execErr = nil
		} else {
			return 0, false, execErr
		}
	}
	count := 0
	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
			count++
		}
	}
	return count, false, nil
}

// CompareThreeFiles compares left, middle, and right, returning the
// number of change hunks between left↔middle (lmDiffs) and middle↔right
// (mrDiffs).
//
// If all three are byte-identical, returns (0, 0, false, nil) without
// invoking diff3. If they differ and any of the three is binary, returns
// (0, 0, true, nil) instead of invoking diff3 — verified empirically that
// diff3 can't meaningfully hunk-count across the triple in that case
// anyway; it fails outright ("diff3: diff failed: Binary files ...
// differ", exit 2) the moment any pairwise comparison hits binary
// content, so there's no partial result to preserve by calling it.
//
// diff3 markers, when it does run:
//
//	====1  left differs  → lm++
//	====2  middle differs → lm++, mr++
//	====3  right differs  → mr++
//	====   all differ    → lm++, mr++
func CompareThreeFiles(left, middle, right string) (lmDiffs, mrDiffs int, binary bool, err error) {
	lmEqual, err := filesEqual(left, middle)
	if err != nil {
		return 0, 0, false, err
	}
	mrEqual, err := filesEqual(middle, right)
	if err != nil {
		return 0, 0, false, err
	}
	if lmEqual && mrEqual {
		return 0, 0, false, nil
	}

	bin, err := anyBinary(left, middle, right)
	if err != nil {
		return 0, 0, false, err
	}
	if bin {
		return 0, 0, true, nil
	}

	out, execErr := exec.Command("diff3", left, middle, right).Output()
	if execErr != nil {
		if exit, ok := execErr.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			// exit 1 means conflicts exist — not an error
		} else {
			return 0, 0, false, execErr
		}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "====") {
			continue
		}
		switch strings.TrimSpace(strings.TrimPrefix(line, "====")) {
		case "1":
			lmDiffs++
		case "2":
			lmDiffs++
			mrDiffs++
		case "3":
			mrDiffs++
		default: // "====" — all three differ
			lmDiffs++
			mrDiffs++
		}
	}
	return lmDiffs, mrDiffs, false, nil
}
