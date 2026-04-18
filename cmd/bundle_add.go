package cmd

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/picker"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleAddCmd)
}

var bundleAddCmd = &cobra.Command{
	Use:   "add <name> [skill...]",
	Short: "Add skills to an existing bundle (fzf-picks when no skills given)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		skills := args[1:]
		if err := rejectReservedBundle(name); err != nil {
			return err
		}

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if _, ok := bundles[name]; !ok {
			return fmt.Errorf("bundle %q does not exist (use `skl bundle create`)", name)
		}

		if len(skills) == 0 {
			skills, err = pickSkillsNotIn(bundles[name], "add to "+name+" > ")
			if err != nil {
				return err
			}
		}
		if err := validateSkillsExist(skills); err != nil {
			return err
		}

		merged := append([]string{}, bundles[name]...)
		merged = append(merged, skills...)
		bundles[name] = merged

		if err := library.WriteBundles(bundles); err != nil {
			return err
		}
		fmt.Printf("%s %d skill(s) to bundle %q\n", style.OK("added"), len(skills), name)
		return nil
	},
}

func pickSkillsNotIn(existing []string, prompt string) ([]string, error) {
	all, err := library.Skills()
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, e := range existing {
		have[e] = true
	}
	var items []picker.Item
	for _, s := range all {
		if have[s.ID] {
			continue
		}
		items = append(items, picker.Item{ID: s.ID, Display: s.ID})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no other skills available")
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
