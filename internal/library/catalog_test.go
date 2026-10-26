package library

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCatalogRecursiveBundleIncludesNestedBundles(t *testing.T) {
	root := setupLibrary(t)
	writeSkill(t, filepath.Join(root, "skills", "sf", "sf-build"))
	writeSkill(t, filepath.Join(root, "skills", "sf", "sf-plan"))
	writeSkill(t, filepath.Join(root, "skills", "sf", "ref", "mp-tdd"))
	writeSkill(t, filepath.Join(root, "skills", "sf", "ref", "deep", "mp-grill"))
	writeBundleSettings(t, filepath.Join(root, "skills", "sf"), "recursive: true\n")

	cat, err := ReadCatalog()
	if err != nil {
		t.Fatalf("ReadCatalog: %v", err)
	}
	want := map[string][]string{
		"sf":          {"sf/ref/deep/mp-grill", "sf/ref/mp-tdd", "sf/sf-build", "sf/sf-plan"},
		"sf/ref":      {"sf/ref/mp-tdd"},
		"sf/ref/deep": {"sf/ref/deep/mp-grill"},
	}
	if !reflect.DeepEqual(cat.Bundles, want) {
		t.Fatalf("Bundles mismatch\ngot:  %#v\nwant: %#v", cat.Bundles, want)
	}
	if !reflect.DeepEqual(cat.Recursive, map[string]bool{"sf": true}) {
		t.Fatalf("Recursive = %#v, want only sf", cat.Recursive)
	}
	if got, want := cat.Nested("sf"), []string{"sf/ref", "sf/ref/deep"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Nested(sf) = %#v, want %#v", got, want)
	}
}

func TestCatalogResolveWidensUnmarkedBundleOnlyWithFlag(t *testing.T) {
	root := setupLibrary(t)
	writeSkill(t, filepath.Join(root, "external", "gstack", "agent"))
	writeSkill(t, filepath.Join(root, "external", "gstack", "browse"))
	writeSkill(t, filepath.Join(root, "external", "gstack", "browser-skills", "scrape"))
	writeSkill(t, filepath.Join(root, "external", "gstack", "openclaw", "skills", "claw"))

	cat, err := ReadCatalog()
	if err != nil {
		t.Fatalf("ReadCatalog: %v", err)
	}
	if len(cat.Recursive) != 0 {
		t.Fatalf("no bundle should be recursive without skl.yaml: %#v", cat.Recursive)
	}

	plain, err := cat.Resolve("external/gstack", false)
	if err != nil {
		t.Fatalf("Resolve(plain): %v", err)
	}
	if want := []string{"external/gstack/agent", "external/gstack/browse"}; !reflect.DeepEqual(plain, want) {
		t.Fatalf("plain load = %#v, want %#v", plain, want)
	}

	all, err := cat.Resolve("external/gstack", true)
	if err != nil {
		t.Fatalf("Resolve(recursive): %v", err)
	}
	want := []string{
		"external/gstack/agent",
		"external/gstack/browse",
		"external/gstack/browser-skills/scrape",
		"external/gstack/openclaw/skills/claw",
	}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("recursive load = %#v, want %#v", all, want)
	}

	// openclaw holds no skills of its own: not a bundle, but -r reaches below it.
	if _, err := cat.Resolve("external/gstack/openclaw", false); err == nil {
		t.Fatalf("folder without direct skills should not resolve without -r")
	}
	nestedOnly, err := cat.Resolve("external/gstack/openclaw", true)
	if err != nil {
		t.Fatalf("Resolve(nested-only, recursive): %v", err)
	}
	if want := []string{"external/gstack/openclaw/skills/claw"}; !reflect.DeepEqual(nestedOnly, want) {
		t.Fatalf("nested-only load = %#v, want %#v", nestedOnly, want)
	}

	for _, recursive := range []bool{false, true} {
		if _, err := cat.Resolve("missing", recursive); err == nil || !strings.Contains(err.Error(), `bundle "missing" not found`) {
			t.Fatalf("Resolve(missing, %v) error = %v", recursive, err)
		}
	}
}

