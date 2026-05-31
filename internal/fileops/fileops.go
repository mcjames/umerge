package fileops

import (
	"bytes"
	"os/exec"
	"strings"
)

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
