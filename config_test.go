package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfigValid(t *testing.T) {
	p := writeConfig(t, t.TempDir(), `
setup = ["echo one", "echo two"]
copy = [".env.local", "config/certs/"]
ready = "true"
notes = "why"

[cache]
shared = ["~/.pnpm-store"]
`)
	cfg, warnings, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(cfg.Setup) != 2 || len(cfg.Copy) != 2 || cfg.Ready != "true" {
		t.Fatalf("bad parse: %+v", cfg)
	}
}

func TestLoadConfigUnknownKeyWarns(t *testing.T) {
	p := writeConfig(t, t.TempDir(), `
setup = ["true"]
setupp = ["typo"]
`)
	_, warnings, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "setupp") {
		t.Fatalf("expected unknown-key warning, got %v", warnings)
	}
}

func TestLoadConfigRejectsEscapingCopy(t *testing.T) {
	for _, entry := range []string{"../outside", "/etc/passwd", "a/../../b", "  "} {
		p := writeConfig(t, t.TempDir(), "copy = [\""+entry+"\"]\n")
		if _, _, err := LoadConfig(p); err == nil {
			t.Fatalf("copy entry %q should be rejected", entry)
		}
	}
}

func TestLoadConfigAllowsDotDotInName(t *testing.T) {
	p := writeConfig(t, t.TempDir(), `copy = ["some..file", "dir/..hidden"]`)
	if _, _, err := LoadConfig(p); err != nil {
		t.Fatalf("dot-dot in a name (not a path segment) should be fine: %v", err)
	}
}

func TestLoadConfigRejectsEmptySetup(t *testing.T) {
	p := writeConfig(t, t.TempDir(), `setup = ["true", " "]`)
	if _, _, err := LoadConfig(p); err == nil {
		t.Fatal("empty setup step should be rejected")
	}
}
