---
name: workstree
description: Author and verify worktree.toml, the committed file declaring what a fresh git worktree needs (files to copy, setup commands, ready check, teardown). Use when a repo lacks worktree.toml, when a fresh worktree fails to build/run (missing node_modules, .env, venv), or when the user mentions workstree.
---

# workstree — author `worktree.toml`

`git worktree add` copies tracked files only; everything gitignored stays behind.
`worktree.toml` at repo root declares what a fresh worktree needs. **If the repo
already has one, just follow it** — `workstree init <path>` if installed, otherwise
copy/setup/ready by hand as its header comment says — and stop reading. Never start
work in a worktree whose init failed; you'll debug the environment instead of the task.

If the repo lacks one:

1. **Draft**: `workstree suggest --write --agent-docs`, or hand-write it from the
   README example. `suggest` detects lockfiles (root and nested dirs), proposes copy
   candidates from git-ignored env files, and adds the discovery pointer to
   `AGENTS.md` / `CLAUDE.md`. Refuses to overwrite an existing file.
2. **Review** — `suggest` is mechanical; you have judgment:
   - Drop setup entries for spikes and abandoned subprojects. If two lockfiles
     coexist for one ecosystem, check which one the team actually uses.
   - Add what detection can't see: codegen (`prisma generate`, protobuf), dev
     migrations, `direnv allow`, repo hooks (husky) that fire before setup has run,
     and `teardown` for per-worktree containers/volumes — prefer a repo script over
     hardcoded docker commands so it derives the same project name/ports as setup.
   - `setup`: lockfile-frozen (`npm ci`, `--frozen-lockfile`), deps before codegen.
     Install nested packages with their own lockfile even if root resolution would
     accidentally work.
   - `copy`: only untracked files the build needs — never tracked files or rebuildable
     artifacts. Add entries the docs say to create even if absent in this checkout;
     missing sources are skipped and the entry documents the need.
   - `ready`: must fail when the environment is broken — a build or typecheck, not
     `echo ok` and not the full test suite.
   - `notes`: the why — where secrets come from, how to regenerate them.
   - Delete the DRAFT block from the header once reviewed.
3. **Verify** on a throwaway worktree. Mandatory — a config that was never executed
   is a guess:
   ```bash
   git worktree add /tmp/wt -b wt && workstree init /tmp/wt   # must exit 0
   git worktree remove --force /tmp/wt && git branch -D wt
   ```
   If it fails, fix the config, not the worktree, and re-verify.
4. **Commit** `worktree.toml` and the pointer as one diff. Flag the copy list in your
   report — it's usually secrets, and a human should consciously approve it.
