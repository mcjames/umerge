package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/mcjames/umerge/internal/entry"
	"github.com/mcjames/umerge/internal/mergetool"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleHeader = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("15"))

	styleSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	styleStatus = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("15"))

	styleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	// styleDirArrow: foreground applied to just the collapse/expand arrow
	// glyph on a directory row — always yellow, independent of the row's
	// status color, matching the Python version's dir_arrow convention
	// (dir_arrow_fg is 226 in every category; only filename_fg varies).
	// The arrow's background still follows the row's own style, applied
	// in renderCell — this is deliberately not a whole-row style.
	styleDirArrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")) // yellow

	// styleCursor: the cursor row when its entry has no diff status to
	// highlight (unchanged/uncompared) — nothing to saturate, so this
	// stays a neutral gray+yellow.
	styleCursor = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("226"))

	// styleUnique: entry exists only on some sides (black on muted green).
	styleUnique = lipgloss.NewStyle().
			Background(lipgloss.Color("#c0dcc0")).
			Foreground(lipgloss.Color("0"))

	// styleChanged: entry exists everywhere but content differs (black on steel blue).
	styleChanged = lipgloss.NewStyle().
			Background(lipgloss.Color("#a6caf0")).
			Foreground(lipgloss.Color("0"))

	// styleError: comparison, copy, or delete failed for this entry.
	styleError = lipgloss.NewStyle().
			Background(lipgloss.Color("#e06c75")).
			Foreground(lipgloss.Color("0"))

	// The cursor-row counterparts of styleUnique/styleChanged/styleError:
	// same hue, pushed to a saturated/dark background instead of the pale
	// pastel used elsewhere, with the cursor's yellow text kept on top —
	// so the cursor row stays readable *and* still shows which columns
	// actually differ, instead of the whole row going flat gray.
	styleCursorUnique = lipgloss.NewStyle().
				Background(lipgloss.Color("#1b8a3c")).
				Foreground(lipgloss.Color("226"))

	styleCursorChanged = lipgloss.NewStyle().
				Background(lipgloss.Color("#2a5db0")).
				Foreground(lipgloss.Color("226"))

	styleCursorError = lipgloss.NewStyle().
				Background(lipgloss.Color("#b3282f")).
				Foreground(lipgloss.Color("226"))

	// Hidden-entry styles (`H` reveals user-hidden items): each category's
	// own hue, pushed much darker, with medium-gray text for legibility on
	// the dark background — replaces the terminal "faint" SGR attribute
	// tried first, which read as barely-different in practice. Applied in
	// place of the cursor variants too, not on top of them: the cursor's
	// yellow highlight would otherwise still read as "bright" on a hidden
	// row, undermining the whole point of dimming it.
	styleHiddenNormal = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	styleHiddenUnique = lipgloss.NewStyle().
				Background(lipgloss.Color("#16261c")).
				Foreground(lipgloss.Color("245"))

	styleHiddenChanged = lipgloss.NewStyle().
				Background(lipgloss.Color("#17233d")).
				Foreground(lipgloss.Color("245"))

	styleHiddenError = lipgloss.NewStyle().
				Background(lipgloss.Color("#35141a")).
				Foreground(lipgloss.Color("245"))

	// styleFocusClean: a directory auto-collapsed under focus mode
	// because it's confirmed clean (TODO.md Priority 3b) — visually
	// distinct from a directory the user collapsed manually themselves,
	// which might still contain something interesting. A confirmed-clean
	// directory has (modulo an empty directory present on only one side —
	// a known, accepted edge case, see the field's own doc comment) no
	// category worth preserving the way Hidden's dimming does, so this is
	// deliberately its own plain gray-on-default style rather than reusing
	// styleHiddenNormal — same look, but the two concepts stay
	// independently named in case one needs to change later without
	// affecting the other.
	styleFocusClean = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// styleSelectedMarker: the `s`-key selection marker glyph. Its own
	// isolated style, deliberately never aliased to or derived from an
	// existing semantic style — an entry's diff-status color (unique/
	// changed/error) stays untouched when it's selected; selection is a
	// second, independent signal shown only in the marker slot. Black on
	// yellow, decided 2026-08-02 (see memory/TODO.md Priority 2).
	styleSelectedMarker = lipgloss.NewStyle().
				Background(lipgloss.Color("226")).
				Foreground(lipgloss.Color("0"))

	// Resolution-status marker colors (TODO.md Priority 4), matching the
	// Python reference's marker_ok/marker_merged/marker_resolved/
	// marker_conflict: unresolved/took-left/took-right read as "fine, no
	// action needed" (green); auto-merged/manually-resolved both read as
	// "touched, worth a glance" (yellow); conflict reads as "needs you"
	// (red). Foreground-only on the default background, unlike the
	// selection marker — this sits in its own gutter column too, so
	// there's no row content underneath it to preserve.
	styleResolutionOK = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	styleResolutionMerged = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226"))

	styleResolutionConflict = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

