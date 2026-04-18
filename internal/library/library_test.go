package library

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBundlesDerivesInboxAndWriteStripsIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	skillsRoot, err := SkillsPath()
	if err != nil {
		t.Fatalf("SkillsPath: %v", err)
	}
	writeSkill(t, filepath.Join(skillsRoot, "alpha"))
	writeSkill(t, filepath.Join(skillsRoot, "beta"))
	writeSkill(t, filepath.Join(skillsRoot, "gamma"))

	if err := WriteBundles(map[string][]string{
		"dev":               {"alpha"},
		ReservedInboxBundle: {"beta"},
	}); err != nil {
		t.Fatalf("WriteBundles: %v", err)
	}

	bundles, err := Bundles()
	if err != nil {
		t.Fatalf("Bundles: %v", err)
	}
	want := map[string][]string{
		"dev":               {"alpha"},
		ReservedInboxBundle: {"beta", "gamma"},
	}
	if !reflect.DeepEqual(bundles, want) {
		t.Fatalf("Bundles mismatch\ngot:  %#v\nwant: %#v", bundles, want)
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

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}
