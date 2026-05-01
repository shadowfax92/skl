package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleRemoveCmd)
}

var bundleRemoveCmd = &cobra.Command{
	Use:   "remove <name> <skill...>",
	Short: "Remove skills from a folder bundle",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("folder bundles are managed by moving skill directories; symlink support is not implemented yet")
	},
}
