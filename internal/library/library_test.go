package library

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestFolderBundlesUseDirectoriesAsSourceOfTruth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root, err := LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkill(t, filepath.Join(root, "gstack", "alpha"))
	writeSkill(t, filepath.Join(root, "gstack", "beta"))
	writeSkill(t, filepath.Join(root, "gstack", "oh", "deploy"))
	writeSkill(t, filepath.Join(root, "external", "gstack", "agent"))
	writeSkill(t, filepath.Join(root, "loose"))

	if err := WriteBundles(map[string][]string{
		"ignored-yaml": {"loose"},
	}); err != nil {
		t.Fatalf("WriteBundles: %v", err)
	}

	skills, err := Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	gotIDs := skillIDs(skills)
	wantIDs := []string{
		"external/gstack/agent",
		"gstack/alpha",
		"gstack/beta",
		"gstack/oh/deploy",
		"loose",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("skill IDs mismatch\ngot:  %#v\nwant: %#v", gotIDs, wantIDs)
	}

	bundles, err := Bundles()
	if err != nil {
		t.Fatalf("Bundles: %v", err)
	}
	want := map[string][]string{
		"external/gstack":   {"external/gstack/agent"},
		"gstack":            {"gstack/alpha", "gstack/beta"},
		"gstack/oh":         {"gstack/oh/deploy"},
		ReservedInboxBundle: {"loose"},
	}
	if !reflect.DeepEqual(bundles, want) {
		t.Fatalf("Bundles mismatch\ngot:  %#v\nwant: %#v", bundles, want)
	}
}

func TestWriteBundlesStillStripsInboxForLegacyCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := WriteBundles(map[string][]string{
		"dev":               {"alpha"},
		ReservedInboxBundle: {"beta"},
	}); err != nil {
		t.Fatalf("WriteBundles: %v", err)
	}

	bundlesPath, err := BundlesPath()
	if err != nil {
		t.Fatalf("BundlesPath: %v", err)
	}
	data, err := os.ReadFile(bundlesPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", bundlesPath, err)
	}
	if strings.Contains(string(data), ReservedInboxBundle+":") {
		t.Fatalf("bundles.yaml should not persist %q:\n%s", ReservedInboxBundle, data)
	}
}

func TestSkillsIgnoreNestedGitInternals(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root, err := LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkill(t, filepath.Join(root, "external", "gstack", "skill"))
	writeSkill(t, filepath.Join(root, "external", "gstack", ".git", "objects", "bad"))

	skills, err := Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	got := skillIDs(skills)
	want := []string{"external/gstack/skill"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skill IDs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestLegacySkillsDirectoryRemainsUnbundled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root, err := LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	writeSkill(t, filepath.Join(root, "skills", "alpha"))

	skills, err := Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	got := skillIDs(skills)
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skill IDs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}

	bundles, err := Bundles()
	if err != nil {
		t.Fatalf("Bundles: %v", err)
	}
	wantBundles := map[string][]string{ReservedInboxBundle: {"alpha"}}
	if !reflect.DeepEqual(bundles, wantBundles) {
		t.Fatalf("bundles mismatch\ngot:  %#v\nwant: %#v", bundles, wantBundles)
	}
}

func TestBundlePathRejectsPathsOutsideLibrary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := BundlePath("../outside"); err == nil {
		t.Fatalf("BundlePath should reject parent traversal")
	}
	if _, err := BundlePath("/tmp/outside"); err == nil {
		t.Fatalf("BundlePath should reject absolute paths")
	}

	got, err := BundlePath("gstack/oh")
	if err != nil {
		t.Fatalf("BundlePath: %v", err)
	}
	want := filepath.Join(home, ".config", "skl", "library", "gstack", "oh")
	if got != want {
		t.Fatalf("BundlePath mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func skillIDs(skills []Skill) []string {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		out = append(out, skill.ID)
	}
	sort.Strings(out)
	return out
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}
