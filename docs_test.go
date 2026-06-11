package towerctl_test

import (
	"os"
	"strings"
	"testing"
)

func TestRootReadmeMentionsHermesAgentReadme(t *testing.T) {
	root, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md missing: %v", err)
	}
	if !strings.Contains(string(root), "docs/hermes-agent/README.md") {
		t.Fatalf("root README must mention docs/hermes-agent/README.md")
	}
}

func TestHermesAgentReadmeMentionsPromptAndTools(t *testing.T) {
	readme, err := os.ReadFile("docs/hermes-agent/README.md")
	if err != nil {
		t.Fatalf("docs/hermes-agent/README.md missing: %v", err)
	}
	for _, want := range []string{"tools.yaml", "system-prompt.md", "towerctl serve-mcp"} {
		if !strings.Contains(string(readme), want) {
			t.Fatalf("Hermes README missing %s", want)
		}
	}
}
