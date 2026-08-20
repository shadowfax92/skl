package cmd

import (
	"fmt"

	"skl/internal/bundle"
	"skl/internal/library"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unpickCmd)
}

var unpickSkillItems pickItemsFunc = picker.Pick

var unpickCmd = &cobra.Command{
	Use:         "unpick [bundle] [skill...]",
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Pick loaded skills to unload",
	Long: `Pick selected skills to unload from ~/.skills/.
With no arguments, unpick opens fzf over every loaded skill. With a bundle
argument, unpick removes only that bundle's claim from selected skills.`,
	Example: `  skl unpick
  skl unpick shadowfax
  skl unpick shadowfax sf-quick sf-auto`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		if len(st.Loaded) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), style.Faint("Nothing to unpick."))
			return nil
		}

		var bundleName string
		var selected []string
		if len(args) == 0 {
			selected, err = pickLoadedSkillIDs("", st, unpickSkillItems)
		} else {
			bundleName = args[0]
			if len(args) == 1 {
				selected, err = pickLoadedSkillIDs(bundleName, st, unpickSkillItems)
			} else {
				selected, err = resolveLoadedBundleSkillArgs(bundleName, st, args[1:])
			}
		}
		if err != nil {
			return err
		}

		if bundleName == "" {
			for _, id := range selected {
				if err := unloadSkillEntirely(st, id); err != nil {
					return err
				}
			}
			if err := mgr.Save(st); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %d selected skill(s)\n", style.OK("unloaded"), len(selected))
			return nil
		}

		plan := bundle.UnloadPlan{Bundle: bundleName}
		for _, id := range selected {
			plan.Actions = append(plan.Actions, bundle.UnloadAction{SkillID: id, Entry: st.Loaded[id]})
		}
		removed, kept := applyUnloadPlan(plan, st)
		if err := mgr.Save(st); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s bundle %q  %s %d removed  %s %d kept (other bundles)\n",
			style.OK("unpicked"), bundleName,
			style.Faint("-"), removed,
			style.Faint("="), kept)
		return nil
	},
}

func pickLoadedSkillIDs(bundleName string, st *state.State, pick pickItemsFunc) ([]string, error) {
	loaded := loadedSkills(st, bundleName)
	if len(loaded) == 0 {
		if bundleName == "" {
			return nil, fmt.Errorf("no skills loaded by skl")
		}
		return nil, fmt.Errorf("bundle %q has no loaded skills", bundleName)
	}
	items := allSkillPickerItems(loaded)
	header := "Loaded skills"
	if bundleName != "" {
		header = fmt.Sprintf("Bundle: %s", bundleName)
	}
	chosen, err := pick(items, picker.Opts{Prompt: "unpick skills > ", Multi: true, Header: header})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	if bundleName == "" {
		return validatePickedSkillIDs(chosen, loaded)
	}
	members := make(map[string]bool, len(loaded))
	for _, skill := range loaded {
		members[skill.ID] = true
	}
	return validatePickedBundleSkillIDs(bundleName, chosen, members)
}

func resolveLoadedBundleSkillArgs(bundleName string, st *state.State, args []string) ([]string, error) {
	loaded := loadedSkills(st, bundleName)
	if len(loaded) == 0 {
		return nil, fmt.Errorf("bundle %q has no loaded skills", bundleName)
	}
	ids := make([]string, 0, len(loaded))
	for _, skill := range loaded {
		ids = append(ids, skill.ID)
	}
	return resolveBundleSkillArgs(bundleName, ids, loaded, args)
}

func loadedSkills(st *state.State, bundleName string) []library.Skill {
	loaded := make([]library.Skill, 0, len(st.Loaded))
	for id, entry := range st.Loaded {
		if bundleName != "" && !hasBundleClaim(entry, bundleName) {
			continue
		}
		loaded = append(loaded, library.Skill{ID: id, DirName: entry.DirName})
	}
	return loaded
}

func hasBundleClaim(entry state.LoadEntry, bundleName string) bool {
	for _, claim := range entry.Bundles {
		if claim == bundleName {
			return true
		}
	}
	return false
}