// selectionGutterWidth is the fixed-width prefix rendered once per row,
// reserved for the selection marker. Deliberately not per-column like the
// tree arrow: selection is subtree-wide regardless of which side(s) an
// entry appears on, so one marker per row is enough.
const selectionGutterWidth = 2

// selectionMarkerGlyphUnicode/ASCII: same split as the tree arrows
// (collapsedArrowUnicode/ASCII below) and for the same reason — U+25CF is
// in the Geometric Shapes block, "Ambiguous" East Asian Width like the
// arrows, so some terminals render it two columns wide instead of one,
// shifting the display. -A/--ascii falls back to a plain asterisk, which
// has no such ambiguity.
const (
	selectionMarkerGlyphUnicode = "●"
	selectionMarkerGlyphASCII   = "*"
)

// resolutionGutterWidth is the fixed-width prefix reserved for the
// 3-way merge resolution-status marker (TODO.md Priority 4) — one
// character plus a trailing space, same shape as the selection gutter.
// Only rendered in 3-way mode; 2-way has no resolution concept at all.
const resolutionGutterWidth = 2

// toolDoneMsg is sent when the external diff/merge tool exits. e is the
// entry that was open in the tool, so it can be re-compared — the file
// may have been edited.
type toolDoneMsg struct {
	e *entry.Entry
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for umerge.
type Model struct {
	ways       int // 2 or 3
	leftRoot   string
	middleRoot string
	rightRoot  string
	mergeTool  string         // "vim" or "emacs"
	ascii      bool           // use ASCII tree symbols (>/v) instead of Unicode (▶/▼)
	readOnly   bool           // disable copy/delete (and any future mutating command)
	ignore     *entry.Ignore  // gitignore-based filtering; nil disables it (--no-gitignore)
	entries    []*entry.Entry // source-of-truth tree
	flat       []*entry.Entry // current visible list (re-derived on collapse/expand)
	cursor     int            // index into flat
	offset     int            // index of first visible row
	width      int
	height     int
	compareCh  <-chan tea.Msg // nil when comparison is done
	comparing  bool           // true while background comparison is running

	pendingCopyFrom byte   // 0 = none; 'a'/'b'/'c' = 3-way copy awaiting a destination choice
	prompt          string // status-bar prompt shown while pendingCopyFrom is set
	flash           string // one-shot status message (e.g. "nothing to copy"), cleared on the next key

	renderHidden bool // `H` key; false by default — user-hidden entries stay out of m.flat until toggled on

	focusMode  bool // `f` key; false by default — TODO.md Priority 3b, auto-collapses confirmed-clean directories
	showCounts bool // `t` key; status bar shows clean/pending/differ counts instead of the hints line while true
}

// hiddenSkip is the Flatten skip-predicate for user-hidden entries (the `h`
// key): an entry is omitted from m.flat when it's hidden and hidden
// rendering is currently off.
func hiddenSkip(renderHidden bool) func(*entry.Entry) bool {
	return func(e *entry.Entry) bool {
		return e.Hidden && !renderHidden
	}
}

// focusSkip is focus mode's (TODO.md Priority 3b) *line*-level omission:
// an individual clean file vanishes entirely under focus mode — this is
// what actually delivers "the visible tree narrows down to just the
// files that differ" (the design's own stated goal), not just whole
// clean directories collapsing. A clean directory instead dims to one
// summary line rather than vanishing (see focusStopRecursion) — that
// distinction exists because a directory has a subtree whose shape would
// otherwise be lost by disappearing; a single file has no such subtree,
// so there's nothing to lose by hiding it outright, same as Hidden does.
func focusSkip(focusMode bool) func(*entry.Entry) bool {
	return func(e *entry.Entry) bool {
		return focusMode && !e.IsDir && e.PendingCount == 0 && e.DirtyCount == 0
	}
}

// lineSkip combines every reason an entry's own line might be omitted
// from m.flat — user-hidden (hiddenSkip) or, independently, focus mode's
// clean-file omission (focusSkip) — into the single predicate Flatten's
// skip parameter accepts.
func lineSkip(renderHidden, focusMode bool) func(*entry.Entry) bool {
	hs, fs := hiddenSkip(renderHidden), focusSkip(focusMode)
	return func(e *entry.Entry) bool {
		return hs(e) || fs(e)
	}
}

// focusStopRecursion is Flatten's other recursion lever (TODO.md
// Priority 3b): a directory's children stop being visited once it's
// confirmed clean — no diffs anywhere beneath it, and nothing left
// pending — while focus mode is on. Unlike hiddenSkip this never omits
// the directory's own line; only its descendants stop being visited.
func focusStopRecursion(focusMode bool) func(*entry.Entry) bool {
	return func(e *entry.Entry) bool {
		return focusMode && e.PendingCount == 0 && e.DirtyCount == 0
	}
}

// New creates the UI model. middleRoot is "" for two-way mode. ascii selects
// ASCII tree symbols (>/v) instead of the Unicode default (▶/▼). readOnly
// disables copy/delete — see TODO.md Priority 3 for why (git difftool -d's
// symlinked working-tree side makes those commands unexpectedly hazardous).
// ig is the compiled gitignore matcher used both for the initial tree (built
// by the caller before entries is passed in) and for any later manual
// refresh (see ops.go's beginRefresh/rebuildChildren) — nil disables
// gitignore filtering entirely.
func New(leftRoot, middleRoot, rightRoot string, entries []*entry.Entry, mergeTool string, ascii, readOnly bool, ig *entry.Ignore) Model {
	ways := 2
	if middleRoot != "" {
		ways = 3
	}
	// Focus mode's Pending/Dirty/Clean counts (TODO.md Priority 3b) need a
	// baseline before the first render or compareResultMsg increments
	// them — otherwise a directory would render as spuriously "confirmed
	// clean" (all fields zero) until the first result about it arrives.
	for _, e := range entries {
		initDiffCounts(e, ways)
	}
	ch := startCompare(entries, ways)
	m := Model{
		ways:       ways,
		leftRoot:   leftRoot,
		middleRoot: middleRoot,
		rightRoot:  rightRoot,
		mergeTool:  mergeTool,
		ascii:      ascii,
		readOnly:   readOnly,
		ignore:     ig,
		entries:    entries,
		compareCh:  ch,
		comparing:  true,
		showCounts: true, // set the moment comparison starts; see compareDoneMsg
	}
	m.flat = entry.Flatten(entries, lineSkip(m.renderHidden, m.focusMode), focusStopRecursion(m.focusMode))
	return m
}

func (m Model) Init() tea.Cmd {
	return listenForCompare(m.compareCh)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case compareResultMsg:
		msg.e.Compare = msg.state
		msg.e.NumDiffs = msg.numDiffs
		msg.e.LMDiffs = msg.lmDiffs
		msg.e.MRDiffs = msg.mrDiffs
		// Depth-bounded, unlike reflatten (see recordCompareResult) — safe
		// to run on every message a scan sends. Only reflatten when this
		// specific message could actually change what's visible under
		// focus mode (msg.e resolved Same, so it just became hideable);
		// reflattening unconditionally here would be an O(n²) trap on a
		// large tree (TODO.md Priority 3b).
		if mayChangeFocusVisibility := recordCompareResult(msg.e); mayChangeFocusVisibility && m.focusMode {
			m.reflatten()
		}
		return m, listenForCompare(m.compareCh)

	case compareDoneMsg:
		m.compareCh = nil
		m.comparing = false
		m.showCounts = false // unconditional reset — see the field's own doc comment

	case toolDoneMsg:
		// The tool may have edited the file — re-derive its comparison
		// state rather than leaving whatever it was before. Synchronous
		// (a single diff/diff3 call): returning from a full-screen
		// external program already involves a redraw pause, so this
		// doesn't introduce a new stall, matching the same reasoning
		// already used for copyEntry's post-copy recompare.
		if msg.e != nil {
			m.recompareSubtree(msg.e)
			m.updateDiffCounts(msg.e)
			if m.focusMode {
				m.reflatten()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		m.flash = ""
		if m.pendingCopyFrom != 0 {
			return m.handleCopyDestination(msg.String())
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.viewHeight() {
					m.offset = m.cursor - m.viewHeight() + 1
				}
			}

		case "left", "right":
			if len(m.flat) > 0 && m.flat[m.cursor].IsDir {
				m.flat[m.cursor].Collapsed = !m.flat[m.cursor].Collapsed
				m.reflatten()
			}

		case "enter":
			if len(m.flat) > 0 {
				e := m.flat[m.cursor]
				if e.IsDir {
					e.Collapsed = !e.Collapsed
					m.reflatten()
				} else if cmd := mergetool.Command(e, m.mergeTool); cmd != nil {
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						return toolDoneMsg{e: e}
					})
				}
			}

		case "s":
			if len(m.flat) > 0 {
				e := m.flat[m.cursor]
				if e.HasSelectedAncestor() {
					m.flash = "Deselect the containing directory first"
				} else {
					e.SetSelected(!e.Selected)
				}
			}

		case "esc":
			// Clears the whole tree-wide selection in one press, regardless
			// of cursor position — the counterpart to "s select" persisting
			// across bulk operations (see beginCopy/handleCopyDestination):
			// without this, getting out of a selection you're done with
			// means walking to and re-toggling every root by hand.
			for _, e := range selectedRoots(m.entries) {
				e.SetSelected(false)
			}

		case "a":
			m.beginCopy('a', 'l', "left", "Copy from A (left) to:")

		case "b":
			m.beginCopy('b', 'r', "right", "Copy from B (right) to:")

		case "c":
			if m.ways == 3 {
				m.beginCopy('c', 'm', "middle", "Copy from C (middle) to:")
			}

		case "m":
			if m.ways == 3 {
				m.mergeCursorItem()
			}

		case "M":
			if m.ways == 3 {
				m.mergeSelection()
			}

		case "n":
			if m.ways == 3 {
				m.mergeAll()
			}

		case "R":
			if m.ways == 3 {
				if m.readOnly {
					m.flash = "Read-only mode (--read-only): mark-resolved is disabled"
				} else if len(m.flat) > 0 {
					m.flat[m.cursor].SetResolution(entry.ResolutionManual)
				}
			}

		case "d":
			if m.readOnly {
				m.flash = "Read-only mode (--read-only): delete is disabled"
			} else if m.comparing {
				// See beginCopy's comment: deleteEntry splices the target
				// out of m.entries while the background scan may still be
				// concurrently reading the tree it's part of.
				m.flash = "Still comparing — please wait"
			} else if roots := selectedRoots(m.entries); len(roots) > 0 {
				m.bulkDelete(roots)
			} else if len(m.flat) > 0 {
				m.deleteEntry(m.flat[m.cursor])
			}

		case "h":
			if len(m.flat) > 0 {
				e := m.flat[m.cursor]
				e.SetHidden(!e.Hidden)
				m.reflatten()
			}

		case "H":
			m.renderHidden = !m.renderHidden
			m.reflatten()

		case "f":
			m.focusMode = !m.focusMode
			m.reflatten()

		case "t":
			m.showCounts = !m.showCounts

		case "r":
			if m.comparing {
				m.flash = "Still comparing — please wait"
			} else if len(m.flat) > 0 {
				// beginRefresh mutates m (compareCh/comparing) as a side
				// effect, so it must run to completion as its own
				// statement before m is read for the return below —
				// inlining it as `return m, m.beginRefresh(...)` would
				// risk m being evaluated before the mutation lands,
				// depending on evaluation order.
				cmd := m.beginRefresh(m.flat[m.cursor])
				return m, cmd
			}

		case "pgup":
			m.offset -= m.viewHeight()
			if m.offset < 0 {
				m.offset = 0
			}
			if m.cursor >= m.offset+m.viewHeight() {
				m.cursor = m.offset + m.viewHeight() - 1
			}

		case "pgdown":
			m.offset += m.viewHeight()
			maxOffset := len(m.flat) - m.viewHeight()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.offset > maxOffset {
				m.offset = maxOffset
			}
			if m.cursor < m.offset {
				m.cursor = m.offset
			}
			if m.cursor >= len(m.flat) {
				m.cursor = len(m.flat) - 1
			}
		}
	}
	return m, nil
}

