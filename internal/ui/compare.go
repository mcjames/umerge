package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"umerge/internal/entry"
	"umerge/internal/fileops"
)

// compareResultMsg carries the result of one file comparison back to Update.
type compareResultMsg struct {
	e        *entry.Entry
	state    entry.CompareState
	numDiffs int // 2-way
	lmDiffs  int // 3-way left↔middle
	mrDiffs  int // 3-way middle↔right
}

// compareDoneMsg is sent when all comparisons have finished.
type compareDoneMsg struct{}

// listenForCompare returns a Cmd that blocks until one message arrives on ch.
func listenForCompare(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// startCompare launches the comparison goroutine and returns the channel to
// listen on.  The goroutine closes the channel after sending compareDoneMsg.
func startCompare(entries []*entry.Entry, ways int) <-chan tea.Msg {
	ch := make(chan tea.Msg)
	go func() {
		walkAndCompare(entries, ways, ch)
		ch <- compareDoneMsg{}
		close(ch)
	}()
	return ch
}

// walkAndCompare walks the tree depth-first and sends one compareResultMsg per
// file that should be compared (both/all sides present, no directories).
func walkAndCompare(entries []*entry.Entry, ways int, ch chan<- tea.Msg) {
	for _, e := range entries {
		if !e.IsDir && allSidesPresent(e, ways) {
			ch <- compareEntry(e, ways)
		}
		walkAndCompare(e.Children, ways, ch)
	}
}

func allSidesPresent(e *entry.Entry, ways int) bool {
	if ways == 2 {
		return e.Left != nil && e.Right != nil
	}
	return e.Left != nil && e.Middle != nil && e.Right != nil
}

func compareEntry(e *entry.Entry, ways int) compareResultMsg {
	msg := compareResultMsg{e: e}

	if ways == 2 {
		n, binary, err := fileops.CompareTwoFiles(*e.Left, *e.Right)
		if err != nil {
			msg.state = entry.CompareError
			return msg
		}
		if binary {
			msg.state = entry.BinaryDifferent
			return msg
		}
		msg.numDiffs = n
		if n == 0 {
			msg.state = entry.Same
		} else {
			msg.state = entry.Different
		}
		return msg
	}

	// 3-way
	lm, mr, binary, err := fileops.CompareThreeFiles(*e.Left, *e.Middle, *e.Right)
	if err != nil {
		msg.state = entry.CompareError
		return msg
	}
	if binary {
		msg.state = entry.BinaryDifferent
		return msg
	}
	msg.lmDiffs = lm
	msg.mrDiffs = mr
	if lm == 0 && mr == 0 {
		msg.state = entry.Same
	} else {
		msg.state = entry.Different
	}
	return msg
}
