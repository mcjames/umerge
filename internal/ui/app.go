package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"umerge/internal/entry"
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
)

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for umerge.
type Model struct {
	root    string
	entries []*entry.Entry // source-of-truth tree
	flat    []*entry.Entry // current visible list (re-derived on collapse/expand)
	cursor  int            // index into flat
	offset  int            // index of first visible row
	width   int
	height  int
}

func New(root string, entries []*entry.Entry) Model {
	m := Model{root: root, entries: entries}
	m.flat = entry.Flatten(entries)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

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

		case "left", "right", "enter":
			if len(m.flat) > 0 && m.flat[m.cursor].IsDir {
				m.flat[m.cursor].Collapsed = !m.flat[m.cursor].Collapsed
				m.reflatten()
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

// reflatten rebuilds the flat list after a collapse/expand and clamps
// cursor and offset to valid positions.
func (m *Model) reflatten() {
	m.flat = entry.Flatten(m.entries)
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// keep cursor inside the viewport
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

	lw := m.leftWidth()
	rw := m.rightWidth()
	sep := styleSep.Render("|")

	var sb strings.Builder

	// Header: root path in both columns.
	sb.WriteString(styleHeader.Render(fit(m.root, lw)))
	sb.WriteString(sep)
	sb.WriteString(styleHeader.Render(fit(m.root, rw)))
	sb.WriteByte('\n')

	// Content rows.
	for row := 0; row < m.viewHeight(); row++ {
		idx := m.offset + row
		isCursor := idx == m.cursor

		var text string
		var isDir bool
		if idx < len(m.flat) {
			text = entryText(m.flat[idx])
			isDir = m.flat[idx].IsDir
		}

		lCell := fit(text, lw)
		rCell := fit(text, rw)

		switch {
		case isCursor:
			sb.WriteString(styleCursor.Render(lCell))
			sb.WriteString(sep)
			sb.WriteString(styleCursor.Render(rCell))
		case isDir:
			sb.WriteString(styleDir.Render(lCell))
			sb.WriteString(sep)
			sb.WriteString(styleDir.Render(rCell))
		default:
			sb.WriteString(styleNormal.Render(lCell))
			sb.WriteString(sep)
			sb.WriteString(styleNormal.Render(rCell))
		}
		sb.WriteByte('\n')
	}

	// Status bar.
	status := fmt.Sprintf(" %d/%d  q quit  ←→/enter collapse  ↑↓/jk move  PgUp/PgDn scroll",
		m.cursor+1, len(m.flat))
	sb.WriteString(styleStatus.Render(fit(status, m.width)))

	return sb.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m Model) viewHeight() int {
	h := m.height - 2 // header + status bar
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) leftWidth() int {
	if m.width < 3 {
		return 1
	}
	return (m.width - 1) / 2
}

func (m Model) rightWidth() int {
	if m.width < 3 {
		return 1
	}
	return m.width - 1 - m.leftWidth()
}

// entryText builds the display string for one entry (indent + arrow + name).
func entryText(e *entry.Entry) string {
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
	return indent + arrow + e.Name
}

// fit truncates or pads s to exactly width display columns.
func fit(s string, width int) string {
	s = runewidth.Truncate(s, width, "")
	return s + strings.Repeat(" ", width-runewidth.StringWidth(s))
}
