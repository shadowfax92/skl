package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/picker"
	"skl/internal/state"
)

func TestPickCommandLoadsOnlySelectedBundleSkills(t *testing.T) {
	setupHome(t)

	root, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkillTree(t, filepath.Join(root, "dev", "alpha"), "alpha")
	writeSkillTree(t, filepath.Join(root, "dev", "beta"), "beta")

	cmd := *pickCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, []string{"dev", "alpha"}); err != nil {
		t.Fatalf("pick RunE: %v", err)
	}

	liveRoot, err := live.LivePath()
	if err != nil {
		t.Fatalf("LivePath: %v", err)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "alpha")); err != nil {
		t.Fatalf("selected skill should be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "beta")); !os.IsNotExist(err) {
		t.Fatalf("unselected skill should not be copied, stat err: %v", err)
	}

	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	st, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	entry, ok := st.Loaded["dev/alpha"]
	if !ok {
		t.Fatalf("state missing selected skill: %#v", st.Loaded)
	}
	if !reflect.DeepEqual(entry.Bundles, []string{"dev"}) {
		t.Fatalf("selected skill bundles = %#v, want dev", entry.Bundles)
	}
	if _, ok := st.Loaded["dev/beta"]; ok {
		t.Fatalf("state should not include unselected skill")
	}
}

func TestResolveBundleSkillArgsAcceptsLocalAndFullIDs(t *testing.T) {
	skills := []library.Skill{
		{ID: "external/gstack/alpha", Name: "Alpha Skill"},
		{ID: "external/gstack/beta", DirName: "beta", Name: "Beta Skill"},
	}

	got, err := resolveBundleSkillArgs(
		"external/gstack",
		[]string{"external/gstack/alpha", "external/gstack/beta"},
		skills,
		[]string{"alpha", "external/gstack/beta"},
	)
	if err != nil {
		t.Fatalf("resolveBundleSkillArgs: %v", err)
	}
	want := []string{"external/gstack/alpha", "external/gstack/beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved IDs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestResolveBundleSkillArgsRejectsAmbiguousManifestName(t *testing.T) {
	skills := []library.Skill{
		{ID: "dev/alpha", Name: "Build"},
		{ID: "dev/beta", Name: "Build"},
	}

	_, err := resolveBundleSkillArgs("dev", []string{"dev/alpha", "dev/beta"}, skills, []string{"Build"})
	if err == nil {
		t.Fatalf("resolveBundleSkillArgs should reject ambiguous manifest names")
	}
	for _, want := range []string{"ambiguous", "dev/alpha", "dev/beta"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestResolveBundleSkillArgsRejectsMissingSkill(t *testing.T) {
	skills := []library.Skill{{ID: "dev/alpha", Name: "Alpha"}}

	_, err := resolveBundleSkillArgs("dev", []string{"dev/alpha"}, skills, []string{"missing"})
	if err == nil {
		t.Fatalf("resolveBundleSkillArgs should reject missing skills")
	}
	if !strings.Contains(err.Error(), `skill "missing" not found in bundle "dev"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPickBundleSkillIDsShowsManifestNames(t *testing.T) {
	skills := []library.Skill{
		{ID: "dev/beta", Name: "Beta Skill"},
		{ID: "dev/alpha", Name: "Alpha Skill"},
	}

	var gotItems []picker.Item
	var gotOpts picker.Opts
	chosen, err := pickBundleSkillIDs("dev", []string{"dev/beta", "dev/alpha"}, skills, func(items []picker.Item, opts picker.Opts) ([]string, error) {
		gotItems = items
		gotOpts = opts
		return []string{"dev/alpha"}, nil
	})
	if err != nil {
		t.Fatalf("pickBundleSkillIDs: %v", err)
	}
	if !gotOpts.Multi {
		t.Fatalf("skill picker should enable multi-select")
	}
	if gotOpts.Prompt != "pick skills > " {
		t.Fatalf("prompt = %q, want %q", gotOpts.Prompt, "pick skills > ")
	}
	if want := []string{"dev/alpha"}; !reflect.DeepEqual(chosen, want) {
		t.Fatalf("chosen skills mismatch\ngot:  %#v\nwant: %#v", chosen, want)
	}
	gotIDs := []string{gotItems[0].ID, gotItems[1].ID}
	if want := []string{"dev/alpha", "dev/beta"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("picker item order mismatch\ngot:  %#v\nwant: %#v", gotIDs, want)
	}
	for _, want := range []string{"Alpha Skill", "dev/alpha"} {
		if !strings.Contains(gotItems[0].Display, want) {
			t.Fatalf("first display missing %q: %q", want, gotItems[0].Display)
		}
	}
}
