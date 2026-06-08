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
	Name     string
	DirName  string
	SrcPath  string
	External bool
	Repo     string
}

const ReservedInboxBundle = "inbox"

const externalReadme = `# External skill repositories

Put cloned or imported third-party skill repos here.

Each direct child can be loaded as a folder bundle, for example:

    skl load external/gstack

Nested .git directories are ignored by skl sync. To update an external repo,
run git commands inside that repo.
`

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
	if err := ensureExternalReadme(external); err != nil {
		return err
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
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		id := filepath.ToSlash(rel)
		if isExternalRepoRoot(id) || !hasSkillManifest(path) {
			return nil
		}
		id = legacySkillID(id)
		name := manifestSkillName(path)
		if name == "" {
			name = id
		}
		out = append(out, Skill{
			ID:       id,
			Name:     name,
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

func hasSkillManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil
}

// manifestSkillName extracts optional display metadata without making malformed frontmatter fatal to discovery.
func manifestSkillName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	body := string(data)
	if !strings.HasPrefix(body, "---\n") {
		return ""
	}
	rest := strings.TrimPrefix(body, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}

	var manifest struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &manifest); err != nil {
		return ""
	}
	return strings.TrimSpace(manifest.Name)
}

func ensureExternalReadme(external string) error {
	path := filepath.Join(external, "README.md")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.WriteFile(path, []byte(externalReadme), 0o644); err != nil {
		return fmt.Errorf("writing external README: %w", err)
	}
	return nil
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".")
}

// isExternalRepoRoot treats external/<repo> as a namespace, not a skill.
func isExternalRepoRoot(id string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 2 && parts[0] == "external"
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
