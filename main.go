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
	args := os.Args[1:]
	if len(args) != 2 && len(args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <left> <right>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "       %s <left> <parent> <right>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	absDirs, err := absAll(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, d := range absDirs {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s: not a directory\n", d)
			os.Exit(1)
		}
	}

	var entries []*entry.Entry
	var left, middle, right string

	if len(absDirs) == 2 {
		left, right = absDirs[0], absDirs[1]
		entries, err = entry.BuildPair(left, right)
	} else {
		left, middle, right = absDirs[0], absDirs[1], absDirs[2]
		entries, err = entry.BuildTriple(left, middle, right)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(left, middle, right, entries)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func absAll(paths []string) ([]string, error) {
	out := make([]string, len(paths))
	for i, p := range paths {
		a, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		out[i] = a
	}
	return out, nil
}
