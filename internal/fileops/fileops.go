package fileops

import (
	"bytes"
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

// CompareTwoFiles runs diff on two files and returns the number of change
// hunks (lines in diff output that start with a digit, e.g. "1,3c1,5").
// Returns 0 and no error when files are identical.
func CompareTwoFiles(left, right string) (int, error) {
	out, err := exec.Command("diff", left, right).Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			// exit 1 means files differ — not an error
			err = nil
		} else {
			return 0, err
		}
	}
	count := 0
	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
			count++
		}
	}
	return count, nil
}

// CompareThreeFiles runs diff3 on three files and returns the number of
// change hunks between left↔middle (lmDiffs) and middle↔right (mrDiffs).
//
// diff3 markers:
//
//	====1  left differs  → lm++
//	====2  middle differs → lm++, mr++
//	====3  right differs  → mr++
//	====   all differ    → lm++, mr++
func CompareThreeFiles(left, middle, right string) (lmDiffs, mrDiffs int, err error) {
	out, execErr := exec.Command("diff3", left, middle, right).Output()
	if execErr != nil {
		if exit, ok := execErr.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			// exit 1 means conflicts exist — not an error
		} else {
			return 0, 0, execErr
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
	return lmDiffs, mrDiffs, nil
}
