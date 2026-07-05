---
name: workstree
description: Bootstrap git worktrees into working environments and author good worktree.toml files. Use when creating a git worktree, when a fresh worktree fails to build/run (missing node_modules, .env, venv), when a repo needs a worktree.toml, or when the user mentions workstree or worktree bootstrap.
---

# workstree — bootstrap worktrees, author worktree.toml

`git worktree add` copies tracked files only. Everything gitignored — installed deps,
`.env` files, build caches — stays behind. `worktree.toml` (committed, repo root)
declares what a fresh worktree needs; `workstree` executes it.

## Using an existing worktree.toml

After creating any worktree, always:

```bash
git worktree add <path> -b <branch>
workstree init <path>     # copy carry-over files, run setup, run ready check
```

Exit `0` = ready. Exit `1` = a step failed (read the output; the failing step is
named). Exit `2` = config/usage error. Never start work in a worktree whose init
failed — you will end up debugging the environment instead of doing the task.

`workstree init` is idempotent: it never overwrites existing files, so re-running
after a failure is safe.

## Authoring worktree.toml for a repo that lacks one

Follow this loop — do not skip the verify step:

1. **Draft**: `workstree suggest --write` (refuses to overwrite an existing config).
   It detects ecosystems from lockfiles (root + nested dirs like `sidecar/`,
   `src-tauri/`, `packages/*`) and proposes copy candidates from git-ignored env
   files that actually exist.
2. **Prune and improve the draft** — `suggest` is mechanical; you have judgment:
   - Remove setup entries for spikes, experiments, and abandoned prototypes
     (e.g. a `spike/` dir with its own lockfile that nobody builds).
   - Add what detection can't see: codegen steps (`prisma generate`, protobuf),
     DB migrations for dev, `direnv allow`, disabling repo-managed git hooks that
     misbehave in worktrees.
   - Check the copy list against the repo's docs: is there a `.env` the README
     says to create? A certs dir? Add entries even if the file doesn't exist in
     this checkout — missing sources are skipped gracefully, and the entry
     documents the need.
   - Write a `ready` check that proves the environment works and runs fast:
     a typecheck or build, not the full test suite.
   - Use `notes` for the why: where secrets come from, how to regenerate them,
     anything a future agent would otherwise have to rediscover.
3. **Verify on a throwaway worktree** — mandatory, a config derived from reading
   is worthless until executed:
   ```bash
   git worktree add /tmp/workstree-verify -b workstree-verify
   workstree init /tmp/workstree-verify        # must exit 0
   git worktree remove --force /tmp/workstree-verify
   git branch -D workstree-verify
   ```
   If init fails, fix the config (not the worktree) and re-verify.
4. **Commit** `worktree.toml` as a reviewable diff. Flag the copy list in your
   report — it is usually secrets, and humans should consciously approve what
   gets replicated into every future worktree.

## Quality bar for entries

- **Setup**: deterministic and lockfile-frozen (`npm ci`, not `npm install`;
  `--frozen-lockfile` variants). Fail-fast order: dependencies before codegen
  before anything else. Each command must be safe to run in a brand-new checkout.
- **Copy**: minimal. Only untracked files the build/run actually needs. Never add
  tracked files (they come with the checkout) or rebuildable artifacts
  (`node_modules`, `dist` — setup rebuilds those).
- **Ready**: must fail when the environment is broken. `echo ok` proves nothing;
  a build or typecheck that needs the installed deps proves everything.

## Pitfalls

- Repo-managed hooks (husky etc.) fire inside worktrees and may fail before setup
  has run — account for them in `setup` if needed.
- Multiple lockfiles for one ecosystem (e.g. stray `package-lock.json` next to
  `pnpm-lock.yaml`): suggest picks by priority, but verify which one the team
  actually uses.
- Node resolution walks up: a nested package may build against the root
  `node_modules` even without its own install. If its lockfile exists, install it
  anyway — ancestry resolution is a fragile accident, not a contract.
- Deleting a worktree deletes copied secrets with it — that's a feature; don't
  "back them up" elsewhere.
