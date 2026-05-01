package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	ID       string
	DirName  string
	SrcPath  string
	External bool
	Repo     string
}

const ReservedInboxBundle = "inbox"

type bundleFile struct {
	Bundles map[string][]string `yaml:"bundles"`
}

func LibraryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "skl", "library"), nil
}

func SkillsPath() (string, error) {
	root, err := LibraryPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "skills"), nil
}

func ExternalPath() (string, error) {
	root, err := LibraryPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "external"), nil
}

func BundlesPath() (string, error) {
	root, err := LibraryPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "bundles.yaml"), nil
}

// BundlePath resolves a folder bundle name inside the library root.
// Bundle names are slash-separated relative paths so commands cannot escape
// the library with absolute paths or ".." traversal.
func BundlePath(name string) (string, error) {
	root, err := LibraryPath()
	if err != nil {
		return "", err
	}
	if err := validateRelativeName(name); err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(name)), nil
}

func EnsureLibrary() error {
	skills, err := SkillsPath()
	if err != nil {
		return err
	}
	external, err := ExternalPath()
	if err != nil {
		return err
	}
	for _, d := range []string{skills, external} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}
	return nil
}

func Skills() ([]Skill, error) {
	if err := EnsureLibrary(); err != nil {
		return nil, err
	}
	var out []Skill

	root, _ := LibraryPath()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if path == root || !hasSkillManifest(path) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		id := filepath.ToSlash(rel)
		id = legacySkillID(id)
		out = append(out, Skill{
			ID:       id,
			DirName:  filepath.Base(path),
			SrcPath:  path,
			External: strings.HasPrefix(id, "external/"),
			Repo:     repoNameFromSkillID(id),
		})
		return filepath.SkipDir
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func FindSkill(id string) (*Skill, error) {
	all, err := Skills()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("skill %q not found in library", id)
}

func Bundles() (map[string][]string, error) {
	skills, err := Skills()
	if err != nil {
		return nil, err
	}

	bundles := make(map[string][]string)
	var inbox []string
	for _, skill := range skills {
		parent := pathDir(skill.ID)
		if parent == "" {
			inbox = append(inbox, skill.ID)
			continue
		}
		bundles[parent] = append(bundles[parent], skill.ID)
	}
	if len(inbox) > 0 {
		bundles[ReservedInboxBundle] = inbox
	}
	for name := range bundles {
		sort.Strings(bundles[name])
	}
	return bundles, nil
}

func readPersistedBundles() (map[string][]string, error) {
	path, err := BundlesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string][]string{}, nil
		}
		return nil, fmt.Errorf("reading bundles.yaml: %w", err)
	}
	var f bundleFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing bundles.yaml: %w", err)
	}
	if f.Bundles == nil {
		f.Bundles = map[string][]string{}
	}
	return f.Bundles, nil
}

func WriteBundles(b map[string][]string) error {
	if err := EnsureLibrary(); err != nil {
		return err
	}
	path, err := BundlesPath()
	if err != nil {
		return err
	}

	cleaned := make(map[string][]string, len(b))
	for name, skills := range b {
		if name == ReservedInboxBundle {
			continue
		}
		cleaned[name] = dedupSorted(skills)
	}

	data, err := yaml.Marshal(bundleFile{Bundles: cleaned})
	if err != nil {
		return fmt.Errorf("marshaling bundles.yaml: %w", err)
	}
	header := "# skl bundles — edit by hand or via `skl bundle ...`\n\n"

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(header+string(data)), 0o644); err != nil {
		return fmt.Errorf("writing bundles.yaml: %w", err)
	}
	return os.Rename(tmp, path)
}

func hasSkillManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".")
}

func legacySkillID(id string) string {
	if rest, ok := strings.CutPrefix(id, "skills/"); ok {
		return rest
	}
	return id
}

func validateRelativeName(name string) error {
	if name == "" || name == "." {
		return fmt.Errorf("bundle name cannot be empty")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("bundle name %q must be relative", name)
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("bundle name %q escapes the library", name)
	}
	return nil
}

func repoNameFromSkillID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 2 && parts[0] == "external" {
		return parts[1]
	}
	return ""
}

func pathDir(id string) string {
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(id)))
	if parent == "." {
		return ""
	}
	return parent
}

func dedupSorted(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
