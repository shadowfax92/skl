package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelpIntroStaysBrief(t *testing.T) {
	if strings.Contains(rootCmd.Long, "mkdir") || strings.Contains(rootCmd.Long, "mv ") {
		t.Fatalf("root help should not include filesystem choreography:\n%s", rootCmd.Long)
	}
	if strings.Count(rootCmd.Long, "\n") > 5 {
		t.Fatalf("root help intro is too long:\n%s", rootCmd.Long)
	}
	if !strings.Contains(rootCmd.Long, "Folder-based skill loadouts") {
		t.Fatalf("root help should explain the tool in one concise line:\n%s", rootCmd.Long)
	}
}

func TestLLMTxtExplainsLibraryLayout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := llmTxt()
	if err != nil {
		t.Fatalf("llmTxt: %v", err)
	}

	library := filepath.Join(home, ".config", "skl", "library")
	required := []string{
		"SKL LLM GUIDE",
		"Library: " + library,
		"legacy unbundled skills",
		"external/<repo>/<skill>/SKILL.md",
		"skl load external/gstack",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Fatalf("llmTxt missing %q:\n%s", want, out)
		}
	}
}
