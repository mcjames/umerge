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
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <left-directory> <right-directory>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	left, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	right, err := filepath.Abs(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, dir := range []string{left, right} {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s: not a directory\n", dir)
			os.Exit(1)
		}
	}

	entries, err := entry.BuildPair(left, right)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(left, right, entries)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
