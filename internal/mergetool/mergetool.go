package mergetool

import (
	"fmt"
	"os/exec"

	"umerge/internal/entry"
)

// Command returns an exec.Cmd to view or diff the given entry using the
// requested tool, or nil if there is nothing to launch (e.g. a directory
// or no paths present).
//
// tool must be "vim" or "emacs".
//
// vim behaviour:
//
//	one file   → vim  <file>
//	two files  → vimdiff <left> <right>
//	three files → vimdiff <left> <middle> <right>
//
// emacs behaviour:
//
//	one file   → emacs <file>
//	two files  → emacs --eval (ediff-files "left" "right")
//	three files → emacs --eval (ediff3 "left" "middle" "right")
func Command(e *entry.Entry, tool string) *exec.Cmd {
	if e.IsDir {
		return nil
	}
	paths := presentPaths(e)
	if len(paths) == 0 {
		return nil
	}
	if tool == "emacs" {
		return emacsCommand(paths)
	}
	return vimCommand(paths)
}

func vimCommand(paths []string) *exec.Cmd {
	switch len(paths) {
	case 1:
		return exec.Command("vim", paths[0])
	default:
		return exec.Command("vimdiff", paths...)
	}
}

func emacsCommand(paths []string) *exec.Cmd {
	switch len(paths) {
	case 1:
		return exec.Command("emacs", paths[0])
	case 2:
		return exec.Command("emacs", "--eval",
			fmt.Sprintf(`(ediff-files "%s" "%s")`, paths[0], paths[1]))
	default:
		return exec.Command("emacs", "--eval",
			fmt.Sprintf(`(ediff3 "%s" "%s" "%s")`, paths[0], paths[1], paths[2]))
	}
}

// presentPaths returns the non-nil paths of e in left→middle→right order.
func presentPaths(e *entry.Entry) []string {
	var paths []string
	for _, p := range []*string{e.Left, e.Middle, e.Right} {
		if p != nil {
			paths = append(paths, *p)
		}
	}
	return paths
}
