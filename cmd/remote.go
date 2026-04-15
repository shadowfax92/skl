package cmd

import (
	"fmt"

	"skl/internal/config"
	"skl/internal/gitlib"
	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(remoteCmd)
}

var remoteCmd = &cobra.Command{
	Use:         "remote [git-url]",
	Annotations: map[string]string{"group": "Sync:"},
	Short:       "Set or show the git remote for the library",
	Args:        cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		libDir, err := library.LibraryPath()
		if err != nil {
			return err
		}
		if err := library.EnsureLibrary(); err != nil {
			return err
		}

		if len(args) == 0 {
			if !gitlib.IsRepo(libDir) {
				fmt.Println(style.Faint("Library is not a git repo. Set a remote with: skl remote <url>"))
				return nil
			}
			url, _ := gitlib.RemoteURL(libDir)
			if url == "" {
				fmt.Println(style.Faint("No `origin` remote configured."))
				return nil
			}
			fmt.Println(url)
			return nil
		}

		url := args[0]
		if !gitlib.IsRepo(libDir) {
			if err := gitlib.Init(libDir); err != nil {
				return err
			}
		}
		if err := gitlib.SetRemote(libDir, url); err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Sync.Remote = url
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("%s remote: %s\n", style.OK("set"), url)
		return nil
	},
}
