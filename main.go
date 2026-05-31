package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	flag "github.com/spf13/pflag"
	"umerge/internal/entry"
	"umerge/internal/ui"
)

const version = "0.1.0"

func main() {
	prog := filepath.Base(os.Args[0])

	helpFlag := flag.BoolP("help", "h", false, "display this help and exit")
	versionFlag := flag.BoolP("version", "V", false, "print version and exit")
	mergeFlag := flag.StringP("merge", "m", "vim", "external diff/merge `tool`: vim or emacs")

	flag.CommandLine.SortFlags = false
	flag.Usage = func() {
		shortUsage(os.Stderr, prog)
		fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", prog)
	}

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		// pflag already printed the error; Usage() is called automatically.
		os.Exit(1)
	}

	if *helpFlag {
		printHelp(os.Stdout, prog)
		os.Exit(0)
	}

	if *versionFlag {
		printVersion(os.Stdout)
		os.Exit(0)
	}

	mergeTool := *mergeFlag
	if mergeTool != "vim" && mergeTool != "emacs" {
		fmt.Fprintf(os.Stderr, "%s: invalid merge tool %q: must be vim or emacs\n", prog, mergeTool)
		flag.Usage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) != 2 && len(args) != 3 {
		flag.Usage()
		os.Exit(1)
	}

	absDirs, err := absAll(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(1)
	}
	for _, d := range absDirs {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s: %s: not a directory\n", prog, d)
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
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(1)
	}

	m := ui.New(left, middle, right, entries, mergeTool)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(1)
	}
}

// ── flag helpers ──────────────────────────────────────────────────────────────

func shortUsage(w io.Writer, prog string) {
	fmt.Fprintf(w, "Usage: %s [OPTION]... LEFT RIGHT\n", prog)
	fmt.Fprintf(w, "       %s [OPTION]... LEFT PARENT RIGHT\n", prog)
}

func printHelp(w io.Writer, prog string) {
	shortUsage(w, prog)
	fmt.Fprintf(w, "Compare two or three directory trees interactively.\n")
	fmt.Fprintf(w, "\nOptions:\n")
	fmt.Fprint(w, flag.CommandLine.FlagUsages())
	fmt.Fprintf(w, "\nArguments:\n")
	fmt.Fprintf(w, "  LEFT, RIGHT           directories to compare (two-way)\n")
	fmt.Fprintf(w, "  LEFT, PARENT, RIGHT   directories to compare (three-way);\n")
	fmt.Fprintf(w, "                        PARENT is the common ancestor,\n")
	fmt.Fprintf(w, "                        displayed in the center column\n")
	fmt.Fprintf(w, "\nKey bindings:\n")
	fmt.Fprintf(w, "  Up/Down, j/k          move cursor\n")
	fmt.Fprintf(w, "  Page Up/Page Down      scroll one page\n")
	fmt.Fprintf(w, "  Left/Right            collapse or expand directory\n")
	fmt.Fprintf(w, "  Enter                 open file in diff/merge tool; toggle directory\n")
	fmt.Fprintf(w, "  q, Ctrl-C             quit\n")
	fmt.Fprintf(w, "\nSee umerge(1) for full documentation.\n")
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "umerge %s\n", version)
	fmt.Fprintf(w, "Copyright (C) 2026 Michael C. James. All rights reserved.\n")
	fmt.Fprintf(w, "This software is distributed under the MIT License.\n")
}

// ── utilities ─────────────────────────────────────────────────────────────────

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
