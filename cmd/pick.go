package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"skl/internal/bundle"
	"skl/internal/library"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pickCmd)
}

var pickCmd = &cobra.Command{
	Use:         "pick [bundle] [skill...]",
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Pick skills from a bundle to load",
	Long: `Pick selected skills from one bundle and load only those skills into
~/.skills/. With no skill names, pick opens fzf and shows parsed skill names
from each SKILL.md when available.`,
	Example: `  skl pick external/gstack
  skl pick external/gstack design browser`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		lib, err := library.Skills()
		if err != nil {
			return err
		}

		var bundleName string
		var skillArgs []string
		if len(args) == 0 {
			bundleName, err = pickOneBundleName(bundles, "pick bundle > ", picker.Pick)
			if err != nil {
				return err
			}
		} else {
			bundleName = args[0]
			skillArgs = args[1:]
		}

		bundleSkills, ok := bundles[bundleName]
		if !ok {
			return fmt.Errorf("bundle %q not found", bundleName)
		}

		var selected []string
		if len(skillArgs) > 0 {
			selected, err = resolveBundleSkillArgs(bundleName, bundleSkills, lib, skillArgs)
		} else {
			selected, err = pickBundleSkillIDs(bundleName, bundleSkills, lib, picker.Pick)
		}
		if err != nil {
			return err
		}

		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		if err := mgr.Lock(); err != nil {
			return err
		}
		defer mgr.Unlock()

		st, err := mgr.Load()
		if err != nil {
			return err
		}
		plan, err := bundle.PlanLoad(bundleName, selected, lib, st)
		if err != nil {
			return err
		}
		newCount, reloaded, err := applyLoadPlan(plan, st)
		if err != nil {
			return fmt.Errorf("loading selected skills from bundle %q: %w", bundleName, err)
		}
		if err := mgr.Save(st); err != nil {
			return err
		}

		total := newCount + reloaded
		fmt.Fprintf(cmd.OutOrStdout(), "%s bundle %q  %s %d selected skill(s)", style.OK("loaded"), bundleName, style.Faint("+"), total)
		if reloaded > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %d reloaded", style.Faint("reload"), reloaded)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	},
}

// pickBundleSkillIDs shows bundle skills by display name and returns canonical skill IDs.
func pickBundleSkillIDs(bundleName string, bundleSkills []string, all []library.Skill, pick pickItemsFunc) ([]string, error) {
	items, err := bundleSkillPickerItems(bundleName, bundleSkills, all)
	if err != nil {
		return nil, err
	}
	chosen, err := pick(items, picker.Opts{
		Prompt: "pick skills > ",
		Multi:  true,
		Header: fmt.Sprintf("Bundle: %s", bundleName),
	})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return chosen, nil
}

// resolveBundleSkillArgs maps bundle-local names to canonical IDs for scriptable selective loads.
func resolveBundleSkillArgs(bundleName string, bundleSkills []string, all []library.Skill, args []string) ([]string, error) {
	byID := indexSkillsByID(all)
	members, err := bundleMemberSet(bundleName, bundleSkills, byID)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var out []string
	for _, arg := range args {
		id, err := resolveBundleSkillArg(bundleName, arg, bundleSkills, byID, members)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrCancelled
	}
	return out, nil
}

func pickOneBundleName(bundles map[string][]string, prompt string, pick pickItemsFunc) (string, error) {
	if len(bundles) == 0 {
		return "", fmt.Errorf("no bundles defined")
	}
	names := make([]string, 0, len(bundles))
	for name := range bundles {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]picker.Item, 0, len(names))
	for _, name := range names {
		items = append(items, picker.Item{
			ID:      name,
			Display: fmt.Sprintf("%s  (%d skills)", name, len(bundles[name])),
		})
	}
	chosen, err := pick(items, picker.Opts{Prompt: prompt})
	if err != nil {
		return "", err
	}
	if len(chosen) == 0 {
		return "", ErrCancelled
	}
	return chosen[0], nil
}

func bundleSkillPickerItems(bundleName string, bundleSkills []string, all []library.Skill) ([]picker.Item, error) {
	byID := indexSkillsByID(all)
	if _, err := bundleMemberSet(bundleName, bundleSkills, byID); err != nil {
		return nil, err
	}

	ids := append([]string(nil), bundleSkills...)
	sort.Slice(ids, func(i, j int) bool {
		left := strings.ToLower(skillDisplayName(byID[ids[i]]))
		right := strings.ToLower(skillDisplayName(byID[ids[j]]))
		if left == right {
			return ids[i] < ids[j]
		}
		return left < right
	})

	items := make([]picker.Item, 0, len(ids))
	for _, id := range ids {
		skill := byID[id]
		items = append(items, picker.Item{
			ID:      id,
			Display: skillPickerDisplay(skill),
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("bundle %q has no skills", bundleName)
	}
	return items, nil
}

func resolveBundleSkillArg(bundleName, arg string, bundleSkills []string, byID map[string]library.Skill, members map[string]bool) (string, error) {
	if members[arg] {
		return arg, nil
	}

	localMatches := matchingBundleSkills(bundleSkills, byID, func(skill library.Skill) bool {
		return localSkillName(skill) == arg
	})
	if len(localMatches) == 1 {
		return localMatches[0], nil
	}
	if len(localMatches) > 1 {
		return "", ambiguousBundleSkillError(bundleName, arg, localMatches)
	}

	nameMatches := matchingBundleSkills(bundleSkills, byID, func(skill library.Skill) bool {
		return skill.Name != "" && skill.Name == arg
	})
	if len(nameMatches) == 1 {
		return nameMatches[0], nil
	}
	if len(nameMatches) > 1 {
		return "", ambiguousBundleSkillError(bundleName, arg, nameMatches)
	}

	return "", fmt.Errorf("skill %q not found in bundle %q", arg, bundleName)
}

func matchingBundleSkills(bundleSkills []string, byID map[string]library.Skill, matches func(library.Skill) bool) []string {
	var out []string
	for _, id := range bundleSkills {
		skill := byID[id]
		if matches(skill) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func ambiguousBundleSkillError(bundleName, arg string, matches []string) error {
	sort.Strings(matches)
	return fmt.Errorf("skill %q is ambiguous in bundle %q: %s", arg, bundleName, strings.Join(matches, ", "))
}

func bundleMemberSet(bundleName string, bundleSkills []string, byID map[string]library.Skill) (map[string]bool, error) {
	members := make(map[string]bool, len(bundleSkills))
	for _, id := range bundleSkills {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("bundle %q references unknown skill %q", bundleName, id)
		}
		members[id] = true
	}
	return members, nil
}

func indexSkillsByID(skills []library.Skill) map[string]library.Skill {
	byID := make(map[string]library.Skill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}
	return byID
}

func localSkillName(skill library.Skill) string {
	if skill.DirName != "" {
		return skill.DirName
	}
	return filepath.Base(filepath.FromSlash(skill.ID))
}

func skillDisplayName(skill library.Skill) string {
	if skill.Name != "" {
		return skill.Name
	}
	return skill.ID
}

func skillPickerDisplay(skill library.Skill) string {
	name := skillDisplayName(skill)
	if name == skill.ID {
		return skill.ID
	}
	return fmt.Sprintf("%s  (%s)", name, skill.ID)
}
