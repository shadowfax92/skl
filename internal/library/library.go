package library

import (
	"errors"
	"fmt"
	"io"
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
const skillManifestReadLimit = 64 * 1024

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
	scan, err := scanLibrary()
	if err != nil {
		return nil, err
	}
	return scan.skills, nil
}

// libraryScan is the raw result of one library walk: discovered skills plus
// the skl.yaml files found in non-skill folders, keyed by bundle name.
type libraryScan struct {
	skills   []Skill
	settings map[string][]string
}

func scanLibrary() (libraryScan, error) {
	scan := libraryScan{settings: map[string][]string{}}
	if err := EnsureLibrary(); err != nil {
		return scan, err
	}

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
			if name := settingsBundleName(id); name != "" && hasBundleSettings(path) {
				scan.settings[name] = append(scan.settings[name], filepath.Join(path, BundleSettingsFile))
			}
			return nil
		}
		id = legacySkillID(id)
		name := manifestSkillName(path)
		if name == "" {
			name = id
		}
		scan.skills = append(scan.skills, Skill{
			ID:       id,
			Name:     name,
			DirName:  filepath.Base(path),
			SrcPath:  path,
			External: strings.HasPrefix(id, "external/"),
			Repo:     repoNameFromSkillID(id),
		})
		// Skill folders are leaves: nested SKILL.md or skl.yaml files inside a
		// skill are part of that skill, not separate skills or bundles.
		return filepath.SkipDir
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return scan, err
	}

	sort.Slice(scan.skills, func(i, j int) bool { return scan.skills[i].ID < scan.skills[j].ID })
	return scan, nil
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

// Bundles maps each loadable bundle to the skill IDs `skl load` installs for
// it, with skl.yaml's recursive opt-in applied. See Catalog.
func Bundles() (map[string][]string, error) {
	cat, err := ReadCatalog()
	if err != nil {
		return nil, err
	}
	return cat.Bundles, nil
}

func hasSkillManifest(dir string) bool {
	return isRegularFile(dir, "SKILL.md")
}

func isRegularFile(dir, name string) bool {
	info, err := os.Lstat(filepath.Join(dir, name))
	return err == nil && info.Mode().IsRegular()
}

// manifestSkillName extracts bounded display metadata without making malformed frontmatter fatal to discovery.
func manifestSkillName(dir string) string {
	path := filepath.Join(dir, "SKILL.md")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, skillManifestReadLimit))
	if err != nil {
		return ""
	}
	body := strings.ReplaceAll(string(data), "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
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
	return cleanManifestName(manifest.Name)
}

func cleanManifestName(name string) string {
	return strings.Join(strings.Fields(name), " ")
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
