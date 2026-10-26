package cmd

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"skl/internal/library"
	"skl/internal/state"
)

func TestLoadedSummaryText(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0 skills loaded"},
		{1, "1 skill loaded"},
		{3, "3 skills loaded"},
	}
	for _, c := range cases {
		if got := loadedSummaryText(c.n); got != c.want {
			t.Errorf("loadedSummaryText(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out)
}

func TestPrintBundlesShowsLoadedTotal(t *testing.T) {
	bundles := map[string][]string{
		"alpha": {"s1", "s2"},
		"beta":  {"s3"},
	}
	st := &state.State{Loaded: map[string]state.LoadEntry{
		"s1": {Bundles: []string{"alpha"}},
		"s3": {Bundles: []string{"beta"}},
	}}
	out := captureStdout(t, func() {
		if err := printBundles(&library.Catalog{Bundles: bundles}, st); err != nil {
			t.Fatalf("printBundles: %v", err)
		}
	})
	if !strings.Contains(out, "2 skills loaded") {
		t.Fatalf("expected footer %q in output:\n%s", "2 skills loaded", out)
	}
	if strings.Contains(out, "INCLUDES") {
		t.Fatalf("libraries without recursive bundles keep the original columns:\n%s", out)
	}
}

func TestPrintBundlesShowsRecursiveInclusion(t *testing.T) {
	cat := &library.Catalog{
		Bundles: map[string][]string{
			"dev":         {"dev/alpha"},
			"sf":          {"sf/ref/deep/mp-grill", "sf/ref/mp-tdd", "sf/sf-build"},
			"sf/ref":      {"sf/ref/mp-tdd"},
			"sf/ref/deep": {"sf/ref/deep/mp-grill"},
		},
		Recursive: map[string]bool{"sf": true},
	}
	st := &state.State{Loaded: map[string]state.LoadEntry{}}
	out := captureStdout(t, func() {
		if err := printBundles(cat, st); err != nil {
			t.Fatalf("printBundles: %v", err)
		}
	})

	rows := map[string][]string{}
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			rows[fields[0]] = fields
		}
	}
	if got := rows["BUNDLE"]; !reflect.DeepEqual(got, []string{"BUNDLE", "SKILLS", "STATUS", "INCLUDES"}) {
		t.Fatalf("header = %#v\n%s", got, out)
	}
	if got := rows["sf"]; !reflect.DeepEqual(got, []string{"sf", "3", "—", "sf/ref,", "sf/ref/deep"}) {
		t.Fatalf("sf row = %#v\n%s", got, out)
	}
	if got := rows["sf/ref"]; !reflect.DeepEqual(got, []string{"sf/ref", "1", "—"}) {
		t.Fatalf("sf/ref row = %#v\n%s", got, out)
	}
}

func TestPrintSkillsShowsLoadedTotal(t *testing.T) {
	skills := []library.Skill{
		{ID: "s1"},
		{ID: "s2"},
		{ID: "s3"},
	}
	bundles := map[string][]string{"alpha": {"s1", "s2"}}
	st := &state.State{Loaded: map[string]state.LoadEntry{
		"s1": {Bundles: []string{"alpha"}},
		"s3": {},
	}}
	out := captureStdout(t, func() {
		if err := printSkills(skills, bundles, st); err != nil {
			t.Fatalf("printSkills: %v", err)
		}
	})
	if !strings.Contains(out, "2 skills loaded") {
		t.Fatalf("expected footer %q in output:\n%s", "2 skills loaded", out)
	}
}
