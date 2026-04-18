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

	skillsRoot, _ := SkillsPath()
	entries, err := os.ReadDir(skillsRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(skillsRoot, e.Name())
		if !hasSkillManifest(dir) {
			continue
		}
		out = append(out, Skill{
			ID:      e.Name(),
			DirName: e.Name(),
			SrcPath: dir,
		})
	}

	externalRoot, _ := ExternalPath()
	repoEntries, err := os.ReadDir(externalRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, repo := range repoEntries {
		if !repo.IsDir() || strings.HasPrefix(repo.Name(), ".") {
			continue
		}
		repoDir := filepath.Join(externalRoot, repo.Name())
		skillEntries, err := os.ReadDir(repoDir)
		if err != nil {
			continue
		}
		for _, s := range skillEntries {
			if !s.IsDir() || strings.HasPrefix(s.Name(), ".") {
				continue
			}
			dir := filepath.Join(repoDir, s.Name())
			if !hasSkillManifest(dir) {
				continue
			}
			out = append(out, Skill{
				ID:       repo.Name() + "/" + s.Name(),
				DirName:  s.Name(),
				SrcPath:  dir,
				External: true,
				Repo:     repo.Name(),
			})
		}
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
	bundles, err := readPersistedBundles()
	if err != nil {
		return nil, err
	}
	skills, err := Skills()
	if err != nil {
		return nil, err
	}
	assigned := make(map[string]bool, len(skills))
	for name, ids := range bundles {
		if name == ReservedInboxBundle {
			continue
		}
		for _, id := range ids {
			assigned[id] = true
		}
	}
	var inbox []string
	for _, skill := range skills {
		if assigned[skill.ID] {
			continue
		}
		inbox = append(inbox, skill.ID)
	}
	if len(inbox) > 0 {
		bundles[ReservedInboxBundle] = inbox
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