// beginCopy starts a copy sourced from column "letter" (internal side
// "side"). Disabled entirely in read-only mode (see Model.readOnly).
// Otherwise, if the entry at the cursor has nothing on that side, there is
// nothing to copy — rather than silently doing nothing (the Go bug this
// replaces) or attempting it anyway and failing with a generic error (what
// the Python version's letter-based copy actually does — its
// "if source is None" guard is commented out in Model3.__copy_aux, so it
// just lets `cp` fail and marks the item ERROR, indistinguishable from a
// real I/O failure), this fails fast with a clear message before any
// prompt is shown. A destination is never invalid to choose — copying to
// an absent side is the normal case, since that's what creates it.
//
// Two-way mode has only one possible destination, so the copy runs
// immediately. Three-way mode starts the two-step "copy from X to:"
// prompt.
//
// When a selection exists (see selectedRoots), it takes priority over the
// cursor item entirely — bulk copy acts on every selected root, and the
// per-item "nothing on that side" check below doesn't apply up front
// (bulkCopy skips individual items missing that side instead of failing
// the whole operation; see its comment).
func (m *Model) beginCopy(letter, side byte, label, prompt string) {
	if m.readOnly {
		m.flash = "Read-only mode (--read-only): copy is disabled"
		return
	}
	// The background comparison goroutine (see startCompare/walkAndCompare)
	// concurrently reads Left/Middle/Right/Children on entries throughout
	// the tree, with no synchronization. copyEntry writes those same fields
	// (setSide, rebuildChildren replacing Children wholesale), so allowing
	// a copy while comparing is true would be a real, unsynchronized data
	// race — not hypothetical, just ordinary use (copy something before a
	// large tree's initial scan finishes). Blocking here is sufficient for
	// the 3-way prompt too: 'r' (the only thing that could start a new
	// compare) is intercepted by Update() whenever pendingCopyFrom is set,
	// so comparing can't flip false→true while a prompt is open once this
	// check has passed.
	if m.comparing {
		m.flash = "Still comparing — please wait"
		return
	}
	if roots := selectedRoots(m.entries); len(roots) > 0 {
		if m.ways == 2 {
			m.bulkCopy(roots, side, twoWayDest(side))
			return
		}
		m.pendingCopyFrom = letter
		m.prompt = prompt
		return
	}
	if len(m.flat) == 0 {
		return
	}
	e := m.flat[m.cursor]
	if getSide(e, side) == nil {
		m.flash = "Nothing to copy: " + label + " is absent"
		return
	}
	if m.ways == 2 {
		m.copyEntry(e, side, twoWayDest(side))
		return
	}
	m.pendingCopyFrom = letter
	m.prompt = prompt
}

