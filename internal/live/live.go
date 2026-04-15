package live

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)


func LivePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skills"), nil
}

func EnsureLive() (string, error) {
	p, err := LivePath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", err
	}
	return p, nil
}

func LoadedDirs() ([]string, error) {
	p, err := LivePath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

func SkillExists(dirName string) (bool, error) {
	if err := guardDirName(dirName); err != nil {
		return false, err
	}
	root, err := LivePath()
	if err != nil {
		return false, err
	}
	info, err := os.Stat(filepath.Join(root, dirName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// CopySkill recursively copies srcDir into ~/.skills/<dirName>.
// Refuses dot names. Rolls back the target on any failure.
func CopySkill(srcDir, dirName string) error {
	if err := guardDirName(dirName); err != nil {
		return err
	}
	root, err := EnsureLive()
	if err != nil {
		return err
	}
	dst := filepath.Join(root, dirName)

	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("skill %q already loaded; unload first", dirName)
	}

	if err := copyTree(srcDir, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return nil
}

func RemoveSkill(dirName string) error {
	if err := guardDirName(dirName); err != nil {
		return err
	}
	root, err := LivePath()
	if err != nil {
		return err
	}
	target := filepath.Join(root, dirName)
	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.RemoveAll(target)
}

func guardDirName(name string) error {
	if name == "" {
		return errors.New("empty skill name")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("refusing to operate on dot-prefixed entry %q", name)
	}
	if strings.ContainsRune(name, filepath.Separator) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid skill name %q", name)
	}
	return nil
}

// copyTree recursively copies src into dst. Symlinks are skipped — skills can
// come from untrusted git repos via `skl install`, and a malicious symlink
// could otherwise cause reads outside the skill directory.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "skl: skipping symlink %s\n", path)
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
