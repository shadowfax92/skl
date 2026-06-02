package gitlib

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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

func TestAddCommitSkipsIgnoredNestedReposWithoutPathspecError(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "external", "gstack"))
	mkdir(t, filepath.Join(root, "skills", "foo"))
	writeFile(t, filepath.Join(root, ".gitignore"), "external/gstack/\n")
	writeFile(t, filepath.Join(root, "skills", "foo", "SKILL.md"), "test skill\n")

	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, filepath.Join(root, "external", "gstack"), "init", "-b", "main")

	if err := AddCommit(root, "sync"); err != nil {
		t.Fatalf("AddCommit: %v", err)
	}

	files := gitOutput(t, root, "ls-files")
	if !strings.Contains(files, "skills/foo/SKILL.md") {
		t.Fatalf("expected normal library file to be tracked, got:\n%s", files)
	}
	if strings.Contains(files, "external/gstack") {
		t.Fatalf("expected ignored nested repo to stay untracked, got:\n%s", files)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
