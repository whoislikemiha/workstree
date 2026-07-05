package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitOut runs git in dir and returns trimmed stdout.
func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ResolveRoots returns the target worktree's root and the source checkout's
// root (the main working tree the worktree was created from). When target IS
// the main working tree, source == target.
func ResolveRoots(path string) (target, source string, err error) {
	target, err = gitOut(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("%s is not inside a git working tree: %w", path, err)
	}

	commonDir, err := gitOut(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", "", err
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(target, commonDir)
	}
	commonDir = filepath.Clean(commonDir)

	if filepath.Base(commonDir) != ".git" {
		// Bare repository or exotic layout: no source checkout to copy from.
		return target, target, nil
	}
	source = filepath.Dir(commonDir)
	return target, source, nil
}
