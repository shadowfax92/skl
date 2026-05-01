package gitlib

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGitAddArgsExcludeNestedRepos(t *testing.T) {
	args := gitAddArgs([]string{
		filepath.Join("external", "gstack"),
		filepath.Join("packs", "obra"),
	})

	want := []string{
		"add",
		"-A",
		"--",
		".",
		":(exclude)external/gstack",
		":(exclude)packs/obra",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("git add args mismatch\ngot:  %#v\nwant: %#v", args, want)
	}
}

func TestNestedRepoPathsSkipsRootRepo(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, ".git"))
	mkdir(t, filepath.Join(root, "external", "gstack", ".git"))
	mkdir(t, filepath.Join(root, "external", "gstack", ".git", "objects", "nested", ".git"))

	paths, err := nestedRepoPaths(root)
	if err != nil {
		t.Fatalf("nestedRepoPaths: %v", err)
	}

	want := []string{filepath.Join("external", "gstack")}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("nested repo paths mismatch\ngot:  %#v\nwant: %#v", paths, want)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}
