package cmd

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleRemoveCmd)
}

var bundleRemoveCmd = &cobra.Command{
	Use:   "remove <name> <skill...>",
	Short: "Remove skills from a bundle (no error if absent)",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		drop := args[1:]

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if _, ok := bundles[name]; !ok {
			return fmt.Errorf("bundle %q does not exist", name)
		}

		dropSet := map[string]bool{}
		for _, d := range drop {
			dropSet[d] = true
		}

		var kept []string
		removed := 0
		for _, s := range bundles[name] {
			if dropSet[s] {
				removed++
				continue
			}
			kept = append(kept, s)
		}
		bundles[name] = kept

		if err := library.WriteBundles(bundles); err != nil {
			return err
		}
		fmt.Printf("%s %d skill(s) from bundle %q\n", style.OK("removed"), removed, name)
		return nil
	},
}
