package mergetool

import (
	"os/exec"

	"umerge/internal/entry"
)

// Command returns an exec.Cmd to view or diff the given entry, or nil
// if there is nothing to launch (e.g. a directory, or no paths present).
//
// Single file present  → vim  <path>
// Two files present    → vimdiff <left> <right>
// Three files present  → vimdiff <left> <middle> <right>
func Command(e *entry.Entry) *exec.Cmd {
	if e.IsDir {
		return nil
	}
	paths := presentPaths(e)
	switch len(paths) {
	case 0:
		return nil
	case 1:
		return exec.Command("vim", paths[0])
	default:
		return exec.Command("vimdiff", paths...)
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
