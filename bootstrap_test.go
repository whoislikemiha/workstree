package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a git repo with one commit and returns its root.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "init")
	return dir
}

// addWorktree creates a linked worktree and returns its path.
func addWorktree(t *testing.T, repo string) string {
	t.Helper()
	wt := filepath.Join(t.TempDir(), "wt")
	cmd := exec.Command("git", "-C", repo, "worktree", "add", "-q", "-b", "test-wt", wt)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("worktree add: %v\n%s", err, out)
	}
	return wt
}

func TestResolveRoots(t *testing.T) {
	repo := initRepo(t)
	wt := addWorktree(t, repo)

	target, source, err := ResolveRoots(wt)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(t, target, wt) {
		t.Fatalf("target = %s, want %s", target, wt)
	}
	if !samePath(t, source, repo) {
		t.Fatalf("source = %s, want %s", source, repo)
	}

	// In the main checkout, source == target.
	target, source, err = ResolveRoots(repo)
	if err != nil {
		t.Fatal(err)
	}
	if target != source {
		t.Fatalf("main checkout: target %s != source %s", target, source)
	}

	// From a subdirectory of the worktree, still resolves to roots.
	sub := filepath.Join(wt, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	target, _, err = ResolveRoots(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(t, target, wt) {
		t.Fatalf("from subdir: target = %s, want %s", target, wt)
	}

	// Outside any repo: error.
	if _, _, err := ResolveRoots(t.TempDir()); err == nil {
		t.Fatal("expected error outside a git repo")
	}
}

func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		t.Fatal(err)
	}
	return ra == rb
}

func TestBootstrapFullFlow(t *testing.T) {
	repo := initRepo(t)

	// Untracked carry-over: a secret file and a directory.
	if err := os.WriteFile(filepath.Join(repo, ".env.local"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	certs := filepath.Join(repo, "config", "certs")
	if err := os.MkdirAll(certs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certs, "dev.pem"), []byte("pem"), 0o644); err != nil {
		t.Fatal(err)
	}

	wt := addWorktree(t, repo)
	cfg := &Config{
		Setup: []string{"echo setup-ran > setup-marker"},
		Copy:  []string{".env.local", "config/certs", "missing-is-ok"},
		Ready: "test -f setup-marker && test -f .env.local",
	}

	var out bytes.Buffer
	b := &Bootstrap{Target: wt, Source: repo, Config: cfg, Out: &out}
	if err := b.Run(); err != nil {
		t.Fatalf("bootstrap failed: %v\noutput:\n%s", err, out.String())
	}

	secret, err := os.ReadFile(filepath.Join(wt, ".env.local"))
	if err != nil || string(secret) != "SECRET=1" {
		t.Fatalf("secret not carried over: %v", err)
	}
	if fi, _ := os.Stat(filepath.Join(wt, ".env.local")); fi.Mode().Perm() != 0o600 {
		t.Fatalf("secret file mode not preserved: %v", fi.Mode())
	}
	if _, err := os.Stat(filepath.Join(wt, "config", "certs", "dev.pem")); err != nil {
		t.Fatal("directory copy missed nested file")
	}
	if !strings.Contains(out.String(), "missing-is-ok missing in source checkout, skipped") {
		t.Fatalf("missing copy source should be reported, got:\n%s", out.String())
	}
}

func TestBootstrapSetupFailureStops(t *testing.T) {
	repo := initRepo(t)
	wt := addWorktree(t, repo)
	cfg := &Config{Setup: []string{"false", "echo never > should-not-exist"}}

	var out bytes.Buffer
	b := &Bootstrap{Target: wt, Source: repo, Config: cfg, Out: &out}
	err := b.Run()
	if err == nil {
		t.Fatal("expected setup failure")
	}
	if !strings.Contains(err.Error(), "setup step 1") {
		t.Fatalf("error should name the failing step: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(wt, "should-not-exist")); statErr == nil {
		t.Fatal("later steps must not run after a failure")
	}
}

func TestBootstrapReadyFailure(t *testing.T) {
	repo := initRepo(t)
	wt := addWorktree(t, repo)
	cfg := &Config{Ready: "false"}

	var out bytes.Buffer
	b := &Bootstrap{Target: wt, Source: repo, Config: cfg, Out: &out}
	if err := b.Run(); err == nil || !strings.Contains(err.Error(), "ready check") {
		t.Fatalf("expected ready-check failure, got %v", err)
	}
}

func TestBootstrapSkipsExistingAndSameDir(t *testing.T) {
	repo := initRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".env.local"), []byte("SOURCE"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := addWorktree(t, repo)

	// Pre-existing file in the worktree must not be overwritten.
	if err := os.WriteFile(filepath.Join(wt, ".env.local"), []byte("KEEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Copy: []string{".env.local"}}
	var out bytes.Buffer
	b := &Bootstrap{Target: wt, Source: repo, Config: cfg, Out: &out}
	if err := b.Run(); err != nil {
		t.Fatal(err)
	}
	kept, _ := os.ReadFile(filepath.Join(wt, ".env.local"))
	if string(kept) != "KEEP" {
		t.Fatal("existing file was overwritten")
	}

	// Running against the main checkout: copy is a no-op, not self-copy.
	out.Reset()
	b = &Bootstrap{Target: repo, Source: repo, Config: cfg, Out: &out}
	if err := b.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "target is the source checkout") {
		t.Fatalf("same-dir copy should be announced as no-op:\n%s", out.String())
	}
}
