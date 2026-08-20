package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/picker"
	"skl/internal/state"
)

func TestUnpickCommandScopesPickerAndPreservesOtherClaims(t *testing.T) {
	setupHome(t)
	liveRoot := seedUnpickState(t, map[string]state.LoadEntry{
		"dev/alpha":  makeLoadEntry("alpha", "alpha-src", "dev"),
		"dev/beta":   makeLoadEntry("beta", "beta-src", "dev"),
		"dev/shared": makeLoadEntry("shared", "shared-src", "dev", "ops"),
		"ops/other":  makeLoadEntry("other", "other-src", "ops"),
	})
	libraryRoot, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkillTree(t, filepath.Join(libraryRoot, "dev", "not-loaded"), "library only")

	oldPick := unpickSkillItems
	defer func() { unpickSkillItems = oldPick }()
	var gotItems []picker.Item
	unpickSkillItems = func(items []picker.Item, opts picker.Opts) ([]string, error) {
		gotItems = append([]picker.Item(nil), items...)
		if !opts.Multi || opts.Prompt != "unpick skills > " {
			t.Fatalf("unexpected picker opts: %#v", opts)
		}
		return []string{"dev/alpha", "dev/shared"}, nil
	}

	cmd := *unpickCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, []string{"dev"}); err != nil {
		t.Fatalf("unpick RunE: %v", err)
	}

	gotIDs := make([]string, 0, len(gotItems))
	for _, item := range gotItems {
		gotIDs = append(gotIDs, item.ID)
	}
	if want := []string{"dev/alpha", "dev/beta", "dev/shared"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("picker IDs = %#v, want %#v", gotIDs, want)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("selected orphan should be removed, stat err: %v", err)
	}
	for _, dir := range []string{"beta", "shared", "other"} {
		if _, err := os.Stat(filepath.Join(liveRoot, dir)); err != nil {
			t.Fatalf("skill %q should remain loaded: %v", dir, err)
		}
	}

	st := loadUnpickState(t)
	if _, ok := st.Loaded["dev/alpha"]; ok {
		t.Fatalf("selected orphan should leave state")
	}
	if got := st.Loaded["dev/shared"].Bundles; !reflect.DeepEqual(got, []string{"ops"}) {
		t.Fatalf("shared claims = %#v, want ops", got)
	}
	for _, id := range []string{"dev/beta", "ops/other"} {
		if _, ok := st.Loaded[id]; !ok {
			t.Fatalf("unselected skill %q should remain in state", id)
		}
	}
}

func TestUnpickCommandWithoutArgsCancelsWithoutChanges(t *testing.T) {
	setupHome(t)
	liveRoot := seedUnpickState(t, map[string]state.LoadEntry{
		"dev/alpha": makeLoadEntry("alpha", "alpha-src", "dev"),
		"ops/beta":  makeLoadEntry("beta", "beta-src", "ops"),
	})

	oldPick := unpickSkillItems
	defer func() { unpickSkillItems = oldPick }()
	var gotIDs []string
	unpickSkillItems = func(items []picker.Item, opts picker.Opts) ([]string, error) {
		for _, item := range items {
			gotIDs = append(gotIDs, item.ID)
		}
		return nil, nil
	}

	cmd := *unpickCmd
	err := cmd.RunE(&cmd, nil)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("unpick error = %v, want ErrCancelled", err)
	}
	if want := []string{"dev/alpha", "ops/beta"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("picker IDs = %#v, want %#v", gotIDs, want)
	}
	if got := loadUnpickState(t).Loaded; len(got) != 2 {
		t.Fatalf("state changed after cancellation: %#v", got)
	}
	for _, dir := range []string{"alpha", "beta"} {
		if _, err := os.Stat(filepath.Join(liveRoot, dir)); err != nil {
			t.Fatalf("skill %q changed after cancellation: %v", dir, err)
		}
	}
}

func TestUnpickCommandAcceptsExplicitBundleSkills(t *testing.T) {
	setupHome(t)
	liveRoot := seedUnpickState(t, map[string]state.LoadEntry{
		"dev/alpha": makeLoadEntry("alpha", "alpha-src", "dev"),
		"dev/beta":  makeLoadEntry("beta", "beta-src", "dev"),
	})

	oldPick := unpickSkillItems
	defer func() { unpickSkillItems = oldPick }()
	unpickSkillItems = func(items []picker.Item, opts picker.Opts) ([]string, error) {
		t.Fatal("explicit skills should not open the picker")
		return nil, nil
	}

	cmd := *unpickCmd
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.RunE(&cmd, []string{"dev", "beta"}); err != nil {
		t.Fatalf("unpick RunE: %v", err)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "beta")); !os.IsNotExist(err) {
		t.Fatalf("explicit skill should be removed, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(liveRoot, "alpha")); err != nil {
		t.Fatalf("unselected skill should remain: %v", err)
	}
	st := loadUnpickState(t)
	if _, ok := st.Loaded["dev/beta"]; ok {
		t.Fatalf("explicit skill should leave state")
	}
	if _, ok := st.Loaded["dev/alpha"]; !ok {
		t.Fatalf("unselected skill should remain in state")
	}
}

func TestUnpickCommandRollsBackOnRemovalFailure(t *testing.T) {
	setupHome(t)
	liveRoot := seedUnpickState(t, map[string]state.LoadEntry{
		"dev/alpha":   makeLoadEntry("alpha", "alpha-src", "dev"),
		"dev/blocked": makeLoadEntry(".blocked", "blocked-src", "dev"),
		"dev/shared":  makeLoadEntry("shared", "shared-src", "dev", "ops"),
	})

	oldPick := unpickSkillItems
	defer func() { unpickSkillItems = oldPick }()
	unpickSkillItems = func(items []picker.Item, opts picker.Opts) ([]string, error) {
		return []string{"dev/alpha", "dev/shared", "dev/blocked"}, nil
	}

	cmd := *unpickCmd
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.RunE(&cmd, []string{"dev"}); err == nil {
		t.Fatalf("unpick should report the removal failure")
	}
	st := loadUnpickState(t)
	if len(st.Loaded) != 3 {
		t.Fatalf("state changed after failed unpick: %#v", st.Loaded)
	}
	if got := st.Loaded["dev/shared"].Bundles; !reflect.DeepEqual(got, []string{"dev", "ops"}) {
		t.Fatalf("state changed after failed unpick: %#v", got)
	}
	for _, dir := range []string{"alpha", ".blocked", "shared"} {
		if _, err := os.Stat(filepath.Join(liveRoot, dir)); err != nil {
			t.Fatalf("skill %q should be restored after failure: %v", dir, err)
		}
	}
}

func seedUnpickState(t *testing.T, loaded map[string]state.LoadEntry) string {
	t.Helper()
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	for _, entry := range loaded {
		writeSkillTree(t, filepath.Join(liveRoot, entry.DirName), entry.DirName)
	}
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := mgr.Save(&state.State{Version: 1, Loaded: loaded}); err != nil {
		t.Fatalf("Save state: %v", err)
	}
	return liveRoot
}

func loadUnpickState(t *testing.T) *state.State {
	t.Helper()
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	st, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	return st
}
