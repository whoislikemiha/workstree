package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// detector maps an ecosystem marker file to suggested setup/ready commands.
// Order matters: first match per group wins (e.g. pnpm-lock beats package-lock).
type detector struct {
	marker string
	group  string // one suggestion per group (js, py, ...)
	setup  string
	ready  string
}

var detectors = []detector{
	{"pnpm-lock.yaml", "js", "pnpm install --frozen-lockfile", "pnpm run build --if-present"},
	{"bun.lock", "js", "bun install --frozen-lockfile", ""},
	{"bun.lockb", "js", "bun install --frozen-lockfile", ""},
	{"yarn.lock", "js", "yarn install --frozen-lockfile", ""},
	{"package-lock.json", "js", "npm ci", "npm run build --if-present"},
	{"uv.lock", "py", "uv sync", ""},
	{"poetry.lock", "py", "poetry install", ""},
	{"requirements.txt", "py", "python3 -m venv .venv && .venv/bin/pip install -r requirements.txt", ""},
	{"go.mod", "go", "go mod download", "go build ./..."},
	// Cargo.lock, not Cargo.toml: workspace members have a Cargo.toml but no
	// lock; only the workspace root (or a standalone crate) should be fetched.
	{"Cargo.lock", "rust", "cargo fetch", "cargo check"},
	{"Gemfile.lock", "ruby", "bundle install", ""},
	{"composer.lock", "php", "composer install", ""},
}

// scanMaxDepth: how deep below the repo root to look for nested ecosystems
// (sidecars, src-tauri, packages/*). Depth 0 is the root itself.
const scanMaxDepth = 2

// skipDirs are never descended into during detection.
var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"out": true, "target": true, ".venv": true, "venv": true,
	"coverage": true, "tmp": true,
}

// envGlobs are untracked-file patterns commonly needed by a working checkout.
var envGlobs = []string{".env", ".env.*", ".envrc", "*.local"}

// Suggestion is a draft config derived from repo inspection.
type Suggestion struct {
	Setup []string
	Copy  []string
	Ready string
}

// Suggest inspects the source checkout and proposes a worktree.toml.
// Detection covers the root and nested ecosystems up to scanMaxDepth
// (sidecars, src-tauri, packages/*); nested setup commands are wrapped in
// `cd <dir> && ...` and ordered root-first.
func Suggest(root string) (*Suggestion, error) {
	s := &Suggestion{}

	dirs, err := scanDirs(root)
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		seenGroup := map[string]bool{}
		for _, d := range detectors {
			if seenGroup[d.group] {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, dir, d.marker)); err == nil {
				cmd := d.setup
				if dir != "." {
					cmd = fmt.Sprintf("cd %s && %s", dir, d.setup)
				}
				s.Setup = append(s.Setup, cmd)
				// Ready check only from the root ecosystem: it runs at repo
				// root and should reflect the primary build.
				if dir == "." && s.Ready == "" && d.ready != "" {
					s.Ready = d.ready
				}
				seenGroup[d.group] = true
			}
		}
	}

	copies, err := ignoredEnvFiles(root)
	if err != nil {
		return nil, err
	}
	s.Copy = copies
	return s, nil
}

// scanDirs returns "." plus non-hidden, non-skiplisted directories up to
// scanMaxDepth below root, in deterministic root-first order.
func scanDirs(root string) ([]string, error) {
	dirs := []string{"."}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			return filepath.SkipDir
		}
		if strings.Count(rel, string(filepath.Separator))+1 > scanMaxDepth {
			return filepath.SkipDir
		}
		dirs = append(dirs, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs[1:])
	return dirs, nil
}

// ignoredEnvFiles returns env-like files at repo root that exist AND are
// git-ignored (tracked files travel with the checkout; only ignored ones
// need carry-over).
func ignoredEnvFiles(root string) ([]string, error) {
	var candidates []string
	seen := map[string]bool{}
	for _, glob := range envGlobs {
		matches, err := filepath.Glob(filepath.Join(root, glob))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || info.IsDir() {
				continue
			}
			rel, err := filepath.Rel(root, m)
			if err != nil || seen[rel] {
				continue
			}
			seen[rel] = true
			candidates = append(candidates, rel)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// Single git call: check-ignore prints the subset that is ignored.
	args := append([]string{"-C", root, "check-ignore", "--"}, candidates...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// Exit 1 = none ignored; anything else is a real error.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("git check-ignore: %w", err)
	}
	var ignored []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			ignored = append(ignored, line)
		}
	}
	sort.Strings(ignored)
	return ignored, nil
}

// Render produces a draft worktree.toml.
func (s *Suggestion) Render() string {
	var b strings.Builder
	b.WriteString("# worktree.toml — generated by `workstree suggest`. DRAFT: verify before\n")
	b.WriteString("# committing (create a throwaway worktree, run `workstree init` on it).\n\n")

	b.WriteString("# Commands run in a fresh worktree, in order.\n")
	if len(s.Setup) > 0 {
		b.WriteString("setup = [\n")
		for _, cmd := range s.Setup {
			fmt.Fprintf(&b, "  %q,\n", cmd)
		}
		b.WriteString("]\n")
	} else {
		b.WriteString("# setup = [\"<no ecosystem detected — fill in>\"]\n")
	}

	b.WriteString("\n# Untracked files copied from the source checkout (found git-ignored here).\n")
	b.WriteString("# Review carefully: this list is usually secrets.\n")
	if len(s.Copy) > 0 {
		b.WriteString("copy = [\n")
		for _, c := range s.Copy {
			fmt.Fprintf(&b, "  %q,\n", c)
		}
		b.WriteString("]\n")
	} else {
		b.WriteString("# copy = []\n")
	}

	b.WriteString("\n# Smoke check; nonzero exit = worktree NOT ready.\n")
	if s.Ready != "" {
		fmt.Fprintf(&b, "ready = %q\n", s.Ready)
	} else {
		b.WriteString("# ready = \"<a fast build/typecheck/test command>\"\n")
	}
	return b.String()
}
