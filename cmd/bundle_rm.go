package cmd

import (
	"fmt"
	"os"

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

		path, err := library.BundlePath(name)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("bundle %q does not exist", name)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}

		if !yes && !confirm(fmt.Sprintf("Delete bundle %q?", name)) {
			return ErrCancelled
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing empty bundle folder %q: %w", name, err)
		}
		fmt.Printf("%s bundle %q\n", style.OK("deleted"), name)
		return nil
	},
}
