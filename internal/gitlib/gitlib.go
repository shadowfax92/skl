package gitlib

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func IsRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

func Init(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return run(dir, "init", "-b", "main")
}

func HasUpstream(dir string) bool {
	_, err := output(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	return err == nil
}

func RemoteURL(dir string) (string, error) {
	out, err := output(dir, "remote", "get-url", "origin")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func SetRemote(dir, url string) error {
	if existing, _ := RemoteURL(dir); existing != "" {
		return run(dir, "remote", "set-url", "origin", url)
	}
	return run(dir, "remote", "add", "origin", url)
}

func HasStagedChanges(dir string) (bool, error) {
	out, err := output(dir, "diff", "--cached", "--name-only")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func AddCommit(dir, msg string) error {
	nestedRepos, err := nestedRepoPaths(dir)
	if err != nil {
		return err
	}
	if err := run(dir, gitAddArgs(nestedRepos)...); err != nil {
		return err
	}
	dirty, err := HasStagedChanges(dir)
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	return run(dir, "commit", "-m", msg)
}

func gitAddArgs(excludePaths []string) []string {
	args := []string{"add", "-A", "--", "."}
	for _, p := range excludePaths {
		args = append(args, ":(exclude)"+filepath.ToSlash(p))
	}
	return args
}

func nestedRepoPaths(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == root || d.Name() != ".git" {
			return nil
		}
		parent := filepath.Dir(path)
		if parent == root {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func PullRebase(dir string) error {
	url, _ := RemoteURL(dir)
	if url == "" {
		return errors.New("no `origin` remote configured")
	}
	return run(dir, "pull", "--rebase", "origin", "HEAD")
}

func Push(dir string) error {
	url, _ := RemoteURL(dir)
	if url == "" {
		return errors.New("no `origin` remote configured")
	}
	return run(dir, "push", "-u", "origin", "HEAD")
}

func Clone(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination %s already exists", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dest)
		return err
	}
	return nil
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
