package entry

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one node in the merged directory tree.
// Left and Right are nil when the file is absent on that side.
type Entry struct {
	Name      string  // display name (from whichever side has it; prefer left)
	Left      *string // absolute path on the left side; nil if absent
	Right     *string // absolute path on the right side; nil if absent
	IsDir     bool    // true if either side is a directory
	Depth     int
	Collapsed bool
	Children  []*Entry
}

// BuildPair constructs a merged tree for a two-way comparison.
// Entries are matched case-insensitively, preserving original case in paths.
func BuildPair(leftRoot, rightRoot string) ([]*Entry, error) {
	return buildTree(&leftRoot, &rightRoot, 0)
}

// buildTree merges the contents of two directories into a unified tree.
// Either root may be nil when one side lacks a subtree entirely.
func buildTree(leftRoot, rightRoot *string, depth int) ([]*Entry, error) {
	var leftFiles, rightFiles []os.DirEntry

	if leftRoot != nil {
		if des, err := os.ReadDir(*leftRoot); err == nil {
			leftFiles = des
			sort.Slice(leftFiles, func(i, j int) bool {
				return strings.ToLower(leftFiles[i].Name()) < strings.ToLower(leftFiles[j].Name())
			})
		}
	}
	if rightRoot != nil {
		if des, err := os.ReadDir(*rightRoot); err == nil {
			rightFiles = des
			sort.Slice(rightFiles, func(i, j int) bool {
				return strings.ToLower(rightFiles[i].Name()) < strings.ToLower(rightFiles[j].Name())
			})
		}
	}

	var entries []*Entry
	li, ri := 0, 0

	for li < len(leftFiles) || ri < len(rightFiles) {
		hasL := li < len(leftFiles)
		hasR := ri < len(rightFiles)

		// Decide which side(s) to consume this iteration.
		var takeLeft, takeRight bool
		if hasL && hasR {
			lname := strings.ToLower(leftFiles[li].Name())
			rname := strings.ToLower(rightFiles[ri].Name())
			switch {
			case lname == rname:
				takeLeft, takeRight = true, true
			case lname < rname:
				takeLeft = true
			default:
				takeRight = true
			}
		} else if hasL {
			takeLeft = true
		} else {
			takeRight = true
		}

		var lde, rde os.DirEntry
		if takeLeft {
			lde = leftFiles[li]
			li++
		}
		if takeRight {
			rde = rightFiles[ri]
			ri++
		}

		// Build the Entry for this matched (or unmatched) pair.
		var leftPath, rightPath *string
		name := ""
		isDir := false

		if lde != nil {
			p := filepath.Join(*leftRoot, lde.Name())
			leftPath = &p
			name = lde.Name()
			if lde.IsDir() {
				isDir = true
			}
		}
		if rde != nil {
			p := filepath.Join(*rightRoot, rde.Name())
			rightPath = &p
			if name == "" {
				name = rde.Name()
			}
			if rde.IsDir() {
				isDir = true
			}
		}

		e := &Entry{
			Name:  name,
			Left:  leftPath,
			Right: rightPath,
			IsDir: isDir,
			Depth: depth,
		}

		// Recurse into whichever sides are directories.
		if isDir {
			var lChild, rChild *string
			if lde != nil && lde.IsDir() {
				lChild = leftPath
			}
			if rde != nil && rde.IsDir() {
				rChild = rightPath
			}
			e.Children, _ = buildTree(lChild, rChild, depth+1)
		}

		entries = append(entries, e)
	}

	return entries, nil
}

// Flatten returns the visible entries in depth-first order, skipping
// children of collapsed directories.
func Flatten(entries []*Entry) []*Entry {
	var out []*Entry
	for _, e := range entries {
		out = append(out, e)
		if e.IsDir && !e.Collapsed {
			out = append(out, Flatten(e.Children)...)
		}
	}
	return out
}
