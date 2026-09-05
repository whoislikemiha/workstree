# workstree

This project is **a convention** — `worktree.toml`, one committed file declaring what
a fresh worktree needs to become a working environment — plus **`workstree`, its
reference CLI**. The convention is the point; the CLI turns five or six manual steps
into one command. Any tool is welcome to implement the same convention.

`git worktree add` gives you a clean checkout — and none of what makes it _runnable_:
`node_modules`, venvs, `.env` files, build caches. Everything gitignored stays behind.
Humans rediscover this occasionally; coding agents rediscover it every single time,
burning their first effort on `npm install` archaeology or "fixing" a build that was
never broken. Every agent tool solves this privately and differently; no repo can
declare it once.

## The convention: `worktree.toml`

One TOML file at repo root, committed. Its header is the whole algorithm, so a human
or agent with no tooling can follow it:

```toml
# worktree.toml — what a fresh `git worktree add` needs to work.
# No tooling required: copy each `copy` path from the main checkout, run `setup`
# in order, then run `ready` (nonzero exit = not ready). Run `teardown` before
# removing the worktree. Shortcut: `workstree init <path>` / `workstree teardown <path>`.

setup = ["pnpm install"]
copy = [".env.local", "config/dev-certs/"]
ready = "pnpm run typecheck"
teardown = ["./scripts/worktree-runtime down -v"]

notes = """
.env.local holds the Stripe test key; regenerate with `make secrets` if missing.
"""
```

### Spec

*Target* is the fresh worktree; *source checkout* is the main working tree it was
created from (`git rev-parse --git-common-dir`). Bootstrap runs the first three
fields in this order and stops at the first failure:

| Field | Type | Semantics |
|---|---|---|
| `copy` | array of strings | Paths copied from the source checkout to the same path in the target. Relative to repo root; absolute paths and `..` escapes are invalid. Never overwrites an existing target path (re-runs are safe). A missing source is reported and skipped, not an error. No-op when target is the source checkout. |
| `setup` | array of strings | Shell commands run in the target, in order; a nonzero exit stops the bootstrap. |
| `ready` | string | Optional shell command run in the target after setup; nonzero exit = worktree not ready. |
| `teardown` | array of strings | Optional shell commands run in the target, in order, fail-fast, *before the worktree is removed*. For resources this worktree owns (Compose projects, DB volumes, dev runtimes). Not part of bootstrap; does not remove the worktree. Whoever owns the lifecycle runs it before `git worktree remove`. |
| `cache` | table | Advisory `shared`/`private` path arrays for caches that may / must not be shared across worktrees. Tools may ignore. |
| `notes` | string | Prose for humans and agents: the *why*. |

Outcome: **ready** / **failed** (a step failed) / **config error**. Nothing outside the
committed file is ever executed or copied — detection and defaults belong in
generators (`suggest`), never in the bootstrap.

### Discovery pointer

Agents don't scan repo roots for conventions they've never heard of, but they do read
`AGENTS.md` / `CLAUDE.md` at session start — exactly where they are when they decide
to create a worktree. A repo that adopts the convention carries this pointer, and
implementations that generate the file should emit the same text:

> When creating or removing git worktrees, read `worktree.toml` first. It declares
> what to copy, run, and check. `workstree init <path>` does all of it in one command
> if installed; otherwise follow the file directly. Run its `teardown`
> (`workstree teardown <path>`) before removing a worktree.

Keep it this short; it gets pasted into crowded files.

## The reference tool: `workstree`

```console
$ git worktree add ../myrepo-feature
$ workstree init ../myrepo-feature
==> copy: .env.local
==> setup 1/1: pnpm install
==> ready check: pnpm run typecheck
==> worktree ready: ../myrepo-feature
```

```console
$ go install github.com/whoislikemiha/workstree@latest
# or, linux/macos, no sudo, installs to ~/.local/bin:
$ curl -fsSL https://raw.githubusercontent.com/whoislikemiha/workstree/main/install.sh | sh
```

```
workstree                  # bootstrap the current directory's worktree
workstree init <path>      # bootstrap the worktree at <path>
workstree teardown [path]  # run teardown commands, but do not remove the worktree
workstree check [path]     # validate worktree.toml without executing
workstree suggest [path]   # inspect the repo, print a draft worktree.toml
workstree suggest --write  # ...save it and add the discovery pointer to AGENTS.md/CLAUDE.md
                           # (refuses to overwrite worktree.toml; --no-agent-docs to skip the pointer)
```

Exit codes: `0` ready/teardown complete · `1` a step failed · `2` usage or config
error. Config is read from the target, falling back to the source checkout.

## Adopting it in a repo

1. **Draft** `worktree.toml` — by hand (the example above is the entire schema) or
   with `workstree suggest --write`, which detects ecosystems from
   lockfiles (pnpm/npm/yarn/bun, uv/poetry/pip, go, cargo, bundler, composer; root and
   nested dirs), proposes copy candidates from git-ignored env-like files, and adds
   the discovery pointer to an existing `AGENTS.md`, else `CLAUDE.md`, else a new
   `AGENTS.md`.
2. **Review.** `suggest` is mechanical: drop setup for dead subprojects, add what
   detection can't see (codegen, migrations, teardown), make `ready` something that
   fails when the environment is broken, put the *why* in `notes`.
3. **Verify** in a throwaway worktree — an unexecuted config is a guess:
   ```console
   $ git worktree add /tmp/wt -b wt && workstree init /tmp/wt   # must exit 0
   $ git worktree remove --force /tmp/wt && git branch -D wt
   ```
4. **Commit** `worktree.toml` and the pointer as one diff. The copy list is usually
   secrets; a human should consciously approve it.

An agent that lands in a repo without the file should do this too, rather than invent
ad-hoc bootstrap steps in every new worktree. An optional Claude Code skill,
[`skills/workstree/SKILL.md`](skills/workstree/SKILL.md), teaches this loop and
triggers on "fresh worktree won't build" / "repo needs a `worktree.toml`":

```console
$ mkdir -p ~/.claude/skills/workstree && curl -fsSL \
    https://raw.githubusercontent.com/whoislikemiha/workstree/main/skills/workstree/SKILL.md \
    -o ~/.claude/skills/workstree/SKILL.md
```

## Known limitations

- Worktrees share repo hooks and config; repo-managed hooks (husky etc.) fire inside
  new worktrees — `setup` may need to account for that.
- Submodules and git-LFS have their own worktree quirks: best effort, test your repo.
- Copying secrets into worktrees means **deleting a worktree is secret cleanup**.

## Design

[DESIGN.md](DESIGN.md): why a declarative file, why TOML, what's out of scope.

## License

MIT
