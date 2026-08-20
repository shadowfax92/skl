package cmd

import (
	"fmt"
	"os"

	"skl/internal/bundle"
	"skl/internal/library"
	"skl/internal/live"
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

		plan := bundle.UnloadPlan{Bundle: bundleName}
		for _, id := range selected {
			plan.Actions = append(plan.Actions, bundle.UnloadAction{SkillID: id, Entry: st.Loaded[id]})
		}
		removed, kept, applied, err := applyUnpickPlan(plan, st)
		if err != nil {
			return err
		}
		if err := mgr.Save(st); err != nil {
			return rollbackUnpickError(err, applied, st)
		}
		cleanupUnpickBackups(applied)

		if bundleName == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %d selected skill(s)\n", style.OK("unloaded"), removed)
			return nil
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

type unpickRollback struct {
	skillID    string
	entry      state.LoadEntry
	dirName    string
	backupPath string
}

func applyUnpickPlan(plan bundle.UnloadPlan, st *state.State) (removed, kept int, applied []unpickRollback, err error) {
	for _, action := range plan.Actions {
		entry, ok := st.Loaded[action.SkillID]
		if !ok {
			return 0, 0, nil, rollbackUnpickError(fmt.Errorf("skill %q not loaded", action.SkillID), applied, st)
		}
		step := unpickRollback{skillID: action.SkillID, entry: entry, dirName: entry.DirName}
		if plan.Bundle != "" && len(entry.Bundles) > 1 {
			applied = append(applied, step)
			st.RemoveBundleClaim(action.SkillID, plan.Bundle)
			kept++
			continue
		}

		exists, err := live.SkillExists(entry.DirName)
		if err != nil {
			return 0, 0, nil, rollbackUnpickError(err, applied, st)
		}
		if exists {
			step.backupPath, err = backupLiveSkill(entry.DirName)
			if err != nil {
				return 0, 0, nil, rollbackUnpickError(err, applied, st)
			}
		}
		applied = append(applied, step)
		st.RemoveLoaded(action.SkillID)
		removed++
	}
	return removed, kept, applied, nil
}

func rollbackUnpickError(cause error, applied []unpickRollback, st *state.State) error {
	if err := rollbackUnpick(applied, st); err != nil {
		return fmt.Errorf("%w (rollback failed: %v)", cause, err)
	}
	return cause
}

func rollbackUnpick(applied []unpickRollback, st *state.State) error {
	var firstErr error
	for i := len(applied) - 1; i >= 0; i-- {
		step := applied[i]
		st.Loaded[step.skillID] = step.entry
		if err := restoreLiveSkill(step.dirName, step.backupPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func cleanupUnpickBackups(applied []unpickRollback) {
	for _, step := range applied {
		if step.backupPath != "" {
			_ = os.RemoveAll(step.backupPath)
		}
	}
}
