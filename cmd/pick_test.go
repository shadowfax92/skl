package cmd

import (
	"bytes"
	"errors"
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

func TestPickCommandWithoutArgsShowsAllSkillsDirectly(t *testing.T) {
	setupHome(t)

	root, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkillTree(t, filepath.Join(root, "dev", "alpha"), "alpha")
	writeSkillTree(t, filepath.Join(root, "dev", "beta"), "beta")
	writeSkillTree(t, filepath.Join(root, "writing", "gamma"), "gamma")

	oldPickSkillItems := pickSkillItems
	defer func() { pickSkillItems = oldPickSkillItems }()

	var gotItems []picker.Item
	var gotOpts picker.Opts
	pickCalls := 0
	pickSkillItems = func(items []picker.Item, opts picker.Opts) ([]string, error) {
		pickCalls++
		gotItems = append([]picker.Item(nil), items...)
		gotOpts = opts
		return []string{"dev/alpha", "writing/gamma"}, nil
	}

	cmd := *pickCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("pick RunE: %v", err)
	}

	if pickCalls != 1 {
		t.Fatalf("pick calls = %d, want 1", pickCalls)
	}
	if !gotOpts.Multi {
		t.Fatalf("flat skill picker should enable multi-select")
	}
	if gotOpts.Prompt != "pick skills > " {
		t.Fatalf("prompt = %q, want %q", gotOpts.Prompt, "pick skills > ")
	}

	gotIDs := make([]string, 0, len(gotItems))
	for _, item := range gotItems {
		gotIDs = append(gotIDs, item.ID)
	}
	if want := []string{"dev/alpha", "dev/beta", "writing/gamma"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("picker items mismatch\ngot:  %#v\nwant: %#v", gotIDs, want)
	}

	liveRoot, err := live.LivePath()
	if err != nil {
		t.Fatalf("LivePath: %v", err)
	}
	for _, name := range []string{"alpha", "gamma"} {
		if _, err := os.Stat(filepath.Join(liveRoot, name)); err != nil {
			t.Fatalf("selected skill %q should be copied: %v", name, err)
		}
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
	for id, wantBundles := range map[string][]string{
		"dev/alpha":     {"dev"},
		"writing/gamma": {"writing"},
	} {
		entry, ok := st.Loaded[id]
		if !ok {
			t.Fatalf("state missing selected skill %q: %#v", id, st.Loaded)
		}
		if !reflect.DeepEqual(entry.Bundles, wantBundles) {
			t.Fatalf("%s bundles = %#v, want %#v", id, entry.Bundles, wantBundles)
		}
	}
	if _, ok := st.Loaded["dev/beta"]; ok {
		t.Fatalf("state should not include unselected skill")
	}
}

func TestLoadSelectedSkillGroupsRollsBackEarlierGroupsOnCancel(t *testing.T) {
	setupHome(t)

	root, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkillTree(t, filepath.Join(root, "dev", "alpha"), "alpha")
	writeSkillTree(t, filepath.Join(root, "writing", "beta"), "beta")

	all, err := library.Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	writeSkillTree(t, filepath.Join(liveRoot, "beta"), "manual")

	var newCount, reloaded int
	withStdin(t, "n\n", func() {
		newCount, reloaded, err = loadSelectedSkillGroups(map[string][]string{
			"dev":     {"dev/alpha"},
			"writing": {"writing/beta"},
		}, all)
	})
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("loadSelectedSkillGroups err = %v, want ErrCancelled", err)
	}
	if newCount != 0 || reloaded != 0 {
		t.Fatalf("counts on failure = new %d reload %d, want zero", newCount, reloaded)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("earlier group should be rolled back, stat err: %v", err)
	}
	if got := readSkillBody(t, filepath.Join(liveRoot, "beta")); got != "manual" {
		t.Fatalf("cancelled group body = %q, want manual", got)
	}

	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	st, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	if len(st.Loaded) != 0 {
		t.Fatalf("state should stay empty after rollback: %#v", st.Loaded)
	}
}

func TestLoadSelectedSkillGroupsRejectsDuplicateLiveDirNames(t *testing.T) {
	setupHome(t)

	root, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkillTree(t, filepath.Join(root, "dev", "foo"), "dev")
	writeSkillTree(t, filepath.Join(root, "ops", "foo"), "ops")

	all, err := library.Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	withStdin(t, "y\n", func() {
		_, _, err = loadSelectedSkillGroups(map[string][]string{
			"dev": {"dev/foo"},
			"ops": {"ops/foo"},
		}, all)
	})
	if err == nil {
		t.Fatalf("loadSelectedSkillGroups should reject duplicate live dir names")
	}
	for _, want := range []string{"dev/foo", "ops/foo", "~/.skills/foo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}

	liveRoot, err := live.LivePath()
	if err != nil {
		t.Fatalf("LivePath: %v", err)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "foo")); !os.IsNotExist(err) {
		t.Fatalf("duplicate selection should fail before copying, stat err: %v", err)
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

func TestResolveBundleSkillArgsRejectsLocalAndManifestNameCollision(t *testing.T) {
	skills := []library.Skill{
		{ID: "dev/build", DirName: "build", Name: "Build Tool"},
		{ID: "dev/test", DirName: "test", Name: "build"},
	}

	_, err := resolveBundleSkillArgs("dev", []string{"dev/build", "dev/test"}, skills, []string{"build"})
	if err == nil {
		t.Fatalf("resolveBundleSkillArgs should reject local/name collisions")
	}
	for _, want := range []string{"ambiguous", "dev/build", "dev/test"} {
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

func TestPickBundleSkillIDsRejectsOutOfBundlePickerResult(t *testing.T) {
	skills := []library.Skill{
		{ID: "dev/alpha", Name: "Alpha Skill"},
		{ID: "other/beta", Name: "Injected Skill"},
	}

	_, err := pickBundleSkillIDs("dev", []string{"dev/alpha"}, skills, func(items []picker.Item, opts picker.Opts) ([]string, error) {
		return []string{"other/beta"}, nil
	})
	if err == nil {
		t.Fatalf("pickBundleSkillIDs should reject out-of-bundle IDs")
	}
	if !strings.Contains(err.Error(), `selected skill "other/beta" is not in bundle "dev"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPickBundleSkillIDsSanitizesDisplayNames(t *testing.T) {
	skills := []library.Skill{
		{ID: "dev/alpha", Name: "Alpha\nother/beta\tInjected\x1b[31m"},
	}

	var gotItems []picker.Item
	_, err := pickBundleSkillIDs("dev", []string{"dev/alpha"}, skills, func(items []picker.Item, opts picker.Opts) ([]string, error) {
		gotItems = items
		return []string{"dev/alpha"}, nil
	})
	if err != nil {
		t.Fatalf("pickBundleSkillIDs: %v", err)
	}
	if strings.ContainsAny(gotItems[0].Display, "\n\t\x1b") {
		t.Fatalf("display should be single-line without control characters: %q", gotItems[0].Display)
	}
	if !strings.Contains(gotItems[0].Display, "Alpha other/beta Injected") {
		t.Fatalf("display should preserve sanitized words: %q", gotItems[0].Display)
	}
}
