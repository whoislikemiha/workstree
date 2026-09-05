package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const agentInstruction = `## Worktree bootstrap

When working with git worktrees, read ` + "`worktree.toml`" + ` first. It is the repo's source of truth for files to copy, setup commands, readiness checks, and teardown/cleanup before removing a worktree. If the ` + "`workstree`" + ` CLI is available, use ` + "`workstree init/teardown`" + ` to execute those instructions.
`

// EnsureAgentInstruction adds the workstree instruction to the repo's agent docs.
// It prefers an existing AGENTS.md, then an existing CLAUDE.md, and otherwise
// creates AGENTS.md. The operation is idempotent.
func EnsureAgentInstruction(root string) (string, bool, error) {
	path := filepath.Join(root, "AGENTS.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		claude := filepath.Join(root, "CLAUDE.md")
		if _, err := os.Stat(claude); err == nil {
			path = claude
		} else if err != nil && !os.IsNotExist(err) {
			return "", false, err
		}
	} else if err != nil {
		return "", false, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", false, err
		}
		initial := "# Agent instructions\n\n" + agentInstruction
		if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
			return "", false, err
		}
		return path, true, nil
	}

	text := string(content)
	if strings.Contains(text, "read `worktree.toml`") && strings.Contains(text, "workstree init/teardown") {
		return path, false, nil
	}
	sep := "\n\n"
	if strings.HasSuffix(text, "\n\n") || text == "" {
		sep = ""
	} else if strings.HasSuffix(text, "\n") {
		sep = "\n"
	}
	updated := fmt.Sprintf("%s%s%s", text, sep, agentInstruction)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", false, err
	}
	return path, true, nil
}
