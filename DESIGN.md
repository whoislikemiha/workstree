# workstree — Design

> Worktrees that work on the first try.

Status: design accepted, not yet implemented. This document seeds the future public
`workstree` repo. **Ships before legwork** — it is small, useful standalone, and a
dress rehearsal for the shared delivery pipeline (Go binary, goreleaser, curl install).

## Problem

Git worktrees became the default isolation primitive for agentic coding tools
(claude-squad, vibe-kanban, Claude Code's native worktree mode, legwork), but a fresh
worktree is not a working environment: everything gitignored stays behind —
`node_modules`, venvs, `.env`, build caches, generated code. Every tool solves
carry-over privately and partially (devcontainers only if you buy containers, direnv
only env vars, per-tool ad-hoc setup hooks). There is no convention a repo can declare
**once** that all tools and agents honor. Agents dropped into a pristine worktree burn
their first tokens rediscovering `npm install` — or "fix" the broken build creatively.

## What it is

Three parts:

1. **A convention**: one declarative file at repo root describing what a fresh worktree
   needs to become a working environment.
2. **A tiny reference CLI** (~a few hundred lines) that executes the convention.
3. **A discovery pointer** in `AGENTS.md` / `CLAUDE.md` — part of the convention, since
   a file nobody is told about is not a convention. An optional skill covers *writing*
   the file for repos that lack it.

## The convention: `worktree.toml`

**Decision: the file is named after the primitive (`worktree.toml`), the tool after the
promise (`workstree`).** Files named after concepts outlive tools; other tools can read
the same convention without "adopting workstree's file." Lives at repo root, committed.

Fields (v1):

```toml
# worktree.toml — what a fresh `git worktree add` needs to actually work.
# Executed by `workstree` (https://github.com/.../workstree) or any compatible tool.

# setup lists shell commands to run in the new worktree, in order.
setup = [
  "pnpm install",
  "uv sync",
]

# teardown lists shell commands to run explicitly before removing the worktree.
# Use it to stop/delete resources owned by this worktree only.
teardown = [
  "./scripts/worktree-runtime down -v",
]

# copy lists untracked files copied from the source checkout into the new worktree.
# Auditable on purpose: this list is usually secrets.
copy = [
  ".env.local",
  "config/dev-certs/",
]

# ready is a fast smoke check; nonzero exit = worktree NOT ready.
ready = "pnpm run typecheck --noEmit"

# Optional hints; advisory, tool may ignore.
[cache]
shared = ["~/.pnpm-store"]   # safe to share across worktrees
private = []                  # must NOT be shared (e.g. cargo target/)

# Prose for humans and agents: the *why*.
notes = """
.env.local holds the Stripe test key; regenerate with `make secrets` if missing.
Husky hooks are repo-shared; setup disables them in worktrees via core.hooksPath.
"""
```

**Why TOML** (by elimination, not fashion): must be declarative and introspectable —
tools render the copy list in UIs, validate before running, and auditing "these files
get copied into every worktree" matters because it's usually credentials. A shell
script is opaque and scary to auto-run. JSON forbids comments (disqualifying — prose
annotations are half the value). YAML is footgun-rich for four keys. TOML has comments,
no surprises, and manifest prior art (`Cargo.toml`, `pyproject.toml`). Escape hatch
preserved: `setup` entries are shell strings. If adopters revolt, format is a one-day
change — not worth defending to the death.

## The CLI

```
workstree init <path>     # bootstrap the worktree at <path>
workstree                 # bare: bootstrap the current worktree
workstree teardown [path] # run cleanup commands; does not remove the worktree
workstree check           # validate worktree.toml without executing
```

Behavior of `init`: locate source checkout (the worktree's parent repo), copy the
`copy` list, run `setup` commands in order, run `ready` if present. Exit 0 = ready.
Nonzero = not ready, with the failing step on stderr. No flags needed for the happy
path. Verb-object symmetry with `git worktree` is deliberate — it sits next to the
primitive it fixes:

```
git worktree add ../myrepo-feature && workstree init ../myrepo-feature
```

Behavior of `teardown`: locate the target and source checkout, read the same
`worktree.toml`, then run only the `teardown` commands in the target, fail-fast. It is
explicit by design: workstree does not hook into `git worktree remove`, stop arbitrary
processes, or remove the worktree itself. Lifecycle owners can call it before removal.

Embeddable by design: other tools (legwork, claude-squad, anyone) either shell out to
the binary or vendor the logic.

## Distribution: a line in AGENTS.md / CLAUDE.md

The adoption mechanism. Agents don't scan repo roots for conventions they've never
heard of, but they do read `AGENTS.md` / `CLAUDE.md` at session start in the main
checkout — exactly where they are when they decide to create a worktree. Repos add:

> When creating or removing git worktrees, read `worktree.toml` first. It declares what
> to copy, run, and check. `workstree init <path>` does all of it in one command if
> installed; otherwise follow the file directly. Run its `teardown`
> (`workstree teardown <path>`) before removing a worktree.

Two deliberate choices. **File first, CLI second**: an agent without the binary still
has a complete instruction, because the file's own header comment spells out the
manual steps; the CLI only collapses them into one command. **Short**: the line gets
pasted into crowded files and must survive that; anything longer belongs in the file's
comments, which the agent reads next anyway. The wording is spec'd in the README so
third-party generators emit the same text. `workstree suggest --write` is
the reference implementation, preferring an existing `AGENTS.md`, then `CLAUDE.md`,
otherwise creating `AGENTS.md`.

Consuming the convention needs nothing else — no skill, no install. The skill exists
only for the authoring moment, and even that is mostly covered by the review checklist
`suggest` puts at the top of the draft, at the point of use.

## The generative direction (writing the file)

Authoring for a cold repo, as taught by the draft header and the optional skill:

1. Read-only inspection: lockfiles, README, CI config, existing dev docs.
2. Propose setup commands + copy list.
3. **Verify — not optional**: create a throwaway worktree, run the proposed setup
   there, confirm `ready` passes. A config derived from the README without verification
   is worthless and moves the flailing one level up.
4. Write `worktree.toml` and the agent-doc instruction; commit as a reviewable diff.

Trigger model (as used by legwork, generalizable): lazily, on first worktree need in a
repo lacking the file ("needs-bootstrap"); re-triggered on setup failure — **configs
rot** (npm→pnpm migrations), and the failure log becomes context for the repair run.
Humans who already know the answer just hand-write the four lines.

## Known limitations (documented honestly in README)

- Worktrees share repo hooks and config; repo hooks (husky etc.) fire inside worker
  worktrees — `setup` may need to disable them.
- Submodules and LFS have worktree quirks: best effort, test your repo.
- Same branch can't be checked out in two worktrees (git rule; namespaced branches
  avoid it).
- Copying secrets into worktrees means deleting worktrees is secret cleanup — callers
  owning worktree lifecycle should treat it that way.

## Implementation

- **Go**, minimal deps: cobra (CLI), BurntSushi/toml, stdlib exec. Single static binary.
- Release: goreleaser → GitHub Releases; `curl -fsSL .../install.sh | sh` (detects
  OS/arch, installs to `~/.local/bin`, no sudo); brew tap; `go install`.
- Unix + WSL. Windows-native out of scope v1.

## Name

`workstree` = "a worktree that works." Rejected `workingtree`: collides with git's own
term of art ("the working tree"), would fight git docs for meaning and search.
Availability verified 2026-07-05: free on GitHub, npm, PyPI, crates.io, Homebrew; no
product collision. **Claim registry names at repo creation.**

## Out of scope

- Owning containers/VMs as a runtime platform (that's devcontainers' job). Repos may
  still declare shell commands that start/stop per-worktree runtimes.
- Worktree *creation/management* (wtp, worktrunk, et al. exist; we only make the
  resulting worktree work).
- Any coupling to legwork: independent projects, clean seam, no shared naming scheme —
  a suite framing would couple their fates for no benefit.
