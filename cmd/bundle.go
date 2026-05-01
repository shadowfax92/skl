package cmd

import "github.com/spf13/cobra"

var bundleCmd = &cobra.Command{
	Use:         "bundle",
	Aliases:     []string{"b"},
	Annotations: map[string]string{"group": "Library:"},
	Short:       "Manage folder bundles",
}

func init() {
	rootCmd.AddCommand(bundleCmd)
}
