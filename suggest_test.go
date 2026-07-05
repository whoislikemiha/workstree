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