// handleCopyDestination resolves the second keypress of a 3-way copy
// prompt ("Copy from A to:" → b or c). Any key other than one of the two
// remaining columns cancels the prompt with a visible "Invalid choice"
// message (matching Python's own wording here, which is fine as-is) rather
// than silently doing nothing. If a selection existed when the prompt was
// opened, it still does now — no other key handling runs while a prompt is
// pending, so re-deriving it here is equivalent to snapshotting it at
// beginCopy time.
func (m Model) handleCopyDestination(key string) (tea.Model, tea.Cmd) {
	fromLetter := m.pendingCopyFrom
	m.pendingCopyFrom = 0
	m.prompt = ""

	toLetter := byte(0)
	if len(key) == 1 {
		toLetter = key[0]
	}
	valid := toLetter == 'a' || toLetter == 'b' || toLetter == 'c'
	if !valid || toLetter == fromLetter {
		m.flash = "Invalid choice"
		return m, nil
	}

	from, to := copyLetterToSide(fromLetter), copyLetterToSide(toLetter)
	if roots := selectedRoots(m.entries); len(roots) > 0 {
		m.bulkCopy(roots, from, to)
	} else if len(m.flat) > 0 {
		m.copyEntry(m.flat[m.cursor], from, to)
	} else {
		m.flash = "Invalid choice"
	}
	return m, nil
}

