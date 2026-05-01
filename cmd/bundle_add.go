package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleAddCmd)
}

var bundleAddCmd = &cobra.Command{
	Use:   "add <name> [skill...]",
	Short: "Add skills to a folder bundle",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("folder bundles are managed by moving skill directories; symlink support is not implemented yet")
	},
}
