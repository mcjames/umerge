package ui

import (
	"fmt"
	"path/filepath"
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

	styleDir = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")) // yellow

	styleCursor = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("226"))

	// styleUnique: entry exists only on some sides (black on muted green).
	styleUnique = lipgloss.NewStyle().
			Background(lipgloss.Color("108")).
			Foreground(lipgloss.Color("0"))

	// styleChanged: entry exists everywhere but content differs (black on steel blue).
	// Used once file comparison is implemented.
	styleChanged = lipgloss.NewStyle().
			Background(lipgloss.Color("67")).
			Foreground(lipgloss.Color("0"))
)

// toolDoneMsg is sent when the external diff/merge tool exits.
type toolDoneMsg struct{}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for umerge.
type Model struct {
	ways      int    // 2 or 3
	leftRoot  string
	middleRoot string
	rightRoot string
	entries   []*entry.Entry // source-of-truth tree
	flat      []*entry.Entry // current visible list (re-derived on collapse/expand)
	cursor    int            // index into flat
	offset    int            // index of first visible row
	width     int
	height    int
}

// New creates the UI model. middleRoot is "" for two-way mode.
func New(leftRoot, middleRoot, rightRoot string, entries []*entry.Entry) Model {
	ways := 2
	if middleRoot != "" {
		ways = 3
	}
	m := Model{
		ways:       ways,
		leftRoot:   leftRoot,
		middleRoot: middleRoot,
		rightRoot:  rightRoot,
		entries:    entries,
	}
	m.flat = entry.Flatten(entries)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case toolDoneMsg:
		// Tool exited. Will trigger re-comparison once diff is implemented.

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
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
				} else if cmd := mergetool.Command(e); cmd != nil {
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						return toolDoneMsg{}
					})
				}
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
	sep := styleSep.Render("|")

	var sb strings.Builder

	// Header: each root in its own column.
	for i, root := range m.roots() {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(styleHeader.Render(fit(root, widths[i])))
	}
	sb.WriteByte('\n')

	// Content rows.
	for row := 0; row < m.viewHeight(); row++ {
		idx := m.offset + row
		texts, styles := m.rowCols(idx, idx == m.cursor)
		for i := range texts {
			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(styles[i].Render(fit(texts[i], widths[i])))
		}
		sb.WriteByte('\n')
	}

	// Status bar.
	status := fmt.Sprintf(" %d/%d  q quit  ←→/enter collapse  ↑↓/jk move  PgUp/PgDn scroll",
		m.cursor+1, len(m.flat))
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

	for i, p := range paths {
		texts[i] = entryText(e, p)
	}

	if isCursor {
		for i := range styles {
			styles[i] = styleCursor
		}
		return texts, styles
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
		case allPresent:
			// All sides have the entry. Normal coloring; blue added when
			// file comparison is implemented.
			if e.IsDir {
				styles[i] = styleDir
			} else {
				styles[i] = styleNormal
			}
		case p != nil:
			// Present on this side but absent on at least one other.
			styles[i] = styleUnique
		default:
			// Absent on this side — blank cell, default terminal color.
			styles[i] = styleNormal
		}
	}

	return texts, styles
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

// entryText returns the display text for one side of an entry.
// Returns "" (blank cell) when path is nil.
func entryText(e *entry.Entry, path *string) string {
	if path == nil {
		return ""
	}
	indent := strings.Repeat("  ", e.Depth)
	var arrow string
	if e.IsDir {
		if e.Collapsed {
			arrow = "> "
		} else {
			arrow = "v "
		}
	} else {
		arrow = "  "
	}
	return indent + arrow + filepath.Base(*path)
}

// fit truncates or pads s to exactly width display columns.
func fit(s string, width int) string {
	s = runewidth.Truncate(s, width, "")
	return s + strings.Repeat(" ", width-runewidth.StringWidth(s))
}

// Silence unused-variable warning for styleChanged until file comparison lands.
var _ = styleChanged
