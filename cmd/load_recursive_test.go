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
	"skl/internal/state"
)

func TestLoadCommandIncludesNestedBundlesWhenMarkedRecursive(t *testing.T) {
	setupHome(t)
	root := libraryRoot(t)
	writeSkillTree(t, filepath.Join(root, "skills", "sf", "sf-build"), "build")
	writeSkillTree(t, filepath.Join(root, "skills", "sf", "ref", "mp-tdd"), "tdd")
	writeSkillTree(t, filepath.Join(root, "skills", "sf", "ref", "deep", "mp-grill"), "grill")
	writeBundleSettingsFile(t, filepath.Join(root, "skills", "sf"), "recursive: true\n")

	out := runLoadCommand(t, "sf")

	if got, want := liveDirs(t), []string{"mp-grill", "mp-tdd", "sf-build"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("live dirs = %#v, want %#v", got, want)
	}
	liveRoot, _ := live.LivePath()
	if _, err := os.Stat(filepath.Join(liveRoot, library.BundleSettingsFile)); !os.IsNotExist(err) {
		t.Fatalf("%s must never reach ~/.skills, stat err: %v", library.BundleSettingsFile, err)
	}
	assertClaims(t, map[string][]string{
		"sf/sf-build":          {"sf"},
		"sf/ref/mp-tdd":        {"sf"},
		"sf/ref/deep/mp-grill": {"sf"},
	})
	if !strings.Contains(out, "includes sf/ref, sf/ref/deep") {
		t.Fatalf("load output should name the included bundles:\n%s", out)
	}
}

func TestLoadCommandRecursiveFlagWidensUnmarkedBundleOnce(t *testing.T) {
	setupHome(t)
	root := libraryRoot(t)
	writeSkillTree(t, filepath.Join(root, "external", "gstack", "agent"), "agent")
	writeSkillTree(t, filepath.Join(root, "external", "gstack", "browser-skills", "scrape"), "scrape")
	writeSkillTree(t, filepath.Join(root, "external", "gstack", "openclaw", "skills", "claw"), "claw")

	runLoadCommand(t, "external/gstack")
	if got, want := liveDirs(t), []string{"agent"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unmarked bundle should load only its own skills: %#v, want %#v", got, want)
	}

	out := runLoadCommand(t, "-r", "external/gstack")
	if got, want := liveDirs(t), []string{"agent", "claw", "scrape"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("-r should add nested bundles: %#v, want %#v", got, want)
	}
	assertClaims(t, map[string][]string{
		"external/gstack/agent":                 {"external/gstack"},
		"external/gstack/browser-skills/scrape": {"external/gstack"},
		"external/gstack/openclaw/skills/claw":  {"external/gstack"},
	})
	if !strings.Contains(out, "includes external/gstack/browser-skills, external/gstack/openclaw/skills") {
		t.Fatalf("-r output should name the included bundles:\n%s", out)
	}

	runUnloadCommand(t, "external/gstack")
	if got := liveDirs(t); len(got) != 0 {
		t.Fatalf("unload should remove everything the -r load added: %#v", got)
	}
}

func TestUnloadRecursiveBundleKeepsSkillsClaimedByNestedBundle(t *testing.T) {
	setupHome(t)
	root := libraryRoot(t)
	writeSkillTree(t, filepath.Join(root, "sf", "sf-build"), "build")
	writeSkillTree(t, filepath.Join(root, "sf", "ref", "mp-tdd"), "tdd")
	writeBundleSettingsFile(t, filepath.Join(root, "sf"), "recursive: true\n")

	runLoadCommand(t, "sf/ref")
	runLoadCommand(t, "sf")
	assertClaims(t, map[string][]string{
		"sf/sf-build":   {"sf"},
		"sf/ref/mp-tdd": {"sf", "sf/ref"},
	})

	runUnloadCommand(t, "sf")
	if got, want := liveDirs(t), []string{"mp-tdd"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("live dirs after unload sf = %#v, want %#v", got, want)
	}
	assertClaims(t, map[string][]string{"sf/ref/mp-tdd": {"sf/ref"}})

	// Unload reads recorded claims, not the library: removing the marker after
	// a load must not strand the nested skills that load added.
	runLoadCommand(t, "sf")
	if err := os.Remove(filepath.Join(root, "sf", library.BundleSettingsFile)); err != nil {
		t.Fatalf("Remove marker: %v", err)
	}
	runUnloadCommand(t, "sf")
	if got, want := liveDirs(t), []string{"mp-tdd"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("live dirs after marker removal + unload = %#v, want %#v", got, want)
	}
	assertClaims(t, map[string][]string{"sf/ref/mp-tdd": {"sf/ref"}})
}

// runLoadCommand drives `skl load` through flag parsing so -r/--recursive is
// exercised as typed. loadCmd's flag set is shared, so the flag is reset after.
func runLoadCommand(t *testing.T, args ...string) string {
	t.Helper()
	cmd := *loadCmd
	t.Cleanup(func() { _ = loadCmd.Flags().Set("recursive", "false") })
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parse load flags %v: %v", args, err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, cmd.Flags().Args()); err != nil {
		t.Fatalf("load %v: %v", args, err)
	}
	return out.String()
}

func runUnloadCommand(t *testing.T, args ...string) {
	t.Helper()
	cmd := *unloadCmd
	captureStdout(t, func() {
		if err := cmd.RunE(&cmd, args); err != nil {
			t.Fatalf("unload %v: %v", args, err)
		}
	})
}

func libraryRoot(t *testing.T) string {
	t.Helper()
	root, err := library.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	return root
}

func writeBundleSettingsFile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, library.BundleSettingsFile), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", library.BundleSettingsFile, err)
	}
}

func liveDirs(t *testing.T) []string {
	t.Helper()
	dirs, err := live.LoadedDirs()
	if err != nil {
		t.Fatalf("LoadedDirs: %v", err)
	}
	return dirs
}

func assertClaims(t *testing.T, want map[string][]string) {
	t.Helper()
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	st, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}
	got := map[string][]string{}
	for id, entry := range st.Loaded {
		got[id] = entry.Bundles
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("claims mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}
