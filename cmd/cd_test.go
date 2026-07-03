package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCDCommandPrintsAndPreparesLibraryRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := *cdCmd
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("cd RunE: %v", err)
	}

	want := filepath.Join(home, ".config", "skl", "library")
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("cd output mismatch\ngot:  %s\nwant: %s", got, want)
	}
	if info, err := os.Stat(want); err != nil {
		t.Fatalf("cd should prepare library root: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("library root should be a directory: %s", want)
	}
}

func TestCDCommandRejectsArgs(t *testing.T) {
	cmd := *cdCmd
	if err := cmd.Args(&cmd, []string{"extra"}); err == nil {
		t.Fatalf("cd should reject extra args")
	}
}
