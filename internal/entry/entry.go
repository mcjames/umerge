package entry

import (
	"os"
	"path/filepath"
)

// Entry is one node in the directory tree shown by umerge.
// A nil Left/Middle/Right means the file is absent on that side.
// For the two-column prototype both sides point to the same path.
type Entry struct {
	Name      string
	Path      string
	IsDir     bool
	Depth     int
	Collapsed bool
	Children  []*Entry
}

// BuildTree reads the directory at root and returns its children as a
// tree.  Subdirectory contents are loaded eagerly.  Symlinks are not
// followed.
func BuildTree(root string, depth int) ([]*Entry, error) {
	des, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	entries := make([]*Entry, 0, len(des))
	for _, de := range des {
		e := &Entry{
			Name:  de.Name(),
			Path:  filepath.Join(root, de.Name()),
			IsDir: de.IsDir(),
			Depth: depth,
		}
		if de.IsDir() {
			e.Children, _ = BuildTree(e.Path, depth+1)
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
