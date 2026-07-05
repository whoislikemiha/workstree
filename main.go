package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// version is set by goreleaser via ldflags.
var version = "dev"

// Exit codes are part of the contract:
//
//	0 = worktree ready
//	1 = a copy/setup/ready step failed
//	2 = usage or configuration error
const (
	exitStepFailed = 1
	exitUsage      = 2
)

func main() {
	root := &cobra.Command{
		Use:     "workstree",
		Short:   "Worktrees that work on the first try",
		Long:    "workstree executes a repo's worktree.toml to turn a fresh `git worktree add` into a working environment: carry-over files, setup commands, ready check.",
		Version: version,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(".")
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	initCmd := &cobra.Command{
		Use:   "init <path>",
		Short: "Bootstrap the worktree at <path> (default: current directory)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			return runInit(path)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	checkCmd := &cobra.Command{
		Use:   "check [path]",
		Short: "Validate worktree.toml without executing anything",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			return runCheck(path)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(initCmd, checkCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "workstree: %v\n", err)
		if _, ok := err.(*stepError); ok {
			os.Exit(exitStepFailed)
		}
		os.Exit(exitUsage)
	}
}

// stepError marks failures of copy/setup/ready execution (exit 1), as
// opposed to usage/config problems (exit 2).
type stepError struct{ err error }

func (e *stepError) Error() string { return e.err.Error() }
func (e *stepError) Unwrap() error { return e.err }

func runInit(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	target, source, err := ResolveRoots(abs)
	if err != nil {
		return err
	}
	cfgPath, err := FindConfig(target, source)
	if err != nil {
		return err
	}
	cfg, warnings, err := LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "workstree: warning: %s: %s\n", cfgPath, w)
	}

	b := &Bootstrap{Target: target, Source: source, Config: cfg, Out: os.Stdout}
	if err := b.Run(); err != nil {
		return &stepError{err}
	}
	return nil
}

func runCheck(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	target, source, err := ResolveRoots(abs)
	if err != nil {
		return err
	}
	cfgPath, err := FindConfig(target, source)
	if err != nil {
		return err
	}
	cfg, warnings, err := LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "workstree: warning: %s: %s\n", cfgPath, w)
	}

	fmt.Printf("config:      %s\n", cfgPath)
	fmt.Printf("setup steps: %d\n", len(cfg.Setup))
	for _, s := range cfg.Setup {
		fmt.Printf("  - %s\n", s)
	}
	fmt.Printf("copy list:   %d\n", len(cfg.Copy))
	for _, c := range cfg.Copy {
		fmt.Printf("  - %s\n", c)
	}
	if cfg.Ready != "" {
		fmt.Printf("ready check: %s\n", cfg.Ready)
	} else {
		fmt.Printf("ready check: none\n")
	}
	fmt.Printf("valid\n")
	return nil
}
