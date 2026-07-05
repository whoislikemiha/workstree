# workstree

> Worktrees that work on the first try.

`git worktree add` gives you a clean checkout — and none of what makes it *runnable*:
`node_modules`, venvs, `.env` files, build caches. Everything gitignored stays behind.
Humans rediscover this occasionally; coding agents rediscover it every single time,
burning their first effort on `npm install` archaeology or "fixing" a build that was
never broken.

**workstree** is a one-file convention plus a tiny CLI that turns a fresh worktree
into a working environment:

```console
$ git worktree add ../myrepo-feature
$ workstree init ../myrepo-feature
==> copy: .env.local
==> setup 1/2: pnpm install
==> setup 2/2: uv sync
==> ready check: pnpm run typecheck
==> worktree ready: ../myrepo-feature
```

## The convention: `worktree.toml`

One declarative file at repo root, committed:

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

The file is named after the primitive (`worktree.toml`), not this tool — any tool is
welcome to honor the same convention.

## Install

```console
$ go install github.com/whoislikemiha/workstree@latest
```

Binary releases (curl installer, Homebrew tap) coming with v0.1.

## Usage

```
workstree                # bootstrap the current directory's worktree
workstree init <path>    # bootstrap the worktree at <path>
workstree check [path]   # validate worktree.toml without executing
```

Exit codes: `0` worktree ready · `1` a copy/setup/ready step failed · `2` usage or
configuration error.

Behavior notes:

- Copy sources come from the **source checkout** (the main working tree the worktree
  was created from). Files already present in the worktree are never overwritten —
  re-running `init` is safe.
- Running `init` in the main checkout itself skips copying (nothing to carry over)
  and just runs setup/ready.
- Config is read from the target worktree first (it's committed, so it's normally
  there), falling back to the source checkout.

## For agents

Add one line to your repo's `AGENTS.md` / `CLAUDE.md`:

> After creating a git worktree, run `workstree init <path>` (config in
> `worktree.toml`) to make it a working environment.

For a repo that doesn't have `worktree.toml` yet, an agent can write one: inspect
lockfiles, README, and CI config; propose setup commands and the copy list; **verify
by creating a throwaway worktree and running the proposed setup there**; then commit
the config as a reviewable diff. A config derived from the README without
verification is worthless — always verify.

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
