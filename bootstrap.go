package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Bootstrap executes the convention against the worktree at path:
// copy carry-over files from the source checkout, run setup, run ready.
type Bootstrap struct {
	Target string
	Source string
	Config *Config
	Out    io.Writer
}

func (b *Bootstrap) Run() error {
	if err := b.copyAll(); err != nil {
		return err
	}
	for i, cmd := range b.Config.Setup {
		fmt.Fprintf(b.Out, "==> setup %d/%d: %s\n", i+1, len(b.Config.Setup), cmd)
		if err := b.sh(cmd); err != nil {
			return fmt.Errorf("setup step %d (%s) failed: %w", i+1, cmd, err)
		}
	}
	if b.Config.Ready != "" {
		fmt.Fprintf(b.Out, "==> ready check: %s\n", b.Config.Ready)
		if err := b.sh(b.Config.Ready); err != nil {
			return fmt.Errorf("ready check (%s) failed: %w", b.Config.Ready, err)
		}
	}
	fmt.Fprintf(b.Out, "==> worktree ready: %s\n", b.Target)
	return nil
}

func (b *Bootstrap) copyAll() error {
	if len(b.Config.Copy) == 0 {
		return nil
	}
	if sameDir(b.Source, b.Target) {
		fmt.Fprintf(b.Out, "==> copy: target is the source checkout, nothing to carry over\n")
		return nil
	}
	for _, entry := range b.Config.Copy {
		rel := filepath.Clean(strings.TrimSpace(entry))
		src := filepath.Join(b.Source, rel)
		dst := filepath.Join(b.Target, rel)

		info, err := os.Stat(src)
		if os.IsNotExist(err) {
			fmt.Fprintf(b.Out, "==> copy: %s missing in source checkout, skipped\n", rel)
			continue
		}
		if err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
		if _, err := os.Stat(dst); err == nil {
			fmt.Fprintf(b.Out, "==> copy: %s already exists in worktree, skipped\n", rel)
			continue
		}
		if info.IsDir() {
			err = copyDir(src, dst)
		} else {
			err = copyFile(src, dst, info.Mode())
		}
		if err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
		fmt.Fprintf(b.Out, "==> copy: %s\n", rel)
	}
	return nil
}

// sh runs one config command through the shell in the target worktree.
func (b *Bootstrap) sh(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = b.Target
	cmd.Stdout = b.Out
	cmd.Stderr = b.Out
	return cmd.Run()
}

func sameDir(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return ra == rb
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, targetPath)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode())
	})
}
