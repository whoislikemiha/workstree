# workstree

This project is **a convention** — `worktree.toml`, one committed file declaring what
a fresh worktree needs to become a working environment plus **`workstree`, its
reference CLI**. The convention is the point, the tool just executes it.  
Any tool is welcome to implement the same convention.

## Problem
`git worktree add` gives you a clean checkout — and none of what makes it _runnable_:
`node_modules`, venvs, `.env` files, build caches. Everything gitignored stays behind.
Humans rediscover this occasionally; coding agents rediscover it every single time,
burning their first effort on `npm install` archaeology or "fixing" a build that was
never broken. Every agent tool solves this privately and differently; no repo can
declare it once.
## The convention: `worktree.toml`

One declarative TOML file at repo root, committed to the repo:

```toml
# What a fresh `git worktree add` needs to actually work.

# setup lists shell commands to run in the new worktree, in order.
setup = ["pnpm install"]

# teardown lists shell commands to run before removing the worktree.
# Use it to stop/delete resources owned by this worktree only.
teardown = ["./scripts/worktree-runtime down -v"]

# copy lists untracked files/dirs copied from the source checkout.
# Tracked files already come with the worktree; this list is usually secrets.
copy = [".env.local", "config/dev-certs/"]

# ready is a smoke check; nonzero exit = worktree NOT ready.
ready = "pnpm run typecheck"

notes = """
.env.local holds the Stripe test key; regenerate with `make secrets` if missing.
"""
```

### Terms

- **Target**: the fresh worktree being bootstrapped.
- **Source checkout**: the main working tree the worktree was created from (resolved
  via `git rev-parse --git-common-dir`).

### Fields

| Field | Type | Meaning |
|---|---|---|
| `setup` | array of strings | Shell commands run **in the target**, in order, fail-fast: a nonzero exit stops the bootstrap. |
| `teardown` | array of strings | Optional shell commands run **in the target** by `workstree teardown`, in order, fail-fast. Use for resources owned by this worktree, such as per-worktree Docker Compose projects or DB volumes. |
| `copy` | array of strings | Paths (files or directories) copied **from the source checkout into the target**. Relative to repo root; absolute paths and `..` escapes are invalid. These are untracked files — tracked files travel with the checkout already. |
| `ready` | string | Optional shell command run in the target after setup. Nonzero exit means the worktree is **not** ready. |
| `cache` | table | Advisory hints (`shared`/`private` path arrays) about caches that may / must not be shared across worktrees. Tools may ignore. |
| `notes` | string | Prose for humans and agents: the *why* behind the entries. |

### Execution semantics

An implementation bootstraps a target by doing exactly this, in this order:

1. **Copy**: for each `copy` entry, copy source-checkout path → same path in target.
   Never overwrite an existing target path (re-runs must be safe). A missing source
   path is reported and skipped, not an error. When target *is* the source checkout,
   copying is a no-op.
2. **Setup**: run each `setup` command in the target, in order; stop on first failure.
3. **Ready**: run `ready` if present; nonzero exit = bootstrap failed.

Outcome: **ready** (all steps passed) / **failed** (a step failed) / **config error**.
Nothing outside the committed file is ever executed or copied — detection, defaults,
or other magic belong in generators (see `suggest`), never in the bootstrap.

`teardown` is separate from bootstrap. `workstree teardown <path>` runs only the
declared teardown commands in the target worktree; it does not remove the worktree or
run copy/setup/ready. Callers that own worktree lifecycle should run it before
`git worktree remove` when the repo declares cleanup for per-worktree resources.

The file is named after the primitive (`worktree.toml`), not after any tool, so the
convention can outlive its implementations. Anything a worktree needs carried over or
run belongs in it.

## The reference tool: `workstree`

```console
$ git worktree add ../myrepo-feature
$ workstree init ../myrepo-feature
==> copy: .env.local
==> setup 1/1: pnpm install
==> ready check: pnpm run typecheck
==> worktree ready: ../myrepo-feature
```

### Install

```console
$ go install github.com/whoislikemiha/workstree@latest
```

Or the curl installer (linux/macos, amd64/arm64, no sudo — installs to `~/.local/bin`):

```console
$ curl -fsSL https://raw.githubusercontent.com/whoislikemiha/workstree/main/install.sh | sh
```

### Usage

```
workstree                  # bootstrap the current directory's worktree
workstree init <path>      # bootstrap the worktree at <path>
workstree teardown [path]  # run teardown commands, but do not remove the worktree
workstree check [path]     # validate worktree.toml without executing
workstree suggest [path]   # inspect the repo, print a draft worktree.toml
workstree suggest --write  # ...and save it (refuses to overwrite)
workstree suggest --write --agent-docs
                          # also add the AGENTS.md/CLAUDE.md instruction
```

