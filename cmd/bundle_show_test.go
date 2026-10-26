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
	writeBundleShow(&out, []string{"dev", "ops"}, &library.Catalog{Skills: skills, Bundles: bundles}, st)

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

func TestWriteBundleShowMarksRecursiveBundle(t *testing.T) {
	cat := &library.Catalog{
		Skills: []library.Skill{{ID: "sf/sf-build"}, {ID: "sf/ref/mp-tdd"}},
		Bundles: map[string][]string{
			"sf":     {"sf/ref/mp-tdd", "sf/sf-build"},
			"sf/ref": {"sf/ref/mp-tdd"},
		},
		Recursive: map[string]bool{"sf": true},
	}
	st := &state.State{Version: 1, Loaded: map[string]state.LoadEntry{}}

	var out bytes.Buffer
	writeBundleShow(&out, []string{"sf", "sf/ref"}, cat, st)

	got := out.String()
	for _, want := range []string{"sf  (recursive: includes sf/ref)", "sf/ref/mp-tdd", "sf/sf-build"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "recursive") != 1 {
		t.Fatalf("only the recursive bundle should be marked:\n%s", got)
	}
}
