package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is the convention: named after the primitive it configures,
// not after this tool, so any tool can honor it.
const ConfigFileName = "worktree.toml"

// Config is the worktree.toml schema.
type Config struct {
	// Setup commands run in the new worktree, in order.
	Setup []string `toml:"setup"`
	// Copy lists untracked files/dirs copied from the source checkout.
	// Auditable on purpose: this list is usually secrets.
	Copy []string `toml:"copy"`
	// Ready is an optional smoke check; nonzero exit = worktree not ready.
	Ready string `toml:"ready"`
	// Cache hints are advisory; tools may ignore them.
	Cache CacheHints `toml:"cache"`
	// Notes carry the why, for humans and agents.
	Notes string `toml:"notes"`
}

type CacheHints struct {
	Shared  []string `toml:"shared"`
	Private []string `toml:"private"`
}

// LoadConfig parses and validates the worktree.toml at path.
func LoadConfig(path string) (*Config, []string, error) {
	var cfg Config
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var warnings []string
	for _, key := range meta.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("unknown key %q (ignored)", key.String()))
	}
	if err := cfg.validate(); err != nil {
		return nil, warnings, err
	}
	return &cfg, warnings, nil
}

// FindConfig locates worktree.toml, preferring the target worktree (the file
// is committed, so it is normally present there) and falling back to the
// source checkout.
func FindConfig(target, source string) (string, error) {
	for _, dir := range []string{target, source} {
		p := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no %s found in %s or %s", ConfigFileName, target, source)
}

func (c *Config) validate() error {
	for i, cmd := range c.Setup {
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("setup[%d] is empty", i)
		}
	}
	for _, entry := range c.Copy {
		if err := validateCopyEntry(entry); err != nil {
			return err
		}
	}
	return nil
}

// validateCopyEntry rejects paths that could read or write outside the
// checkout roots: absolute paths and anything escaping via "..".
func validateCopyEntry(entry string) error {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" {
		return fmt.Errorf("copy entry is empty")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("copy entry %q must be relative", entry)
	}
	clean := filepath.Clean(trimmed)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("copy entry %q escapes the checkout", entry)
	}
	return nil
}
