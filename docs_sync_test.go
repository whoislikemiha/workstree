package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var wsRE = regexp.MustCompile(`\s+`)

// squash collapses whitespace so text can be compared across line-wrapping and
// markdown blockquote prefixes.
func squash(s string) string {
	s = strings.ReplaceAll(s, "\n> ", " ")
	s = strings.ReplaceAll(s, "\n>", " ")
	return strings.TrimSpace(wsRE.ReplaceAllString(s, " "))
}

func readme(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	return squash(string(b))
}

// The discovery pointer is spec'd in the README; the CLI must emit the same text.
func TestAgentInstructionMatchesREADME(t *testing.T) {
	body := squash(strings.TrimPrefix(agentInstruction, "## Worktrees\n\n"))
	if !strings.Contains(readme(t), body) {
		t.Fatalf("README Discovery pointer does not contain agentInstruction:\n%s", body)
	}
}

// The file header is the no-CLI algorithm; README example and suggest draft must agree.
func TestSuggestHeaderMatchesREADME(t *testing.T) {
	lines := strings.SplitN((&Suggestion{}).Render(), "\n", 5)
	header := squash(strings.Join(lines[:4], "\n"))
	if !strings.Contains(readme(t), header) {
		t.Fatalf("README example does not contain the suggest draft header:\n%s", header)
	}
}
