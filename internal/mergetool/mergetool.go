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
//	one file   → emacs -nw <file>
//	two files  → emacs -nw --eval (ediff-files "left" "right")
//	three files → emacs -nw --eval (ediff3 "left" "middle" "right")
//
// -nw forces emacs to stay in the invoking terminal rather than opening a
// new GUI frame — otherwise plain `emacs` pops a separate window whenever
// a display is available, which breaks the "suspend the TUI, run the tool
// inline, resume" model umerge otherwise offers consistently for both
// vim and emacs (vim never has this problem: it stays in the terminal it
// was invoked from by default).
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

// ediffFaceArgs returns emacs --eval expressions that recolor ediff's
// background diff faces (ediff-odd-diff-<letter>/ediff-even-diff-<letter>
// — every diff region gets one or the other) to match umerge's own
// directory-view palette, mirroring diffHighlightArgs' vim treatment.
// letters is which buffer identifiers ediff uses for this comparison —
// "A"/"B" for a plain ediff-files (2-way), "A"/"B"/"C" for a plain ediff3
// (3-way; umerge never calls the ancestor-merge variant, so the separate
// "Ancestor" faces ediff also defines are irrelevant here).
//
// Unlike vim, ediff has no separate "this region is a pure insertion, not
// present on the other side" face (no equivalent of DiffAdd) — verified
// against ediff-init.el's actual defface list. Every diff region, edit or
// one-sided insertion alike, gets the same blue "changed" hue; there's no
// green "unique" treatment to carry over.
//
// Deliberately does NOT theme ediff-current-diff-<letter> (the hunk the
// cursor is on) or ediff-fine-diff-<letter> (word-level highlighting
// within a hunk), even though both exist and an earlier version of this
// function set them. Verified empirically (2026-07-23, after switching to
// -nw so ediff stays in the terminal rather than opening a GUI frame)
// that neither actually renders distinctly in a real -nw session — every
// region just shows the plain odd/even background color, even after
// forcing ediff-use-faces/ediff-force-faces. `ediff-highlight-diff` in
// ediff-util.el is directly documented "Invoked for X displays only",
// and fine-diff word-level highlighting showed the same behavior in
// testing (confirmed with a clean single-word-difference case — no bold
// emphasis on just the differing word, the whole line got one flat
// color). So there's nothing to theme there in the mode umerge actually
// launches in; setting those faces anyway would be dead code implying an
// effect that never happens.
//
// Applied via --eval at launch time, not by editing the user's init
// file — only affects umerge-launched sessions, same principle as vim's
// -c flags. Unlike vim's ctermbg/guibg split, no separate terminal-color
// value is needed: Emacs approximates a hex color to the terminal's
// actual palette itself (confirmed via real -nw sessions rendering true
// 24-bit color escape sequences for these faces).
func ediffFaceArgs(letters []string) []string {
	args := []string{"--eval", "(require 'ediff-init)"}
	for _, l := range letters {
		args = append(args,
			"--eval", fmt.Sprintf(`(set-face-attribute 'ediff-odd-diff-%s nil :background "%s" :foreground "black")`, l, changedHex),
			"--eval", fmt.Sprintf(`(set-face-attribute 'ediff-even-diff-%s nil :background "%s" :foreground "black")`, l, changedHex),
		)
	}
	return args
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
		return exec.Command("emacs", "-nw", paths[0])
	case 2:
		args := append([]string{"-nw"}, ediffFaceArgs([]string{"A", "B"})...)
		args = append(args, "--eval",
			fmt.Sprintf(`(ediff-files "%s" "%s")`, elispQuote(paths[0]), elispQuote(paths[1])))
		return exec.Command("emacs", args...)
	default:
		args := append([]string{"-nw"}, ediffFaceArgs([]string{"A", "B", "C"})...)
		args = append(args, "--eval",
			fmt.Sprintf(`(ediff3 "%s" "%s" "%s")`, elispQuote(paths[0]), elispQuote(paths[1]), elispQuote(paths[2])))
		return exec.Command("emacs", args...)
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
