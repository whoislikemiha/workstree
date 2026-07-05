# Working on workstree

workstree is a convention (`worktree.toml`) plus its reference CLI. **The
convention is the product** — the README's spec section (terms, fields, execution
semantics) is the contract; code changes that alter behavior must update the spec,
and spec changes are breaking changes to third-party implementors. Rationale and
rejected ideas: [DESIGN.md](DESIGN.md).

## Layout

Single root `package main`, a few hundred lines by design: `config.go` (schema +
validation), `git.go` (root resolution), `bootstrap.go` (copy/setup/ready),
`suggest.go` (draft generation), `main.go` (cobra wiring).
`skills/workstree/SKILL.md` is the agent-facing skill.

## Verify before claiming done

```bash
gofmt -l . && go vet ./... && go test ./... -count=1
```

Tests use real git repos/worktrees in temp dirs. For changes to `suggest` or
`bootstrap`, also verify against a real project (e.g. run `workstree suggest` on a
repo with lockfiles and check the draft makes sense).

## Hard rules

- **`init` never guesses.** Detection/defaults belong in `suggest` (generator
  time); what executes is exactly what the committed file declares.
- **Copy entries are a security surface**: relative-only, no `..` escapes —
  validation stays strict. Never overwrite existing files (idempotent re-runs).
- **Exit codes are contract**: 0 ready / 1 step failed / 2 usage-or-config.
- **`suggest` stays dumb and deterministic** — no README parsing, no LLM calls;
  judgment belongs to the reviewing agent (that's what the skill teaches).
- The config file is `worktree.toml` (named after the primitive, not the tool).

## Conventions

- Commit messages: plain, no Co-Authored-By or other trailers.
- Do not tag releases or publish packages unless explicitly asked.
- Keep it small: this tool wins by being boring and embeddable.
