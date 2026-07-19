package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"umerge/internal/entry"
	"umerge/internal/mergetool"
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
)

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
	}
	m.flat = entry.Flatten(entries)
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
		return m, listenForCompare(m.compareCh)

	case compareDoneMsg:
		m.compareCh = nil
		m.comparing = false

	case toolDoneMsg:
		// The tool may have edited the file — re-derive its comparison
		// state rather than leaving whatever it was before. Synchronous
		// (a single diff/diff3 call): returning from a full-screen
		// external program already involves a redraw pause, so this
		// doesn't introduce a new stall, matching the same reasoning
		// already used for copyEntry's post-copy recompare.
		if msg.e != nil {
			m.recompareSubtree(msg.e)
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

		case "a":
			m.beginCopy('a', 'l', "left", "Copy from A (left) to:")

		case "b":
			m.beginCopy('b', 'r', "right", "Copy from B (right) to:")

		case "c":
			if m.ways == 3 {
				m.beginCopy('c', 'm', "middle", "Copy from C (middle) to:")
			}

		case "d":
			if m.readOnly {
				m.flash = "Read-only mode (--read-only): delete is disabled"
			} else if len(m.flat) > 0 {
				m.deleteEntry(m.flat[m.cursor])
			}

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
func (m *Model) beginCopy(letter, side byte, label, prompt string) {
	if m.readOnly {
		m.flash = "Read-only mode (--read-only): copy is disabled"
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
		dest := byte('r')
		if side == 'r' {
			dest = 'l'
		}
		m.copyEntry(e, side, dest)
		return
	}
	m.pendingCopyFrom = letter
	m.prompt = prompt
}

// handleCopyDestination resolves the second keypress of a 3-way copy
// prompt ("Copy from A to:" → b or c). Any key other than one of the two
// remaining columns cancels the prompt with a visible "Invalid choice"
// message (matching Python's own wording here, which is fine as-is) rather
// than silently doing nothing.
func (m Model) handleCopyDestination(key string) (tea.Model, tea.Cmd) {
	fromLetter := m.pendingCopyFrom
	m.pendingCopyFrom = 0
	m.prompt = ""

	toLetter := byte(0)
	if len(key) == 1 {
		toLetter = key[0]
	}
	valid := toLetter == 'a' || toLetter == 'b' || toLetter == 'c'
	if valid && toLetter != fromLetter && len(m.flat) > 0 {
		m.copyEntry(m.flat[m.cursor], copyLetterToSide(fromLetter), copyLetterToSide(toLetter))
	} else {
		m.flash = "Invalid choice"
	}
	return m, nil
}

func (m *Model) reflatten() {
	m.flat = entry.Flatten(m.entries)
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

	// Header: each root in its own column.
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
		for i := range texts {
			if i > 0 {
				sb.WriteString(separatorStyle(styles[i-1], styles[i]).Render("|"))
			}
			sb.WriteString(renderCell(texts[i], widths[i], styles[i], e, m.ascii))
		}
		sb.WriteByte('\n')
	}

	// Status bar.
	comparing := ""
	if m.comparing {
		comparing = "  comparing..."
	}
	status := fmt.Sprintf(" %d/%d%s  q quit  ←→/enter collapse  ↑↓/jk move  PgUp/PgDn scroll  a/b/d copy/del",
		m.cursor+1, len(m.flat), comparing)
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
	if isCursor {
		normal, unique, err = styleCursor, styleCursorUnique, styleCursorError
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
		case allPresent && (e.Compare == entry.Different || e.Compare == entry.BinaryDifferent):
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
// BinaryDifferent entries never have LMDiffs/MRDiffs set (diff3 is never
// invoked for them — see fileops.CompareThreeFiles), so they're colored
// uniformly across all columns rather than falling through the per-pair
// logic below, which would otherwise see zero counts and wrongly render
// them as unchanged.
func (m Model) diffStyleForCol(e *entry.Entry, col int, isCursor bool) lipgloss.Style {
	changed, normal := styleChanged, styleNormal
	if isCursor {
		changed, normal = styleCursorChanged, styleCursor
	}
	if m.ways == 2 || e.Compare == entry.BinaryDifferent {
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
func renderCell(text string, width int, style lipgloss.Style, e *entry.Entry, ascii bool) string {
	fitted := fit(text, width)
	if e == nil || !e.IsDir || e.Compare == entry.CompareError {
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

// ── helpers ───────────────────────────────────────────────────────────────────

func (m Model) viewHeight() int {
	h := m.height - 2 // header + status bar
	if h < 1 {
		return 1
	}
	return h
}

// colWidths distributes the terminal width evenly across m.ways columns,
// giving any remainder to the leftmost columns.
func (m Model) colWidths() []int {
	seps := m.ways - 1
	total := m.width - seps
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
// BinaryDifferent entries never get a numeric count — entryText shows a
// "bin" marker for those instead, since a hunk count doesn't apply.
func (m Model) diffCounts(e *entry.Entry) []*int {
	none := make([]*int, m.ways)
	if e.IsDir || e.Compare == entry.Uncompared || e.Compare == entry.CompareError || e.Compare == entry.BinaryDifferent {
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
