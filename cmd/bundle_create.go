package cmd

import (
	"fmt"
	"os"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleCreateCmd)
}

var bundleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a folder bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := rejectReservedBundle(name); err != nil {
			return err
		}

		path, err := library.BundlePath(name)
		if err != nil {
			return err
		}
		if info, err := os.Stat(path); err == nil {
			if !info.IsDir() {
				return fmt.Errorf("%s exists and is not a directory", path)
			}
			fmt.Printf("%s bundle %q\n", style.Faint("exists"), name)
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating bundle %q: %w", name, err)
		}
		fmt.Printf("%s bundle %q\n", style.OK("created"), name)
		return nil
	},
}
