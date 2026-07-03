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
	Short:       "Change to the library path with shell integration",
	Long: `Change to the skl library root when shell integration is loaded.

Install the wrapper once:
  eval "$(skl init zsh)"

Without the wrapper, external commands cannot change the parent shell, so skl cd
prints the path for command substitution:
  cd "$(skl cd)"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := library.EnsureLibrary(); err != nil {
			return err
		}
		path, err := library.LibraryPath()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}
