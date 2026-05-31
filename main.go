package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"umerge/internal/entry"
	"umerge/internal/ui"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <directory>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "%s: not a directory\n", root)
		os.Exit(1)
	}

	entries, err := entry.BuildTree(root, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", root, err)
		os.Exit(1)
	}

	m := ui.New(root, entries)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
