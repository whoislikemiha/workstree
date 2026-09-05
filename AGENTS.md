# Working on workstree

workstree is a convention (`worktree.toml`) plus its reference CLI. **The convention
is the product.** The README's Spec section (fields, execution semantics, discovery
pointer) is the contract: code that changes behavior must change the spec, and spec
changes break third-party implementors. Rationale and rejected ideas: [DESIGN.md](DESIGN.md).

## Layout

Single root `package main`, a few hundred lines by design:

| File | Owns |
|---|---|
| `config.go` | schema + validation |
| `git.go` | target / source-checkout root resolution |
| `bootstrap.go` | copy / setup / ready / teardown execution |
| `suggest.go` | draft generation, including the draft's header comment |
| `agent_docs.go` | the discovery pointer written to `AGENTS.md` / `CLAUDE.md` |
| `main.go` | cobra wiring, exit codes |
| `skills/workstree/SKILL.md` | optional Claude Code skill; authoring only |

## Verify before claiming done

```bash
gofmt -l . && go vet ./... && go test ./... -count=1
```

Tests use real git repos and worktrees in temp dirs. For changes to `suggest`,
`bootstrap`, or `agent_docs`, also run it on a real repo: `workstree suggest --write
--agent-docs` in a throwaway clone, then `workstree init` on a worktree of it, and read
what it printed.

## Hard rules

- **`init` never guesses.** Detection and defaults live in `suggest`; what executes is
  exactly what the committed file declares.
- **Copy entries are a security surface.** Relative-only, no `..` escapes, validation
  stays strict. Never overwrite existing files; re-runs must be idempotent.
- **Exit codes are contract.** 0 ready / 1 step failed / 2 usage-or-config.
- **`suggest` stays dumb and deterministic.** No README parsing, no LLM calls.
  Judgment belongs to the reviewing agent.
- **The file works without the CLI.** Its four-line header comment spells out
  copy → setup → ready → teardown so any agent can follow it by hand. The README
  example and the `suggest` draft carry the same header (test-enforced). The CLI is a
  shortcut, never a requirement.
- **The pointer stays short and file-first.** A few sentences: read the file, the CLI
  is a shortcut if installed, run teardown before removal. It gets pasted into crowded
  `AGENTS.md` files. The README's Discovery pointer is the source; `agent_docs.go`
  copies it (test-enforced so they can't drift).
- The config file is `worktree.toml`, named after the git primitive rather than the
  tool, so other tools can read the convention without "adopting workstree's file".
  Don't rename it to match the CLI.
