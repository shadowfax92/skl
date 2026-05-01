package cmd

import (
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
