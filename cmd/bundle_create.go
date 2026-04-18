package cmd

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleCreateCmd.Flags().BoolP("yes", "y", false, "Skip overwrite confirmation")
	bundleCmd.AddCommand(bundleCreateCmd)
}

var bundleCreateCmd = &cobra.Command{
	Use:   "create <name> <skill...>",
	Short: "Create or replace a bundle with the given skills",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		name := args[0]
		skills := args[1:]
		if err := rejectReservedBundle(name); err != nil {
			return err
		}

		if err := validateSkillsExist(skills); err != nil {
			return err
		}

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if _, exists := bundles[name]; exists && !yes {
			if !confirm(fmt.Sprintf("Bundle %q exists. Replace?", name)) {
				return ErrCancelled
			}
		}

		bundles[name] = skills
		if err := library.WriteBundles(bundles); err != nil {
			return err
		}
		fmt.Printf("%s bundle %q with %d skill(s)\n", style.OK("created"), name, len(skills))
		return nil
	},
}

func validateSkillsExist(ids []string) error {
	for _, id := range ids {
		if _, err := library.FindSkill(id); err != nil {
			return err
		}
	}
	return nil
}
