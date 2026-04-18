package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"skl/internal/bundle"
	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/state"
)

func TestApplyLoadPlanReloadsTrackedSkill(t *testing.T) {
	setupHome(t)

	srcDir := filepath.Join(t.TempDir(), "foo-src")
	writeSkillTree(t, srcDir, "new")
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	writeSkillTree(t, filepath.Join(liveRoot, "foo"), "old")

	st := &state.State{
		Version: 1,
		Loaded: map[string]state.LoadEntry{
			"foo": makeLoadEntry("foo", "old-src", "dev"),
		},
	}

	plan := makeLoadPlan("dev", library.Skill{ID: "foo", DirName: "foo", SrcPath: srcDir}, true)
	newCount, reloaded, err := applyLoadPlan(plan, st)
	if err != nil {
		t.Fatalf("applyLoadPlan: %v", err)
	}
	if newCount != 0 || reloaded != 1 {
		t.Fatalf("counts mismatch: new=%d reloaded=%d", newCount, reloaded)
	}
	if got := readSkillBody(t, filepath.Join(liveRoot, "foo")); got != "new" {
		t.Fatalf("live skill body = %q, want %q", got, "new")
	}
	entry := st.Loaded["foo"]
	if entry.Source != srcDir {
		t.Fatalf("state source = %q, want %q", entry.Source, srcDir)
	}
	if !reflect.DeepEqual(entry.Bundles, []string{"dev"}) {
		t.Fatalf("state bundles = %#v", entry.Bundles)
	}
}

func TestApplyLoadPlanOverwritesUntrackedDirAfterConfirmation(t *testing.T) {
	setupHome(t)

	srcDir := filepath.Join(t.TempDir(), "foo-src")
	writeSkillTree(t, srcDir, "fresh")
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	writeSkillTree(t, filepath.Join(liveRoot, "foo"), "manual")

	st := &state.State{Version: 1, Loaded: map[string]state.LoadEntry{}}
	plan := makeLoadPlan("dev", library.Skill{ID: "foo", DirName: "foo", SrcPath: srcDir}, false)

	withStdin(t, "y\n", func() {
		_, _, err = applyLoadPlan(plan, st)
	})
	if err != nil {
		t.Fatalf("applyLoadPlan: %v", err)
	}
	if got := readSkillBody(t, filepath.Join(liveRoot, "foo")); got != "fresh" {
		t.Fatalf("live skill body = %q, want %q", got, "fresh")
	}
	if _, ok := st.Loaded["foo"]; !ok {
		t.Fatalf("state should contain newly loaded skill")
	}
}

func TestApplyLoadPlanReplacesConflictingLoadedDir(t *testing.T) {
	setupHome(t)

	srcDir := filepath.Join(t.TempDir(), "foo-src")
	writeSkillTree(t, srcDir, "replacement")
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	writeSkillTree(t, filepath.Join(liveRoot, "foo"), "old")

	st := &state.State{
		Version: 1,
		Loaded: map[string]state.LoadEntry{
			"pack/foo": makeLoadEntry("foo", "pack-src", "pack"),
		},
	}
	plan := makeLoadPlan("dev", library.Skill{ID: "foo", DirName: "foo", SrcPath: srcDir}, false)

	withStdin(t, "y\n", func() {
		_, _, err = applyLoadPlan(plan, st)
	})
	if err != nil {
		t.Fatalf("applyLoadPlan: %v", err)
	}
	if got := readSkillBody(t, filepath.Join(liveRoot, "foo")); got != "replacement" {
		t.Fatalf("live skill body = %q, want %q", got, "replacement")
	}
	if _, ok := st.Loaded["pack/foo"]; ok {
		t.Fatalf("conflicting state entry should be removed")
	}
	if _, ok := st.Loaded["foo"]; !ok {
		t.Fatalf("replacement skill should be present in state")
	}
}

func TestApplyLoadPlanRestoresReloadedSkillOnFailure(t *testing.T) {
	setupHome(t)

	srcDir := filepath.Join(t.TempDir(), "foo-src")
	writeSkillTree(t, srcDir, "new")
	liveRoot, err := live.EnsureLive()
	if err != nil {
		t.Fatalf("EnsureLive: %v", err)
	}
	writeSkillTree(t, filepath.Join(liveRoot, "foo"), "old")

	st := &state.State{
		Version: 1,
		Loaded: map[string]state.LoadEntry{
			"foo": makeLoadEntry("foo", "old-src", "dev"),
		},
	}
	plan := bundle.LoadPlan{
		Bundle: "dev",
		Actions: []bundle.LoadAction{
			{
				Skill:   library.Skill{ID: "foo", DirName: "foo", SrcPath: srcDir},
				Already: true,
			},
			{
				Skill: library.Skill{ID: "bar", DirName: "bar", SrcPath: filepath.Join(t.TempDir(), "missing")},
			},
		},
	}

	if _, _, err := applyLoadPlan(plan, st); err == nil {
		t.Fatalf("applyLoadPlan should fail when a later copy fails")
	}
	if got := readSkillBody(t, filepath.Join(liveRoot, "foo")); got != "old" {
		t.Fatalf("live skill body after rollback = %q, want %q", got, "old")
	}
	entry := st.Loaded["foo"]
	if entry.Source != "old-src" {
		t.Fatalf("state source after rollback = %q, want %q", entry.Source, "old-src")
	}
	if !reflect.DeepEqual(entry.Bundles, []string{"dev"}) {
		t.Fatalf("state bundles after rollback = %#v", entry.Bundles)
	}
}

func setupHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func makeLoadPlan(bundleName string, skill library.Skill, already bool) bundle.LoadPlan {
	return bundle.LoadPlan{
		Bundle: bundleName,
		Actions: []bundle.LoadAction{
			{
				Skill:   skill,
				Already: already,
			},
		},
	}
}

func makeLoadEntry(dirName, source string, bundles ...string) state.LoadEntry {
	return state.LoadEntry{
		DirName:  dirName,
		Source:   source,
		Bundles:  bundles,
		LoadedAt: time.Unix(123, 0).UTC(),
	}
}

func writeSkillTree(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "body.txt"), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(body.txt): %v", err)
	}
}

func readSkillBody(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "body.txt"))
	if err != nil {
		t.Fatalf("ReadFile(body.txt): %v", err)
	}
	return string(data)
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("WriteString(stdin): %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close(stdin writer): %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()
	fn()
}
