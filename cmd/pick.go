package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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

var pickSkillItems pickItemsFunc = picker.Pick

var pickCmd = &cobra.Command{
	Use:         "pick [bundle] [skill...]",
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Pick skills to load",
	Long: `Pick selected skills and load only those skills into ~/.skills/.
With no arguments, pick opens fzf over every loadable SKILL.md. With a bundle
argument, pick stays scoped to that bundle.`,
	Example: `  skl pick external/gstack
  skl pick
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
			selected, err := pickAllSkillIDs(lib, pickSkillItems)
			if err != nil {
				return err
			}
			grouped, err := groupSkillIDsByBundle(selected, bundles)
			if err != nil {
				return err
			}
			newCount, reloaded, err := loadSelectedSkillGroups(grouped, lib)
			if err != nil {
				return fmt.Errorf("loading selected skills: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s selected skills  %s %d selected skill(s)", style.OK("loaded"), style.Faint("+"), newCount)
			if reloaded > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %d reloaded", style.Faint("reload"), reloaded)
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
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
			selected, err = pickBundleSkillIDs(bundleName, bundleSkills, lib, pickSkillItems)
		}
		if err != nil {
			return err
		}

		newCount, reloaded, err := loadSelectedSkillGroups(map[string][]string{bundleName: selected}, lib)
		if err != nil {
			return fmt.Errorf("loading selected skills from bundle %q: %w", bundleName, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s bundle %q  %s %d selected skill(s)", style.OK("loaded"), bundleName, style.Faint("+"), newCount)
		if reloaded > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %d reloaded", style.Faint("reload"), reloaded)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	},
}

// pickAllSkillIDs lets the default picker select canonical IDs across every discovered skill.
func pickAllSkillIDs(all []library.Skill, pick pickItemsFunc) ([]string, error) {
	items := allSkillPickerItems(all)
	chosen, err := pick(items, picker.Opts{
		Prompt: "pick skills > ",
		Multi:  true,
		Header: "All skills",
	})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return validatePickedSkillIDs(chosen, all)
}

// pickBundleSkillIDs shows bundle skills by display name and returns canonical skill IDs.
func pickBundleSkillIDs(bundleName string, bundleSkills []string, all []library.Skill, pick pickItemsFunc) ([]string, error) {
	items, err := bundleSkillPickerItems(bundleName, bundleSkills, all)
	if err != nil {
		return nil, err
	}
	byID := indexSkillsByID(all)
	members, err := bundleMemberSet(bundleName, bundleSkills, byID)
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
	return validatePickedBundleSkillIDs(bundleName, chosen, members)
}

// loadSelectedSkillGroups applies selected IDs through the existing per-bundle load planner.
func loadSelectedSkillGroups(grouped map[string][]string, all []library.Skill) (newCount, reloaded int, err error) {
	mgr, err := state.NewManager()
	if err != nil {
		return 0, 0, err
	}
	if err := mgr.Lock(); err != nil {
		return 0, 0, err
	}
	defer mgr.Unlock()

	st, err := mgr.Load()
	if err != nil {
		return 0, 0, err
	}

	bundleNames := make([]string, 0, len(grouped))
	for name := range grouped {
		bundleNames = append(bundleNames, name)
	}
	sort.Strings(bundleNames)

	for _, name := range bundleNames {
		plan, err := bundle.PlanLoad(name, grouped[name], all, st)
		if err != nil {
			return 0, 0, err
		}
		groupNew, groupReloaded, err := applyLoadPlan(plan, st)
		if err != nil {
			return 0, 0, err
		}
		newCount += groupNew
		reloaded += groupReloaded
	}

	if err := mgr.Save(st); err != nil {
		return 0, 0, err
	}
	return newCount, reloaded, nil
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

func allSkillPickerItems(all []library.Skill) []picker.Item {
	skills := append([]library.Skill(nil), all...)
	sort.Slice(skills, func(i, j int) bool {
		left := strings.ToLower(skillDisplayName(skills[i]))
		right := strings.ToLower(skillDisplayName(skills[j]))
		if left == right {
			return skills[i].ID < skills[j].ID
		}
		return left < right
	})

	items := make([]picker.Item, 0, len(skills))
	for _, skill := range skills {
		items = append(items, picker.Item{
			ID:      skill.ID,
			Display: skillPickerDisplay(skill),
		})
	}
	return items
}

func resolveBundleSkillArg(bundleName, arg string, bundleSkills []string, byID map[string]library.Skill, members map[string]bool) (string, error) {
	if members[arg] {
		return arg, nil
	}

	localMatches := matchingBundleSkills(bundleSkills, byID, func(skill library.Skill) bool {
		return localSkillName(skill) == arg
	})
	nameMatches := matchingBundleSkills(bundleSkills, byID, func(skill library.Skill) bool {
		return skill.Name != "" && skill.Name == arg
	})
	matches := uniqueSortedIDs(append(localMatches, nameMatches...))
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", ambiguousBundleSkillError(bundleName, arg, matches)
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
	if name := singleLinePickerText(skill.Name); name != "" {
		return name
	}
	return singleLinePickerText(skill.ID)
}

func skillPickerDisplay(skill library.Skill) string {
	name := skillDisplayName(skill)
	id := singleLinePickerText(skill.ID)
	if name == id {
		return id
	}
	return fmt.Sprintf("%s  (%s)", name, id)
}

func validatePickedBundleSkillIDs(bundleName string, chosen []string, members map[string]bool) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, id := range chosen {
		if !members[id] {
			return nil, fmt.Errorf("selected skill %q is not in bundle %q", id, bundleName)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

func validatePickedSkillIDs(chosen []string, all []library.Skill) ([]string, error) {
	byID := indexSkillsByID(all)
	seen := map[string]bool{}
	var out []string
	for _, id := range chosen {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("selected skill %q is not loadable", id)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

func groupSkillIDsByBundle(selected []string, bundles map[string][]string) (map[string][]string, error) {
	bundleBySkill := map[string]string{}
	bundleNames := make([]string, 0, len(bundles))
	for name := range bundles {
		bundleNames = append(bundleNames, name)
	}
	sort.Strings(bundleNames)
	for _, name := range bundleNames {
		for _, id := range bundles[name] {
			bundleBySkill[id] = name
		}
	}

	grouped := map[string][]string{}
	for _, id := range selected {
		name, ok := bundleBySkill[id]
		if !ok {
			return nil, fmt.Errorf("selected skill %q is not in a bundle", id)
		}
		grouped[name] = append(grouped[name], id)
	}
	return grouped, nil
}

func singleLinePickerText(text string) string {
	var b strings.Builder
	for _, r := range text {
		if unicode.IsControl(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func uniqueSortedIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
