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
		`eval "$(skl init zsh)"`,
		"skl cd                  change to the library root with shell integration",
		"skl pick                pick skills from across the library to load",
		"skl pick external/gstack",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Fatalf("llmTxt missing %q:\n%s", want, out)
		}
	}
}

func TestRootDoesNotExposeLegacyBoardCommand(t *testing.T) {
	if _, _, err := rootCmd.Find([]string{"board"}); err == nil {
		t.Fatalf("root command should not expose legacy board command")
	}
}
