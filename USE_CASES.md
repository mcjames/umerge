# umerge — use cases (raw material, not README prose)

Captured 2026-08-02 while discussing the by-hand README rewrite planned
for the 1.0 push. The problem this is answering: Beyond Compare/Araxis
Merge are GUI tools people mostly stumble into via "my company bought a
license," so almost nobody has independently discovered *why*
directory-level diff/merge is a distinct category from `git diff` or a
single-file tool like `delta`/`difftastic`. This file is a starter set of
concrete scenarios to pick from, cut, or rewrite when drafting the
README's use-case section — not a finished draft itself.

The throughline across the first four: **arbitrary directories with no
shared history.** That's the actual category boundary that makes `git
diff`/lazygit inapplicable and justifies the tool's existence independent
of git. The fifth case is deliberately different — git-adjacent, not
git-independent — and is kept separate for that reason.

---

## 1. Comparing vendor/third-party code drops

You get a new SDK tarball, a firmware blob, or a fork of a library
someone hand-patched — with no shared git history against what you're
currently vendoring. `git diff` has nothing to anchor to; you need two
arbitrary trees compared file-by-file.

This is the one with a real, lived example already on record: the user
professionally relied on Araxis Merge for exactly this — comparing vendor
code drops, a directory comparison with nothing to do with git history
(see `project_umerge` memory, "Validated real-world use case"). Probably
the strongest, most credible example — use a concrete version of it
rather than a hypothetical.

## 2. Auditing config/environment drift

`/etc` on server A vs. server B, two Ansible-managed hosts that should be
identical but aren't, or your dotfiles vs. a "golden" reference. No git
involved at all — just "why are these two directories different."

## 3. Reviewing generated/build output after a toolchain or dependency bump

Comparing `dist/` (or equivalent) before and after upgrading a
compiler/bundler/dependency, to eyeball what actually changed across
dozens of files at once. A spatial tree view catches "wait, why did this
unrelated file change" faster than paging through a linear patch stream.

## 4. Reconciling independent edits with no version control

Two people each got a copy of a directory — not a git branch, genuinely
just a folder — worked on it separately, and now need a 3-way merge
against the original to combine both sets of changes. This is the case
git literally cannot help with, since there's no repo at all.

## 5. A nicer bulk-review UI on top of git itself

`git difftool -d` for reviewing a commit/branch with many changed files:
a spatial map of what changed, plus one-keystroke access to the merge
tool per file, instead of git's own linear diff stream. Git-adjacent on
purpose — "you already use git but want better bulk review," not
"git can't help you here at all" — which is why it's kept separate from
1–4 rather than folded in with them.

---

## Tooling note (unrelated to the list above, captured while discussing it)

For the planned asciinema recordings: `vhs` (Charm, same ecosystem as
Bubble Tea/Lipgloss which umerge already uses) is worth a look alongside
asciinema. It's a scriptable `.tape` file that renders a terminal session
to GIF/asciinema-format output, so the demo stays reproducible and easy
to re-record when the UI changes, instead of manually re-doing a live
recording each time. Not a strong push either way, just worth five
minutes of investigation.
