package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAgentInstructionCreatesAGENTSMD(t *testing.T) {
	root := t.TempDir()

	path, changed, err := EnsureAgentInstruction(root)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected new AGENTS.md to be written")
	}
	if filepath.Base(path) != "AGENTS.md" {
		t.Fatalf("path = %s, want AGENTS.md", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "read `worktree.toml`") || !strings.Contains(text, "workstree init <path>") {
		t.Fatalf("agent instruction should point to worktree.toml as source of truth:\n%s", text)
	}

	againPath, againChanged, err := EnsureAgentInstruction(root)
	if err != nil {
		t.Fatal(err)
	}
	if againPath != path {
		t.Fatalf("second path = %s, want %s", againPath, path)
	}
	if againChanged {
		t.Fatal("second call should be idempotent")
	}
	againContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(againContent) != text {
		t.Fatal("idempotent call changed AGENTS.md")
	}
}

func TestEnsureAgentInstructionPrefersExistingClaudeMD(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# Project guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, changed, err := EnsureAgentInstruction(root)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected CLAUDE.md to be updated")
	}
	if path != claude {
		t.Fatalf("path = %s, want %s", path, claude)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "# Project guidance") || !strings.Contains(text, "read `worktree.toml`") {
		t.Fatalf("existing guidance was not preserved with instruction appended:\n%s", text)
	}
}