Exit codes: `0` worktree ready/teardown complete · `1` a copy/setup/ready/teardown
step failed · `2` usage or configuration error.

Config is read from the target first (the file is committed, so it's normally there),
falling back to the source checkout.

## Onboarding a repo

workstree is adopted per repo. Humans and agents should not need prior knowledge of the
convention once the repo has committed a commented `worktree.toml` and a short agent
instruction pointing at it.

### For maintainers

1. **Install workstree** locally.
2. **Draft the config and agent instruction**:
   ```console
   $ workstree suggest --write --agent-docs
   ```
3. **Review and edit `worktree.toml`**. `suggest` detects lockfiles and existing
   git-ignored env-like files; humans still decide what setup, secrets, ready checks,
   and teardown commands are actually correct. `--agent-docs` adds a short
   "read `worktree.toml`" pointer to an existing `AGENTS.md`, then an existing
   `CLAUDE.md`, or creates `AGENTS.md` if neither exists.
4. **Verify in a throwaway worktree**:
   ```console
   $ git worktree add /tmp/workstree-verify -b workstree-verify
   $ workstree init /tmp/workstree-verify
   $ workstree teardown /tmp/workstree-verify   # if teardown is declared
   $ git worktree remove --force /tmp/workstree-verify
   $ git branch -D workstree-verify
   ```
5. **Commit the convention**:
   - `worktree.toml`
   - an `AGENTS.md` / `CLAUDE.md` instruction for agents and tools

If you do not use `--agent-docs`, add this one-liner to `AGENTS.md` / `CLAUDE.md`:

> When working with git worktrees, read `worktree.toml` first. It is the repo's source
> of truth for files to copy, setup commands, readiness checks, and teardown/cleanup
> before removing a worktree. If the `workstree` CLI is available, use
> `workstree init/teardown` to execute those instructions.

### For agents and tools

If the repo has `worktree.toml`, read it instead of guessing. It is intentionally
commented so it can be followed by humans/tools even without the workstree CLI. If the
CLI is available, use it to execute the convention:

```console
$ git worktree add ../myrepo-feature -b feature
$ workstree init ../myrepo-feature
```

If `worktree.toml` declares teardown, run it before deleting the worktree:

```console
$ workstree teardown ../myrepo-feature
$ git worktree remove ../myrepo-feature
```

If the repo lacks `worktree.toml`, do not invent ad-hoc bootstrap steps in every new
worktree. Draft the convention with `workstree suggest --write --agent-docs`, verify it
in a throwaway worktree, then commit it as a reviewable diff.

### Skills are optional

This repo ships a skill — [`skills/workstree/SKILL.md`](skills/workstree/SKILL.md) —
teaching skill-aware agents to bootstrap worktrees and to *author good
`worktree.toml` files* (the prune-improve-verify loop, quality bar for entries, known
pitfalls). It is an acceleration path, not a requirement: the committed
`worktree.toml`, the CLI, and the `AGENTS.md` / `CLAUDE.md` line are the actual
cross-tool contract.

For Claude Code, copy the skill into your project or user skills directory:

```console
$ mkdir -p ~/.claude/skills/workstree
$ curl -fsSL https://raw.githubusercontent.com/whoislikemiha/workstree/main/skills/workstree/SKILL.md \
    -o ~/.claude/skills/workstree/SKILL.md
```

## Who writes `worktree.toml`?

Usually the repo maintainer or the first agent that notices a repo lacks the convention.
`workstree suggest` drafts it: it detects the ecosystem from lockfiles
(pnpm/npm/yarn/bun, uv/poetry/pip, go, cargo, bundler, composer) and proposes copy
candidates from git-ignored env-like files that actually exist in your checkout
(`.env`, `.env.*`, `.envrc`, `*.local`). The draft is deliberately **not** executed on
trust: review it, verify it, commit it. `workstree suggest --write --agent-docs` also
adds the discovery pointer so future agents know to read the committed convention.

## Known limitations

- Worktrees share repo hooks and config; repo-managed hooks (husky etc.) fire inside
  new worktrees — your `setup` may want to account for that.
- Submodules and git-LFS have their own worktree quirks: best effort, test your repo.
- Copying secrets into worktrees means **deleting a worktree is secret cleanup** —
  treat it that way.
- `workstree teardown` is explicit: workstree does not hook into `git worktree remove`
  or automatically stop long-lived runtimes unless the caller invokes it.

## Design

See [DESIGN.md](DESIGN.md) for the full rationale (why a declarative file, why TOML,
what's out of scope).

## License

MIT
