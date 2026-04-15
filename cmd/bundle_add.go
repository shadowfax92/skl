package cmd

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleAddCmd)
}

var bundleAddCmd = &cobra.Command{
	Use:   "add <name> <skill...>",
	Short: "Add skills to an existing bundle",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		skills := args[1:]

		if err := validateSkillsExist(skills); err != nil {
			return err
		}

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if _, ok := bundles[name]; !ok {
			return fmt.Errorf("bundle %q does not exist (use `skl bundle create`)", name)
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
