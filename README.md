# workstree

> Worktrees that work on the first try.

`git worktree add` gives you a clean checkout — and none of what makes it _runnable_:
`node_modules`, venvs, `.env` files, build caches. Everything gitignored stays behind.
Humans rediscover this occasionally; coding agents rediscover it every single time,
burning their first effort on `npm install` archaeology or "fixing" a build that was
never broken. Every agent tool solves this privately and differently; no repo can
declare it once.

This project is **a convention** — `worktree.toml`, one committed file declaring what
a fresh worktree needs to become a working environment — plus **`workstree`, its
reference CLI**. The convention is the point; the tool just executes it. Any tool is
welcome to implement the same convention.

## The convention: `worktree.toml`

One declarative TOML file at repo root, committed to the repo:

```toml
# What a fresh `git worktree add` needs to actually work.

# Commands run in the new worktree, in order.
setup = ["pnpm install"]

# Untracked files/dirs copied from the source checkout.
# Auditable on purpose: this list is usually secrets.
copy = [".env.local", "config/dev-certs/"]

# Optional smoke check; nonzero exit = worktree NOT ready.
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

Binary releases (curl installer, Homebrew tap) coming with v0.1.

### Usage

```
workstree                  # bootstrap the current directory's worktree
workstree init <path>      # bootstrap the worktree at <path>
workstree check [path]     # validate worktree.toml without executing
workstree suggest [path]   # inspect the repo, print a draft worktree.toml
workstree suggest --write  # ...and save it (refuses to overwrite)
```

Exit codes: `0` worktree ready · `1` a copy/setup/ready step failed · `2` usage or
configuration error.

Config is read from the target first (the file is committed, so it's normally there),
falling back to the source checkout.

## Who writes `worktree.toml`?

`workstree suggest` drafts it: it detects the ecosystem from lockfiles
(pnpm/npm/yarn/bun, uv/poetry/pip, go, cargo, bundler, composer) and proposes copy
candidates from git-ignored env-like files that actually exist in your checkout
(`.env`, `.env.*`, `.envrc`, `*.local`). The draft is deliberately **not** executed on
trust: review it, verify it, commit it.

## For agents

Add one line to your repo's `AGENTS.md` / `CLAUDE.md`:

> After creating a git worktree, run `workstree init <path>` (config in
> `worktree.toml`) to make it a working environment.

For a repo without `worktree.toml`: run `workstree suggest --write`, then **verify —
create a throwaway worktree and run `workstree init` on it** — adjust until the ready
check passes, and commit the config as a reviewable diff. Never commit an unverified
draft.

### Skill

This repo ships a skill — [`skills/workstree/SKILL.md`](skills/workstree/SKILL.md) —
teaching agents to bootstrap worktrees and to *author good `worktree.toml` files*
(the prune-improve-verify loop, quality bar for entries, known pitfalls). For Claude
Code, copy it into your project or user skills directory:

```console
$ mkdir -p ~/.claude/skills/workstree
$ curl -fsSL https://raw.githubusercontent.com/whoislikemiha/workstree/main/skills/workstree/SKILL.md \
    -o ~/.claude/skills/workstree/SKILL.md
```

## Known limitations

- Worktrees share repo hooks and config; repo-managed hooks (husky etc.) fire inside
  new worktrees — your `setup` may want to account for that.
- Submodules and git-LFS have their own worktree quirks: best effort, test your repo.
- Copying secrets into worktrees means **deleting a worktree is secret cleanup** —
  treat it that way.

## Design

See [DESIGN.md](DESIGN.md) for the full rationale (why a declarative file, why TOML,
what's out of scope).

## License

MIT
