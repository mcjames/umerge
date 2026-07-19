package entry

import (
	"os"
	"path/filepath"
	"sort"
)

// CompareState is the result of comparing the file content across sides.
type CompareState int

const (
	Uncompared CompareState = iota
	Same
	Different
	CompareError
	// BinaryDifferent means the files differ and at least one side is
	// binary content — hunk-counting doesn't apply, since diff/diff3 are
	// never invoked for this case (see fileops.CompareTwoFiles).
	BinaryDifferent
)

// Entry is one node in the merged directory tree.
// Left, Middle, Right are nil when the file is absent on that side.
// Middle is always nil in two-way mode.
type Entry struct {
	Name      string  // display name (prefer left, then middle, then right)
	Left      *string // absolute path on the left side; nil if absent
	Middle    *string // absolute path on the middle side; nil if absent or two-way
	Right     *string // absolute path on the right side; nil if absent
	IsDir     bool    // true if any side is a directory
	Depth     int
	Collapsed bool
	Children  []*Entry

	// Comparison results (files only; set asynchronously after build).
	Compare  CompareState
	NumDiffs int // 2-way: hunks between left and right
	LMDiffs  int // 3-way: hunks between left and middle
	MRDiffs  int // 3-way: hunks between middle and right
}

// BuildPair constructs a merged tree for a two-way comparison.
func BuildPair(leftRoot, rightRoot string) ([]*Entry, error) {
	return BuildTree(&leftRoot, nil, &rightRoot, 0)
}

// BuildTriple constructs a merged tree for a three-way comparison.
// middleRoot is the parent/ancestor directory; left and right are the children.
func BuildTriple(leftRoot, middleRoot, rightRoot string) ([]*Entry, error) {
	return BuildTree(&leftRoot, &middleRoot, &rightRoot, 0)
}

// BuildTree merges the contents of two or three directories into a unified tree.
// Any root may be nil when that side lacks a subtree entirely. Exported so
// callers that need to rebuild a subtree at a nonzero depth (e.g. after a
// directory copy) can reuse the same merge logic as BuildPair/BuildTriple
// instead of duplicating it.
func BuildTree(leftRoot, middleRoot, rightRoot *string, depth int) ([]*Entry, error) {
	lf := readSorted(leftRoot)
	mf := readSorted(middleRoot)
	rf := readSorted(rightRoot)

	var entries []*Entry
	li, mi, ri := 0, 0, 0

	for li < len(lf) || mi < len(mf) || ri < len(rf) {
		// Find the lexicographically lowest name across all active lists.
		lowest := lowestName(lf, mf, rf, li, mi, ri)

		// Consume every list that matches the lowest name.
		var lde, mde, rde os.DirEntry
		if li < len(lf) && lf[li].Name() == lowest {
			lde = lf[li]
			li++
		}
		if mi < len(mf) && mf[mi].Name() == lowest {
			mde = mf[mi]
			mi++
		}
		if ri < len(rf) && rf[ri].Name() == lowest {
			rde = rf[ri]
			ri++
		}

		var leftPath, middlePath, rightPath *string
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
		if mde != nil {
			p := filepath.Join(*middleRoot, mde.Name())
			middlePath = &p
			if name == "" {
				name = mde.Name()
			}
			if mde.IsDir() {
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
			Name:   name,
			Left:   leftPath,
			Middle: middlePath,
			Right:  rightPath,
			IsDir:  isDir,
			Depth:  depth,
		}

		if isDir {
			var lc, mc, rc *string
			if lde != nil && lde.IsDir() {
				lc = leftPath
			}
			if mde != nil && mde.IsDir() {
				mc = middlePath
			}
			if rde != nil && rde.IsDir() {
				rc = rightPath
			}
			e.Children, _ = BuildTree(lc, mc, rc, depth+1)
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

// ── helpers ───────────────────────────────────────────────────────────────────

func readSorted(root *string) []os.DirEntry {
	if root == nil {
		return nil
	}
	des, err := os.ReadDir(*root)
	if err != nil {
		return nil
	}
	sort.Slice(des, func(i, j int) bool {
		return des[i].Name() < des[j].Name()
	})
	return des
}

func lowestName(lf, mf, rf []os.DirEntry, li, mi, ri int) string {
	lowest := ""
	if li < len(lf) {
		lowest = lf[li].Name()
	}
	if mi < len(mf) {
		if n := mf[mi].Name(); lowest == "" || n < lowest {
			lowest = n
		}
	}
	if ri < len(rf) {
		if n := rf[ri].Name(); lowest == "" || n < lowest {
			lowest = n
		}
	}
	return lowest
}
