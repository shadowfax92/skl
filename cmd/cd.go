package cmd

import (
	"fmt"

	"skl/internal/library"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:         "cd",
	Annotations: map[string]string{"group": "Library:"},
	Short:       "Print the library path for cd",
	Long: `Print the skl library root path.

Use it from your shell with:
  cd "$(skl cd)"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := library.LibraryPath()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}
