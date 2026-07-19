package mergetool

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mcjames/umerge/internal/entry"
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

// changedHex/uniqueHex mirror internal/ui/app.go's styleChanged/styleUnique
// backgrounds exactly, so a file opened in vimdiff reads with the same
// color meaning as the directory tree it was opened from — matching how
// Araxis Merge uses one consistent palette across its directory and file
// comparison views, rather than vim's own unrelated defaults. Keep these
// in sync with app.go by hand; there's no shared package between ui and
// mergetool to hang a single source of truth off of for just two colors.
const (
	changedHex = "#a6caf0" // styleChanged: present everywhere, content differs
	uniqueHex  = "#c0dcc0" // styleUnique: present on some sides only
)

// diffHighlightArgs returns vim -c flags that recolor vimdiff's built-in
// diff highlight groups to match umerge's own directory-view palette:
//   - DiffChange/DiffText (a line — or, for DiffText, the specific changed
//     words within it — differs between the two/three files) get the same
//     blue as a changed entry in the tree.
//   - DiffAdd (a line present in this buffer but not the other) gets the
//     same green as an entry present on only some sides.
//   - DiffDelete (the filler shown in this buffer for lines only the
//     *other* buffer has) gets a plain neutral gray with matching
//     foreground, so the filler reads as "nothing here" rather than a
//     third loud color — umerge itself never colors an absent side, it's
//     just left blank.
//
// ctermbg values are the closest xterm-256 approximations of the same
// hues, for terminals without true-color support; guibg/guifg carry the
// exact hex for anything that does.
func diffHighlightArgs() []string {
	return []string{
		"-c", "highlight DiffChange ctermbg=153 ctermfg=black guibg=" + changedHex + " guifg=black",
		"-c", "highlight DiffText ctermbg=153 ctermfg=black cterm=bold guibg=" + changedHex + " guifg=black gui=bold",
		"-c", "highlight DiffAdd ctermbg=151 ctermfg=black guibg=" + uniqueHex + " guifg=black",
		"-c", "highlight DiffDelete ctermbg=240 ctermfg=240 guibg=#444444 guifg=#444444",
	}
}

func vimCommand(paths []string) *exec.Cmd {
	switch len(paths) {
	case 1:
		return exec.Command("vim", paths[0])
	default:
		args := append(diffHighlightArgs(), paths...)
		return exec.Command("vimdiff", args...)
	}
}

func emacsCommand(paths []string) *exec.Cmd {
	switch len(paths) {
	case 1:
		return exec.Command("emacs", paths[0])
	case 2:
		return exec.Command("emacs", "--eval",
			fmt.Sprintf(`(ediff-files "%s" "%s")`, elispQuote(paths[0]), elispQuote(paths[1])))
	default:
		return exec.Command("emacs", "--eval",
			fmt.Sprintf(`(ediff3 "%s" "%s" "%s")`, elispQuote(paths[0]), elispQuote(paths[1]), elispQuote(paths[2])))
	}
}

// elispQuote escapes s for safe embedding inside an Emacs Lisp string
// literal (backslash, then double-quote). Without this, a filename
// containing a literal `"` breaks out of the string early — e.g. a path
// like `foo".png") (shell-command "...`  — turning the rest of the
// attacker-controlled filename into arbitrary Lisp that --eval happily
// runs. Not hypothetical: umerge's own stated use case includes comparing
// vendor code drops and other untrusted trees, where a crafted filename is
// exactly the kind of thing that could show up.
func elispQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
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
