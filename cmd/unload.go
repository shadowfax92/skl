package cmd

import (
	"fmt"

	"skl/internal/bundle"
	"skl/internal/live"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	unloadCmd.Flags().Bool("all", false, "Unload every skill loaded by skl")
	unloadCmd.Flags().StringSlice("skill", nil, "Unload individual skill(s) by ID (repeatable)")
	rootCmd.AddCommand(unloadCmd)
}

var unloadCmd = &cobra.Command{
	Use:         "unload [bundle...]",
	Aliases:     []string{"u"},
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Unload bundles from ~/.skills/ (fzf when no args)",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		individualSkills, _ := cmd.Flags().GetStringSlice("skill")

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

		if all {
			return unloadAll(mgr, st)
		}

		if len(individualSkills) > 0 {
			for _, id := range individualSkills {
				if err := unloadSkillEntirely(st, id); err != nil {
					return err
				}
			}
			fmt.Printf("%s %d skill(s)\n", style.OK("unloaded"), len(individualSkills))
			return mgr.Save(st)
		}

		chosen := args
		if len(chosen) == 0 {
			loaded := st.LoadedBundles()
			if len(loaded) == 0 {
				dirs, _ := live.LoadedDirs()
				if len(dirs) > 0 {
					fmt.Printf("%s %d untracked skill(s) in ~/.skills/ — run %s to remove them.\n",
						style.Faint("Nothing loaded by skl, but"),
						len(dirs),
						style.Cmd("skl prune"))
					return nil
				}
				fmt.Println(style.Faint("Nothing to unload."))
				return nil
			}
			chosen, err = pickLoadedBundles(loaded)
			if err != nil {
				return err
			}
		}

		for _, name := range chosen {
			plan := bundle.PlanUnload(name, st)
			if len(plan.Actions) == 0 {
				fmt.Printf("%s bundle %q (not loaded)\n", style.Faint("skip"), name)
				continue
			}
			removed, kept := applyUnloadPlan(plan, st)
			fmt.Printf("%s bundle %q  %s %d removed  %s %d kept (other bundles)\n",
				style.OK("unloaded"), name,
				style.Faint("-"), removed,
				style.Faint("="), kept)
		}

		return mgr.Save(st)
	},
}

func applyUnloadPlan(plan bundle.UnloadPlan, st *state.State) (removed, kept int) {
	for _, action := range plan.Actions {
		orphan := st.RemoveBundleClaim(action.SkillID, plan.Bundle)
		if orphan {
			if err := live.RemoveSkill(action.Entry.DirName); err == nil {
				st.RemoveLoaded(action.SkillID)
				removed++
			}
		} else {
			kept++
		}
	}
	return removed, kept
}

func unloadAll(mgr *state.StateManager, st *state.State) error {
	count := 0
	for id, entry := range st.Loaded {
		if err := live.RemoveSkill(entry.DirName); err != nil {
			return err
		}
		st.RemoveLoaded(id)
		count++
	}
	fmt.Printf("%s %d skill(s)\n", style.OK("unloaded all:"), count)
	return mgr.Save(st)
}

func unloadSkillEntirely(st *state.State, id string) error {
	entry, ok := st.Loaded[id]
	if !ok {
		return fmt.Errorf("skill %q not loaded", id)
	}
	if err := live.RemoveSkill(entry.DirName); err != nil {
		return err
	}
	st.RemoveLoaded(id)
	return nil
}

func pickLoadedBundles(bundles []string) ([]string, error) {
	var items []picker.Item
	for _, b := range bundles {
		items = append(items, picker.Item{ID: b, Display: b})
	}
	chosen, err := picker.Pick(items, picker.Opts{Prompt: "unload > ", Multi: true})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return chosen, nil
}
