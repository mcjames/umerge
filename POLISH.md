# umerge — polish & UX findings

Split out from `TODO.md` 2026-07-23 (that file was getting long) to keep
the versioned feature roadmap focused on priority-numbered work. This
file collects **UX/trust/discoverability findings** — things a normal
developer or sysadmin would likely notice or be surprised by — that
don't map cleanly onto a numbered priority yet. Not triaged into a
release yet; capture first, prioritize later.

Origin: a 2026-07-23 "30,000 foot" review done ahead of a friend trying
umerge for outside feedback, followed by a code-verified audit (not
speculation — every claim below was checked against the actual source,
with citations). Framing at the time: the friend's test happens *before*
Selection/3-way-merge (the two things gating 1.0) exist, so their
feedback is much more likely to hit the items below than either of those
two features.

---

## Discoverability — the biggest first-impression risk

**No in-app help overlay.** Every serious TUI tool (vim, less, tmux,
htop) has a `?`-or-similar key that shows keybindings *from inside the
running program*. umerge has no such thing: `Update`'s key switch
(`internal/ui/app.go`) has no `?` case and no help-view state anywhere on
`Model`. The only in-app reference is the abbreviated status-bar hint
line:

```go
status := fmt.Sprintf(" %d/%d%s  q quit  ←→/enter collapse  ↑↓/jk move  PgUp/PgDn scroll  a/b/d copy/del", ...)
```

— a 5-item abbreviation that doesn't even mention `c`, `h`, `H`, or `r`.
Everything else (full keybinding list) only lives in `--help` and
`umerge.1`, neither visible mid-session. Someone who launches umerge cold
via `git difftool -d` has no way to discover `h`/`H` exist without having
read the docs first. This is the kind of gap that decides whether a tool
clicks in the first minute or gets abandoned as confusing — probably the
single highest-leverage, lowest-cost fix in this whole file.

**No search / jump-to-filename.** Grepped `internal/ui` for anything
search/find/query/filter/jump-related — nothing. No `/` key case, no
incremental-search state. On a toy tree this doesn't matter; on a real
repo with hundreds of files, not being able to jump to a name by typing
it is a real "this doesn't scale" moment, and it's the one thing every
comparable TUI (less, vim, fzf, ranger, yazi) already has. Arguably more
valuable day-to-day than Priority 3b's focus mode, and considerably
cheaper to build — no async tri-state tracking needed, just an
incremental filter/jump over the tree that's already flattened in
`m.flat`.

---

## Trust & safety

**No delete confirmation, by deliberate decision — worth revisiting on
purpose, not just resting on precedent.** Confirmed in code: the `"d"`
case in `Update` (`app.go`) calls `deleteEntry` directly, guarded only by
read-only mode and "still comparing," never a confirmation prompt.
`ops.go`'s `deleteEntry` loops over every present side
(`Left`/`Middle`/`Right`) and calls `fileops.Delete` (`os.RemoveAll`)
immediately — this is side-count-agnostic, so it'll behave identically
once 3-way delete exists. This matches the Python original and Unix
`rm -Rf` semantics on purpose (see `TODO.md` Priority 1's "Copy/delete
semantics" section and `feedback_umerge_scope_boundaries` — default to
exact Unix behavior). Still worth a deliberate gut-check for a tool whose
whole positioning is "trust it on a real, messy tree": a first-time
sysadmin hitting instant recursive delete with zero prompt is a real
"wait, it just did what?" moment, not a hypothetical one.

**Permission-denied directories are silently swallowed as empty.**
`readSorted` (`internal/entry/entry.go`) calls `os.ReadDir` and on *any*
error — including `EACCES` — just `return nil`. No warning, log line, or
marker surfaces to the user; the directory renders exactly like a
directory with zero contents. A tester pointing umerge at anything with
restricted permissions will see "nothing here" with no way to tell it's
actually "couldn't read this." Worth at minimum a distinguishable render
state (e.g. reusing the existing `CompareError`/red styling, or a new
one) rather than silent equivalence with "genuinely empty."

**Ordinary symlinks are silently flattened to plain files, never
followed.** Verified: `os.DirEntry.IsDir()` (which `BuildTree` uses
exclusively to decide `isDir`) returns `false` for a symlink regardless
of what it points to — a symlink to a directory and a symlink to a file
both come back `IsDir()==false`. So any symlink in a compared tree is
treated as a leaf, never expanded/recursed into, with no error or marker
at build time. If later content-compared, `fileops.filesEqual` opens it
fine but fails on read ("is a directory"), surfacing as a `CompareError`
red row — not a crash, but a confusing one with no explanation of *why*.
(This is a distinct, more general problem from the `git difftool -d`
specific symlink hazard already documented and solved in `TODO.md`
Priority 6 via `--read-only` — that one is about *safety*; this one is
about *correctness/visibility* of ordinary symlinked content anywhere in
a compared tree.)

---

## Distribution

**`go install` / build-from-source only — no packaged distribution.**
Confirmed: no `Makefile`, no goreleaser config, no Homebrew formula.
`.github/workflows/ci.yml` only runs `go build`/`go vet`/`go test` on
push/PR — it doesn't build or publish release binaries. `README.md`'s
Installation section offers exactly two paths: `go install
github.com/mcjames/umerge@latest`, or clone + `go build .`. For a
"normal sysadmin" audience specifically (not necessarily Go developers),
this is real friction before they ever get to try the tool at all.

**Planned, not a gap (decided 2026-07-23): Homebrew packaging, deliberately
post-1.0.** User's intent is to package for Homebrew *after* the 1.0
release, not before — so this stays a known, intentionally-deferred
follow-up rather than something blocking the release. Likely needs a
goreleaser config (or equivalent) to produce tagged release binaries
before a Homebrew formula can point at them; worth sequencing that
groundwork right after 1.0 ships, alongside or ahead of the formula
itself.

---

## Minor CLI polish

**Bad-path error message doesn't distinguish "doesn't exist" from "not a
directory."** `main.go`'s arg validation:

```go
info, err := os.Stat(d)
if err != nil || !info.IsDir() {
    fmt.Fprintf(os.Stderr, "%s: %s: not a directory\n", prog, d)
    os.Exit(1)
}
```

A nonexistent path and an existing-but-non-directory path both print the
identical `umerge: /path/x: not a directory` — the real underlying
`os.Stat` error (which would correctly say "no such file or directory")
is discarded. Also no `flag.Usage()` reminder on this specific error path
(only wrong *argument count* gets the usage reminder today). Small, but
a first-time user mistyping a path gets a slightly misleading message
with no nudge toward correct usage.
