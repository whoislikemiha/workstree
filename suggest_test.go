package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, root string, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSuggestDetectsEcosystems(t *testing.T) {
	repo := initRepo(t)
	touch(t, repo, "pnpm-lock.yaml", "")
	touch(t, repo, "package-lock.json", "") // must lose to pnpm within js group
	touch(t, repo, "go.mod", "module x")

	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(s.Setup, "\n")
	if !strings.Contains(joined, "pnpm install") {
		t.Fatalf("pnpm not detected: %v", s.Setup)
	}
	if strings.Contains(joined, "npm ci") {
		t.Fatalf("npm suggested despite pnpm lockfile: %v", s.Setup)
	}
	if !strings.Contains(joined, "go mod download") {
		t.Fatalf("go not detected: %v", s.Setup)
	}
}

func TestSuggestNestedEcosystems(t *testing.T) {
	repo := initRepo(t)
	touch(t, repo, "package-lock.json", "")
	touch(t, repo, "sidecar/package-lock.json", "")
	touch(t, repo, "src-tauri/Cargo.lock", "")
	touch(t, repo, "src-tauri/Cargo.toml", "")
	// Below scanMaxDepth: must not be detected.
	touch(t, repo, "a/b/c/package-lock.json", "")
	// Inside skiplisted/hidden dirs: must not be detected.
	touch(t, repo, "node_modules/x/package-lock.json", "")
	touch(t, repo, ".claude/worktrees/y/package-lock.json", "")

	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"npm ci",
		"cd sidecar && npm ci",
		"cd src-tauri && cargo fetch",
	}
	if len(s.Setup) != len(want) {
		t.Fatalf("setup = %v, want %v", s.Setup, want)
	}
	for i := range want {
		if s.Setup[i] != want[i] {
			t.Fatalf("setup[%d] = %q, want %q (all: %v)", i, s.Setup[i], want[i], s.Setup)
		}
	}
	// Ready comes from the root ecosystem only.
	if s.Ready != "npm run build --if-present" {
		t.Fatalf("ready = %q", s.Ready)
	}
}

func TestSuggestCargoWorkspaceMembersIgnored(t *testing.T) {
	repo := initRepo(t)
	touch(t, repo, "Cargo.lock", "")
	touch(t, repo, "Cargo.toml", "")
	// Workspace member: Cargo.toml but no lock — must not get its own fetch.
	touch(t, repo, "crates/member/Cargo.toml", "")

	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Setup) != 1 || s.Setup[0] != "cargo fetch" {
		t.Fatalf("setup = %v, want [cargo fetch]", s.Setup)
	}
}

func TestSuggestEnvFilesOnlyIgnoredOnes(t *testing.T) {
	repo := initRepo(t)
	touch(t, repo, ".gitignore", ".env\n.env.local\n")
	touch(t, repo, ".env", "SECRET=1")        // ignored + exists → suggested
	touch(t, repo, ".env.example", "SAMPLE=") // exists but NOT ignored → excluded
	// .env.local is ignored but doesn't exist → excluded

	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Copy) != 1 || s.Copy[0] != ".env" {
		t.Fatalf("copy = %v, want [.env]", s.Copy)
	}
}

func TestSuggestEmptyRepo(t *testing.T) {
	repo := initRepo(t)
	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Setup) != 0 || len(s.Copy) != 0 {
		t.Fatalf("expected empty suggestion, got %+v", s)
	}
	out := s.Render()
	if !strings.Contains(out, "no ecosystem detected") {
		t.Fatalf("empty render should include fill-in hints:\n%s", out)
	}
}

func TestSuggestRenderIsValidConfig(t *testing.T) {
	repo := initRepo(t)
	touch(t, repo, ".gitignore", ".env\n")
	touch(t, repo, ".env", "S=1")
	touch(t, repo, "package-lock.json", "")

	s, err := Suggest(repo)
	if err != nil {
		t.Fatal(err)
	}
	// The rendered draft must round-trip through our own parser.
	dir := t.TempDir()
	p := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(p, []byte(s.Render()), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("rendered draft does not parse: %v\n%s", err, s.Render())
	}
	if len(warnings) != 0 {
		t.Fatalf("rendered draft has warnings: %v", warnings)
	}
	if len(cfg.Setup) != 1 || cfg.Setup[0] != "npm ci" || cfg.Copy[0] != ".env" {
		t.Fatalf("round-trip mismatch: %+v", cfg)
	}
}
