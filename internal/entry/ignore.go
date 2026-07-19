package entry

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// Ignore wraps compiled .gitignore patterns gathered from the comparison
// roots, plus an always-on rule for ".git" itself. A nil *Ignore matches
// nothing (see Match), which is how gitignore filtering is disabled
// entirely (e.g. --no-gitignore).
type Ignore struct {
	gi *gitignore.GitIgnore
}

// LoadIgnore reads a top-level .gitignore from each given root (roots that
// are nil, or have no .gitignore file, are skipped) and compiles their
// combined patterns into one matcher. ".git/" is always added, so umerge
// never surfaces git's own internal object store as a difference, even
// with no user .gitignore at all.
//
// Only root-level .gitignore files are read — real git also honors
// .gitignore files nested in subdirectories, cascading down the tree; that
// is a known follow-up (see TODO.md Priority 2) rather than something
// handled here, since a single repo-root .gitignore is the overwhelmingly
// common case.
func LoadIgnore(roots ...*string) *Ignore {
	lines := []string{".git/"}
	for _, root := range roots {
		if root == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(*root, ".gitignore"))
		if err != nil {
			continue
		}
		lines = append(lines, strings.Split(string(data), "\n")...)
	}
	return &Ignore{gi: gitignore.CompileIgnoreLines(lines...)}
}

// Match reports whether relPath (slash-separated, relative to the
// comparison roots, shared across all sides) should be excluded. isDir
// must be true when the entry is a directory: gitignore's directory-only
// patterns (a trailing "/", e.g. "node_modules/") are compiled into a
// regexp that only matches a candidate path that itself ends in a slash.
func (ig *Ignore) Match(relPath string, isDir bool) bool {
	if ig == nil {
		return false
	}
	if isDir {
		relPath += "/"
	}
	return ig.gi.MatchesPath(relPath)
}