func (m *Model) reflatten() {
	m.flat = entry.Flatten(m.entries, lineSkip(m.renderHidden, m.focusMode), focusStopRecursion(m.focusMode))
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.viewHeight() {
		m.offset = m.cursor - m.viewHeight() + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	widths := m.colWidths()

	var sb strings.Builder

	// Header: blank selection and resolution gutters (nothing to select
	// or resolve on the header itself), then each root in its own column.
	sb.WriteString(selectionGutter(nil, m.ascii))
	sb.WriteString(resolutionGutter(nil, m.ways))
	headerStyles := make([]lipgloss.Style, len(m.roots()))
	for i := range headerStyles {
		headerStyles[i] = styleHeader
	}
	for i, root := range m.roots() {
		if i > 0 {
			sb.WriteString(separatorStyle(headerStyles[i-1], headerStyles[i]).Render("|"))
		}
		sb.WriteString(styleHeader.Render(fit(root, widths[i])))
	}
	sb.WriteByte('\n')

	// Content rows.
	for row := 0; row < m.viewHeight(); row++ {
		idx := m.offset + row
		texts, styles := m.rowCols(idx, idx == m.cursor)
		var e *entry.Entry
		if idx < len(m.flat) {
			e = m.flat[idx]
		}
		sb.WriteString(selectionGutter(e, m.ascii))
		sb.WriteString(resolutionGutter(e, m.ways))
		for i := range texts {
			if i > 0 {
				sb.WriteString(separatorStyle(styles[i-1], styles[i]).Render("|"))
			}
			sb.WriteString(renderCell(texts[i], widths[i], styles[i], e, m.ascii, idx == m.cursor))
		}
		sb.WriteByte('\n')
	}

	// Status bar.
	comparing := ""
	if m.comparing {
		comparing = "  comparing..."
	}
	selected := ""
	// Shown regardless of scroll position — selection is tree-wide, so a
	// selected item can easily be scrolled out of view. Without this, "d"
	// (or "a"/"b"/"c") could bulk-act on a forgotten selection somewhere
	// else in the tree while looking, from the cursor's row alone, like an
	// ordinary single-item action.
	if n := len(selectedRoots(m.entries)); n > 0 {
		selected = fmt.Sprintf("  %d selected (esc clear)", n)
	}
	// The default status-bar slot: clean/pending/differ counts while
	// showCounts is on (set automatically the moment comparison starts,
	// reset the moment it finishes — see compareDoneMsg — but toggleable
	// with `t` at any time in either phase), otherwise the usual hints
	// line. prompt/flash below still take priority over either — they're
	// transient, higher-urgency signals; this only governs what shows
	// when nothing else is going on.
	defaultSlot := "q quit  ←→/enter collapse  ↑↓/jk move  PgUp/PgDn scroll  s select  a/b/d copy/del"
	if m.ways == 3 {
		defaultSlot += "  m/M/n merge  R resolved"
	}
	if m.showCounts {
		defaultSlot = m.diffCountsSummary()
	}
	status := fmt.Sprintf(" %d/%d%s%s  %s",
		m.cursor+1, len(m.flat), comparing, selected, defaultSlot)
	switch {
	case m.prompt != "":
		status = " " + m.prompt
	case m.flash != "":
		status = " " + m.flash
	}
	sb.WriteString(styleStatus.Render(fit(status, m.width)))

	return sb.String()
}

// rowCols returns the display text and style for each column of row idx.
func (m Model) rowCols(idx int, isCursor bool) ([]string, []lipgloss.Style) {
	texts := make([]string, m.ways)
	styles := make([]lipgloss.Style, m.ways)
	for i := range styles {
		styles[i] = styleNormal
	}

	if idx >= len(m.flat) {
		return texts, styles
	}

	e := m.flat[idx]
	paths := m.paths(e)

	counts := m.diffCounts(e)
	for i, p := range paths {
		texts[i] = entryText(e, p, counts[i], m.ascii)
	}

	normal, unique, err := styleNormal, styleUnique, styleError
	switch {
	case isCursor:
		// The cursor row looks the same whether or not the entry under it
		// is hidden — the point of Hidden's own dark/gray look is to read
		// as different *from the cursor row's other rows*; overriding the
		// cursor itself would make the cursor unreadable exactly when
		// resting on a hidden entry. Moving the cursor off it is how you
		// tell it's hidden.
		normal, unique, err = styleCursor, styleCursorUnique, styleCursorError
	case e.Hidden:
		normal, unique, err = styleHiddenNormal, styleHiddenUnique, styleHiddenError
	case m.focusMode && e.IsDir && e.PendingCount == 0 && e.DirtyCount == 0:
		// Same "confirmed clean" test as focusStopRecursion — a directory
		// whose children just stopped being visited because there's
		// nothing left in it worth looking at, dimmed to read as
		// distinct from a directory the user collapsed themselves.
		normal, unique, err = styleFocusClean, styleFocusClean, styleFocusClean
	}

	// Determine whether every side is present.
	allPresent := true
	for _, p := range paths {
		if p == nil {
			allPresent = false
			break
		}
	}

	for i, p := range paths {
		switch {
		case e.Compare == entry.CompareError:
			// A prior compare, copy, or delete failed for this entry.
			// Takes priority over the presence-based cases below —
			// otherwise a failed copy that never set its destination
			// pointer would just look like a normal absent side.
			if p != nil {
				styles[i] = err
			} else {
				styles[i] = normal
			}
		case allPresent && (e.Compare == entry.Different || e.Compare == entry.BinaryDifferent || e.Compare == entry.SymlinkDifferent):
			styles[i] = m.diffStyleForCol(e, i, isCursor)
		case allPresent:
			// Same or still Uncompared — normal white.
			styles[i] = normal
		case p != nil:
			// Present on this side but absent on at least one other.
			styles[i] = unique
		default:
			styles[i] = normal
		}
	}

	return texts, styles
}

// diffStyleForCol returns the appropriate style for column col when an entry
// is present on all sides but has differences.
//
// 2-way: both columns blue (only one comparison).
// 3-way: mirrors Python's per-pair logic —
//
//	lmDiffs > 0  →  left + middle blue
//	mrDiffs > 0  →  middle + right blue
//
// BinaryDifferent/SymlinkDifferent entries never have LMDiffs/MRDiffs set
// (diff3 is never invoked for them — see fileops.CompareThreeFiles and
// ui.compareSymlinks), so they're colored uniformly across all columns
// rather than falling through the per-pair logic below, which would
// otherwise see zero counts and wrongly render them as unchanged.
func (m Model) diffStyleForCol(e *entry.Entry, col int, isCursor bool) lipgloss.Style {
	changed, normal := styleChanged, styleNormal
	switch {
	case isCursor: // see rowCols: the cursor row looks the same whether or not it's hidden
		changed, normal = styleCursorChanged, styleCursor
	case e.Hidden:
		changed, normal = styleHiddenChanged, styleHiddenNormal
	}
	if m.ways == 2 || e.Compare == entry.BinaryDifferent || e.Compare == entry.SymlinkDifferent {
		return changed
	}
	// 3-way: color only the columns adjacent to the differing pair.
	switch col {
	case 0: // left: blue if left↔middle differ
		if e.LMDiffs > 0 {
			return changed
		}
	case 1: // middle: blue if either pair differs
		if e.LMDiffs > 0 || e.MRDiffs > 0 {
			return changed
		}
	case 2: // right: blue if middle↔right differ
		if e.MRDiffs > 0 {
			return changed
		}
	}
	return normal
}

// separatorStyle picks the style for the "|" between two adjacent
// columns. If both columns share the same *real* background color (green,
// blue, the cursor's gray, an error's red, ...) the separator matches it
// so the color reads as one continuous block instead of being interrupted
// by a flat gray bar. Two plain/unstyled columns don't count as "sharing a
// color" just because neither has one set — that would color the
// separator white on every ordinary row, which isn't a highlight, just
// noise. (Python always colors a separator to match the column on its
// right, regardless of whether the left side matches — we deliberately do
// it differently: only when both sides genuinely share a highlight color,
// which reads as more intentional and doesn't imply a boundary that isn't
// really there.)
//
// lipgloss.Style isn't comparable with ==, so this compares the
// configured background color instead. GetBackground() returns
// lipgloss.NoColor{} when nothing was ever set.
func separatorStyle(left, right lipgloss.Style) lipgloss.Style {
	bg := left.GetBackground()
	if bg == right.GetBackground() && bg != (lipgloss.NoColor{}) {
		return left
	}
	return styleSep
}

// renderCell renders one column's already-fitted text. For a directory
// row, only the collapse/expand arrow glyph gets the dedicated yellow
// arrow color (matching Python: dir_arrow_fg is always yellow, in every
// status category, while the filename itself uses that category's normal
// foreground) — the rest of the text, critically including the directory
// name, keeps the column's own style. Skipped for CompareError rows: an
// error should read as a single, unambiguous red line, not a yellow arrow
// on a red background.
func renderCell(text string, width int, style lipgloss.Style, e *entry.Entry, ascii, isCursor bool) string {
	fitted := fit(text, width)
	if e == nil || !e.IsDir || e.Compare == entry.CompareError || (e.Hidden && !isCursor) {
		// A hidden row should read as one uniform muted line, not a yellow
		// arrow breaking up the dimming — same reasoning as the
		// CompareError skip just above. But not when it's also the cursor
		// row: the cursor's look (arrow included) stays identical whether
		// or not the entry under it is hidden, matching rowCols.
		return style.Render(fitted)
	}
	// The arrow is 2 *bytes* for the ASCII symbols ("> "/"v ") but 4 for
	// the Unicode ones ("▶ "/"▼ " — each triangle is a 3-byte UTF-8
	// sequence plus a 1-byte space). Both symbols within a mode share the
	// same byte length, so this is a fixed lookup, not a per-call
	// computation. runewidth.Truncate (inside fit) only ever cuts at rune
	// boundaries, so this is always a valid slice point unless the arrow
	// itself got truncated away entirely — which the bounds check below
	// catches.
	arrowBytes := len(collapsedArrowUnicode)
	if ascii {
		arrowBytes = len(collapsedArrowASCII)
	}
	indentLen := 2 * e.Depth
	if indentLen+arrowBytes > len(fitted) {
		return style.Render(fitted)
	}
	indent := fitted[:indentLen]
	arrow := fitted[indentLen : indentLen+arrowBytes]
	rest := fitted[indentLen+arrowBytes:]
	arrowStyle := style.Foreground(styleDirArrow.GetForeground())
	return style.Render(indent) + arrowStyle.Render(arrow) + style.Render(rest)
}

// selectionGutter renders the fixed-width selection marker slot at the
// very start of a row: the marker glyph (styleSelectedMarker) for a
// selected entry, or blank space otherwise. e is nil for the header row
// and for filler rows past the end of m.flat, both of which just get
// blank space. ascii selects the ASCII marker fallback, matching the tree
// arrows' -A/--ascii convention.
func selectionGutter(e *entry.Entry, ascii bool) string {
	if e != nil && e.Selected {
		glyph := selectionMarkerGlyphUnicode
		if ascii {
			glyph = selectionMarkerGlyphASCII
		}
		return styleSelectedMarker.Render(glyph) + " "
	}
	return strings.Repeat(" ", selectionGutterWidth)
}

// resolutionGutter renders the fixed-width resolution-status marker slot
// (TODO.md Priority 4), immediately after the selection gutter: the
// status character in its category color for a 3-way entry, or blank
// space in 2-way mode (no resolution concept there) or for the header/
// filler rows, where e is nil.
func resolutionGutter(e *entry.Entry, ways int) string {
	if ways != 3 || e == nil {
		return strings.Repeat(" ", resolutionGutterWidth)
	}
	style := styleResolutionOK
	switch e.Resolution {
	case entry.ResolutionMerged, entry.ResolutionManual:
		style = styleResolutionMerged
	case entry.ResolutionConflict:
		style = styleResolutionConflict
	}
	return style.Render(string(e.Resolution.Char())) + " "
}

// resolutionGutterWidth reports how much horizontal space the resolution
// gutter actually occupies for this model — colWidths needs to reserve
// it in 3-way mode and not in 2-way, where the gutter itself renders as
// nothing extra beyond what's already blank.
func (m Model) resolutionGutterWidth() int {
	if m.ways == 3 {
		return resolutionGutterWidth
	}
	return 0
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m Model) viewHeight() int {
	h := m.height - 2 // header + status bar
	if h < 1 {
		return 1
	}
	return h
}

// colWidths distributes the terminal width evenly across m.ways columns,
// giving any remainder to the leftmost columns. Reserves
// selectionGutterWidth off the top for the selection marker gutter, which
// View renders before column 0 on every row.
func (m Model) colWidths() []int {
	seps := m.ways - 1
	total := m.width - seps - selectionGutterWidth - m.resolutionGutterWidth()
	if total < m.ways {
		total = m.ways
	}
	base := total / m.ways
	extra := total % m.ways
	widths := make([]int, m.ways)
	for i := range widths {
		widths[i] = base
		if i < extra {
			widths[i]++
		}
	}
	return widths
}

// roots returns the header strings in column order.
func (m Model) roots() []string {
	if m.ways == 2 {
		return []string{m.leftRoot, m.rightRoot}
	}
	return []string{m.leftRoot, m.middleRoot, m.rightRoot}
}

// paths returns the path pointers for e in column order.
func (m Model) paths(e *entry.Entry) []*string {
	if m.ways == 2 {
		return []*string{e.Left, e.Right}
	}
	return []*string{e.Left, e.Middle, e.Right}
}

// diffCounts returns a per-column diff count pointer (nil = don't show).
// For 2-way: [left count, nil]. For 3-way: [lm count, nil, mr count].
// BinaryDifferent/SymlinkDifferent entries never get a numeric count —
// entryText shows a "bin"/"link" marker for those instead, since a hunk
// count doesn't apply to either.
func (m Model) diffCounts(e *entry.Entry) []*int {
	none := make([]*int, m.ways)
	if e.IsDir || e.Compare == entry.Uncompared || e.Compare == entry.CompareError ||
		e.Compare == entry.BinaryDifferent || e.Compare == entry.SymlinkDifferent {
		return none
	}
	counts := make([]*int, m.ways)
	if m.ways == 2 {
		n := e.NumDiffs
		counts[0] = &n
	} else {
		lm, mr := e.LMDiffs, e.MRDiffs
		counts[0] = &lm
		counts[2] = &mr
	}
	return counts
}

// diffCountsSummary renders the "N clean · M pending · K differ" status
// line shown while showCounts is on (TODO.md Priority 3b), summed across
// every top-level entry — the root's totals fall out for free from the
// same Pending/Dirty/Clean bookkeeping focus mode's collapse-gating
// already needs, no separate tally required. The pending segment is
// dropped entirely once it reaches zero, rather than showing "0 pending".
func (m Model) diffCountsSummary() string {
	var clean, pending, dirty int
	for _, e := range m.entries {
		clean += e.CleanCount
		pending += e.PendingCount
		dirty += e.DirtyCount
	}
	s := fmt.Sprintf("%d clean", clean)
	if pending > 0 {
		s += fmt.Sprintf(" · %d pending", pending)
	}
	s += fmt.Sprintf(" · %d differ", dirty)
	return s
}

// collapsedArrow/expandedArrow: the Unicode default looks better and
// renders correctly in most terminals (confirmed: WezTerm); some
// terminals give these "Ambiguous" East Asian Width characters the wrong
// column width (confirmed: COSMIC terminal), which is what the ascii
// fallback (-A/--ascii) is for. See CLAUDE.md and TODO.md Priority 9.
const (
	collapsedArrowUnicode = "▶ "
	expandedArrowUnicode  = "▼ "
	collapsedArrowASCII   = "> "
	expandedArrowASCII    = "v "
)

// entryText returns the display text for one side of an entry.
// count is non-nil when a diff count should be appended.
// Returns "" (blank cell) when path is nil.
func entryText(e *entry.Entry, path *string, count *int, ascii bool) string {
	if path == nil {
		return ""
	}
	indent := strings.Repeat("  ", e.Depth)
	var arrow string
	if e.IsDir {
		if e.Collapsed {
			if ascii {
				arrow = collapsedArrowASCII
			} else {
				arrow = collapsedArrowUnicode
			}
		} else {
			if ascii {
				arrow = expandedArrowASCII
			} else {
				arrow = expandedArrowUnicode
			}
		}
	} else {
		arrow = "  "
	}
	text := indent + arrow + filepath.Base(*path)
	switch {
	case e.Compare == entry.BinaryDifferent:
		// No hunk count applies — diff/diff3 are never invoked for this
		// entry (see fileops.CompareTwoFiles/CompareThreeFiles).
		text += " bin"
	case e.Compare == entry.SymlinkDifferent:
		// No hunk count applies — classified by link target, not content
		// (see ui.compareSymlinks).
		text += " link"
	case count != nil:
		if *count == 0 {
			text += " ="
		} else {
			text += " " + strconv.Itoa(*count)
		}
	}
	return text
}

// fit truncates or pads s to exactly width display columns.
func fit(s string, width int) string {
	s = runewidth.Truncate(s, width, "")
	return s + strings.Repeat(" ", width-runewidth.StringWidth(s))
}
