package cmd

import (
	"fmt"
	"time"

	"skl/internal/gitlib"
	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:         "sync",
	Annotations: map[string]string{"group": "Sync:"},
	Short:       "Commit, pull --rebase, and push the library",
	RunE: func(cmd *cobra.Command, args []string) error {
		libDir, err := library.LibraryPath()
		if err != nil {
			return err
		}
		if err := library.EnsureLibrary(); err != nil {
			return err
		}
		if !gitlib.IsRepo(libDir) {
			return fmt.Errorf("library is not a git repo (run `skl remote <url>` first)")
		}
		if url, _ := gitlib.RemoteURL(libDir); url == "" {
			return fmt.Errorf("no `origin` remote (run `skl remote <url>` first)")
		}

		msg := fmt.Sprintf("skl sync %s", time.Now().UTC().Format(time.RFC3339))
		if err := gitlib.AddCommit(libDir, msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		if err := gitlib.PullRebase(libDir); err != nil {
			return fmt.Errorf("pull --rebase: %w", err)
		}
		if err := gitlib.Push(libDir); err != nil {
			return fmt.Errorf("push: %w", err)
		}
		fmt.Println(style.OK("synced"))
		return nil
	},
}
