package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// agentInstruction is the discovery pointer the convention relies on. The file is
// the source of truth; the CLI is a shortcut. Keep it short: it gets pasted into
// crowded AGENTS.md files.
const agentInstruction = "## Worktrees\n\n" +
	"When creating or removing git worktrees, read `worktree.toml` first. It declares what to copy, " +
	"run, and check. `workstree init <path>` does all of it in one command if installed; otherwise " +
	"follow the file directly. Run its `teardown` (`workstree teardown <path>`) before removing a worktree.\n"

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
	if strings.Contains(text, "read `worktree.toml`") {
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
