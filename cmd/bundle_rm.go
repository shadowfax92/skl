package cmd

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleRmCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	bundleCmd.AddCommand(bundleRmCmd)
}

var bundleRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Delete a bundle entirely",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		name := args[0]
		if err := rejectReservedBundle(name); err != nil {
			return err
		}

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if _, ok := bundles[name]; !ok {
			return fmt.Errorf("bundle %q does not exist", name)
		}

		if !yes && !confirm(fmt.Sprintf("Delete bundle %q?", name)) {
			return ErrCancelled
		}

		delete(bundles, name)
		if err := library.WriteBundles(bundles); err != nil {
			return err
		}
		fmt.Printf("%s bundle %q\n", style.OK("deleted"), name)
		return nil
	},
}
