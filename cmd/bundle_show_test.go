package cmd

import (
	"bytes"
	"strings"
	"testing"

	"skl/internal/library"
	"skl/internal/state"
)

func TestWriteBundleShowRendersAllSelectedBundles(t *testing.T) {
	bundles := map[string][]string{
		"dev": {"dev/alpha"},
		"ops": {"ops/deploy"},
	}
	st := &state.State{
		Version: 1,
		Loaded: map[string]state.LoadEntry{
			"dev/alpha": {DirName: "alpha"},
		},
	}
	skills := []library.Skill{
		{ID: "dev/alpha"},
		{ID: "ops/deploy", External: true, Repo: "gstack"},
	}

	var out bytes.Buffer
	writeBundleShow(&out, []string{"dev", "ops"}, bundles, st, skills)

	got := out.String()
	for _, want := range []string{
		"dev",
		"dev/alpha",
		"loaded",
		"ops",
		"ops/deploy",
		"ext: gstack",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "SKILL") != 2 {
		t.Fatalf("expected one table per selected bundle:\n%s", got)
	}
}
