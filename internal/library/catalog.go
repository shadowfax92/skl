package library

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// BundleSettingsFile is the optional per-bundle settings file at a bundle
// folder's root. It sits outside every skill folder, so live.CopySkill never
// carries it into ~/.skills/, and it syncs with the rest of the library.
const BundleSettingsFile = "skl.yaml"

const bundleSettingsReadLimit = 64 * 1024

// bundleSettings is the skl.yaml schema. Decoding is strict so a misspelled
// key fails loudly instead of silently loading a smaller bundle.
type bundleSettings struct {
	// Recursive makes the bundle include every bundle nested below its folder.
	Recursive bool `yaml:"recursive"`
}

// Catalog is one consistent read of the library: every skill, every loadable
// bundle, and which bundles opted into their nested bundles via skl.yaml.
// Commands read it once and resolve bundle names against it, so display
// (ls, bundle show, pickers) and loading agree on what a bundle contains.
type Catalog struct {
	Skills []Skill
	// Bundles maps each bundle to the skill IDs `skl load <bundle>` installs.
	// A plain bundle holds the skills directly inside its folder; a recursive
	// one holds every skill anywhere below it, so one skill can appear under
	// several bundles (sf/ref/x is in both "sf/ref" and a recursive "sf").
	Bundles map[string][]string
	// Recursive marks bundles whose skl.yaml says `recursive: true`.
	Recursive map[string]bool
}

// ReadCatalog walks the library once and applies every bundle's skl.yaml.
// A malformed skl.yaml is an error naming the file; Skills() alone never
// parses settings, so skill-only commands are immune to it.
func ReadCatalog() (*Catalog, error) {
	scan, err := scanLibrary()
	if err != nil {
		return nil, err
	}
	cat := &Catalog{
		Skills:    scan.skills,
		Bundles:   folderBundles(scan.skills),
		Recursive: map[string]bool{},
	}

	names := make([]string, 0, len(scan.settings))
	for name := range scan.settings {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		recursive := false
		// Legacy skills/<b> and top-level <b> both map to bundle <b>; either
		// folder's skl.yaml can opt the shared bundle in.
		for _, path := range scan.settings[name] {
			settings, err := readBundleSettings(path)
			if err != nil {
				return nil, err
			}
			recursive = recursive || settings.Recursive
		}
		if !recursive {
			continue
		}
		// A recursive folder with no skills of its own still becomes a
		// bundle when something is nested below it; with nothing below it
		// there is nothing to load and it stays out of the catalog.
		if ids := cat.skillsUnder(name); len(ids) > 0 {
			cat.Bundles[name] = ids
			cat.Recursive[name] = true
		}
	}
	return cat, nil
}

// Resolve returns the skill IDs `skl load [--recursive] <name>` installs.
// Without the flag it is the catalog's membership (skl.yaml already applied).
// With it, any library folder widens to every skill below it, including a
// folder that only holds nested bundles; inbox has no folder and stays as is.
func (c *Catalog) Resolve(name string, recursive bool) ([]string, error) {
	if recursive && name != ReservedInboxBundle {
		if ids := c.skillsUnder(name); len(ids) > 0 {
			return ids, nil
		}
	} else if ids, ok := c.Bundles[name]; ok {
		return ids, nil
	}
	return nil, fmt.Errorf("bundle %q not found", name)
}

// Nested lists the bundles inside name's folder at any depth, sorted.
func (c *Catalog) Nested(name string) []string {
	prefix := name + "/"
	var out []string
	for b := range c.Bundles {
		if strings.HasPrefix(b, prefix) {
			out = append(out, b)
		}
	}
	sort.Strings(out)
	return out
}

// skillsUnder relies on discovery stopping at skill folders: every ID below
// the prefix is either a direct skill or a nested bundle's skill.
func (c *Catalog) skillsUnder(name string) []string {
	prefix := name + "/"
	var out []string
	for _, s := range c.Skills {
		if strings.HasPrefix(s.ID, prefix) {
			out = append(out, s.ID)
		}
	}
	return out
}

// HomeBundle is the folder bundle a skill sits in directly, ignoring any
// recursive bundle that also includes it. Root-level skills belong to inbox.
func HomeBundle(skillID string) string {
	if parent := pathDir(skillID); parent != "" {
		return parent
	}
	return ReservedInboxBundle
}

// folderBundles groups skills by their direct parent folder: the membership
// every bundle has before skl.yaml widens it.
func folderBundles(skills []Skill) map[string][]string {
	bundles := make(map[string][]string)
	for _, skill := range skills {
		home := HomeBundle(skill.ID)
		bundles[home] = append(bundles[home], skill.ID)
	}
	for name := range bundles {
		sort.Strings(bundles[name])
	}
	return bundles
}

// settingsBundleName maps a library folder holding skl.yaml to the bundle it
// configures, using the same legacy skills/ stripping as skill IDs. The
// legacy skills/ root itself is inbox storage, not a bundle.
func settingsBundleName(rel string) string {
	if rel == "skills" {
		return ""
	}
	return legacySkillID(rel)
}

// hasBundleSettings ignores symlinks, matching how SKILL.md symlinks are ignored.
func hasBundleSettings(dir string) bool {
	return isRegularFile(dir, BundleSettingsFile)
}

func readBundleSettings(path string) (bundleSettings, error) {
	var settings bundleSettings
	file, err := os.Open(path)
	if err != nil {
		return settings, fmt.Errorf("reading %s: %w", path, err)
	}
	defer file.Close()

	dec := yaml.NewDecoder(io.LimitReader(file, bundleSettingsReadLimit))
	dec.KnownFields(true)
	if err := dec.Decode(&settings); err != nil && !errors.Is(err, io.EOF) {
		return bundleSettings{}, fmt.Errorf("parsing %s (supported setting: recursive: true|false): %w", path, err)
	}
	return settings, nil
}
