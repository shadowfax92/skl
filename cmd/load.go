package cmd

import (
	"fmt"

	"skl/internal/bundle"
	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	loadCmd.Flags().StringSlice("skill", nil, "Load individual skill(s) directly (repeatable)")
	rootCmd.AddCommand(loadCmd)
}

var loadCmd = &cobra.Command{
	Use:         "load [bundle...]",
	Aliases:     []string{"l"},
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Load bundles into ~/.skills/ (fzf when no args)",
	RunE: func(cmd *cobra.Command, args []string) error {
		individualSkills, _ := cmd.Flags().GetStringSlice("skill")

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		lib, err := library.Skills()
		if err != nil {
			return err
		}

		chosen := args
		if len(chosen) == 0 && len(individualSkills) == 0 {
			chosen, err = pickBundles(bundles, "load > ")
			if err != nil {
				return err
			}
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

		totalLoaded, totalAlready := 0, 0

		for _, name := range chosen {
			skills, ok := bundles[name]
			if !ok {
				return fmt.Errorf("bundle %q not found", name)
			}
			plan, err := bundle.PlanLoad(name, skills, lib, st)
			if err != nil {
				return err
			}
			loaded, already, err := applyLoadPlan(plan, st)
			if err != nil {
				return fmt.Errorf("loading bundle %q: %w", name, err)
			}
			totalLoaded += loaded
			totalAlready += already
			fmt.Printf("%s bundle %q  %s %d new  %s %d already loaded\n",
				style.OK("loaded"), name,
				style.Faint("+"), loaded,
				style.Faint("="), already)
		}

		for _, id := range individualSkills {
			s, err := library.FindSkill(id)
			if err != nil {
				return err
			}
			synthetic := bundle.LoadPlan{
				Bundle:  "__skill__",
				Actions: []bundle.LoadAction{{Skill: *s, Already: stateHas(st, id)}},
			}
			loaded, already, err := applyLoadPlan(synthetic, st)
			if err != nil {
				return fmt.Errorf("loading skill %q: %w", id, err)
			}
			totalLoaded += loaded
			totalAlready += already
			fmt.Printf("%s skill %q\n", style.OK("loaded"), id)
		}

		if err := mgr.Save(st); err != nil {
			return err
		}
		fmt.Printf("\n%s %d new skill(s)  %s %d already loaded\n",
			style.OK("done:"), totalLoaded, style.Faint("+"), totalAlready)
		return nil
	},
}

func applyLoadPlan(plan bundle.LoadPlan, st *state.State) (loaded, already int, err error) {
	var copied []string
	for _, action := range plan.Actions {
		if action.Already {
			st.AddBundleClaim(action.Skill.ID, action.Skill.DirName, action.Skill.SrcPath, plan.Bundle)
			already++
			continue
		}
		if err := live.CopySkill(action.Skill.SrcPath, action.Skill.DirName); err != nil {
			rollbackCopies(copied)
			return 0, 0, err
		}
		copied = append(copied, action.Skill.DirName)
		st.AddBundleClaim(action.Skill.ID, action.Skill.DirName, action.Skill.SrcPath, plan.Bundle)
		loaded++
	}
	return loaded, already, nil
}

func rollbackCopies(dirs []string) {
	for _, d := range dirs {
		_ = live.RemoveSkill(d)
	}
}

func stateHas(st *state.State, id string) bool {
	_, ok := st.Loaded[id]
	return ok
}

func pickBundles(bundles map[string][]string, prompt string) ([]string, error) {
	if len(bundles) == 0 {
		return nil, fmt.Errorf("no bundles defined")
	}
	var items []picker.Item
	for name, skills := range bundles {
		items = append(items, picker.Item{
			ID:      name,
			Display: fmt.Sprintf("%s\t(%d skills)", name, len(skills)),
		})
	}
	chosen, err := picker.Pick(items, picker.Opts{Prompt: prompt, Multi: true})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return chosen, nil
}