func TestCatalogRecursiveGroupFolderBecomesBundle(t *testing.T) {
	root := setupLibrary(t)
	writeSkill(t, filepath.Join(root, "writing", "blog", "draft"))
	writeSkill(t, filepath.Join(root, "writing", "social", "thread"))
	writeBundleSettings(t, filepath.Join(root, "writing"), "recursive: true\n")
	writeBundleSettings(t, filepath.Join(root, "empty"), "recursive: true\n")

	cat, err := ReadCatalog()
	if err != nil {
		t.Fatalf("ReadCatalog: %v", err)
	}
	if got, want := cat.Bundles["writing"], []string{"writing/blog/draft", "writing/social/thread"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("writing = %#v, want %#v", got, want)
	}
	if _, ok := cat.Bundles["empty"]; ok {
		t.Fatalf("recursive folder with nothing below it should not be a bundle")
	}
	if cat.Recursive["empty"] {
		t.Fatalf("empty folder should not be listed as recursive")
	}
}

func TestCatalogRejectsMalformedBundleSettings(t *testing.T) {
	cases := map[string]string{
		"misspelled key": "recursve: true\n",
		"wrong type":     "recursive: [true]\n",
		"not a mapping":  "- recursive\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			root := setupLibrary(t)
			writeSkill(t, filepath.Join(root, "dev", "alpha"))
			writeBundleSettings(t, filepath.Join(root, "dev"), body)

			_, err := ReadCatalog()
			if err == nil {
				t.Fatalf("ReadCatalog should reject %q", body)
			}
			if path := filepath.Join(root, "dev", BundleSettingsFile); !strings.Contains(err.Error(), path) {
				t.Fatalf("error should name %s, got: %v", path, err)
			}
			if _, err := Skills(); err != nil {
				t.Fatalf("Skills should not parse bundle settings: %v", err)
			}
		})
	}
}

func TestCatalogIgnoresInertBundleSettings(t *testing.T) {
	root := setupLibrary(t)
	writeSkill(t, filepath.Join(root, "empty", "alpha"))
	writeSkill(t, filepath.Join(root, "empty", "nested", "beta"))
	writeBundleSettings(t, filepath.Join(root, "empty"), "")

	writeSkill(t, filepath.Join(root, "off", "alpha"))
	writeSkill(t, filepath.Join(root, "off", "nested", "beta"))
	writeBundleSettings(t, filepath.Join(root, "off"), "# skl bundle settings\nrecursive: false\n")

	writeSkill(t, filepath.Join(root, "linked", "alpha"))
	writeSkill(t, filepath.Join(root, "linked", "nested", "beta"))
	target := filepath.Join(t.TempDir(), BundleSettingsFile)
	if err := os.WriteFile(target, []byte("recursive: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(target): %v", err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked", BundleSettingsFile)); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	// skl.yaml inside a skill folder belongs to the skill, not a bundle.
	writeSkill(t, filepath.Join(root, "inner", "alpha"))
	writeSkill(t, filepath.Join(root, "inner", "nested", "beta"))
	writeBundleSettings(t, filepath.Join(root, "inner", "alpha"), "recursive: true\n")

	cat, err := ReadCatalog()
	if err != nil {
		t.Fatalf("ReadCatalog: %v", err)
	}
	if len(cat.Recursive) != 0 {
		t.Fatalf("no bundle should be recursive: %#v", cat.Recursive)
	}
	for _, name := range []string{"empty", "off", "linked", "inner"} {
		if got, want := cat.Bundles[name], []string{name + "/alpha"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func TestHomeBundleIsDirectFolder(t *testing.T) {
	cases := map[string]string{
		"sf/ref/mp-tdd":         "sf/ref",
		"sf/sf-build":           "sf",
		"external/gstack/agent": "external/gstack",
		"loose":                 ReservedInboxBundle,
	}
	for id, want := range cases {
		if got := HomeBundle(id); got != want {
			t.Errorf("HomeBundle(%q) = %q, want %q", id, got, want)
		}
	}
}

func setupLibrary(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root, err := LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	return root
}

func writeBundleSettings(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, BundleSettingsFile), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", BundleSettingsFile, err)
	}
}
