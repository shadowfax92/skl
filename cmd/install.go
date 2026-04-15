package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skl/internal/gitlib"
	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	installCmd.Flags().String("bundle", "", "Add the imported skills to this bundle (creates if absent)")
	installCmd.Flags().String("name", "", "Override the directory name for the cloned repo")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:         "install <git-url>",
	Annotations: map[string]string{"group": "Library:"},
	Short:       "Clone a remote skills repo into library/external/",
	Args:        cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundleName, _ := cmd.Flags().GetString("bundle")
		nameOverride, _ := cmd.Flags().GetString("name")
		url := args[0]

		repoName := nameOverride
		if repoName == "" {
			repoName = repoNameFromURL(url)
		}
		if repoName == "" {
			return fmt.Errorf("could not derive a repo name from %q (use --name)", url)
		}

		extDir, err := library.ExternalPath()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(extDir, 0o755); err != nil {
			return err
		}
		dest := filepath.Join(extDir, repoName)
		if err := gitlib.Clone(url, dest); err != nil {
			return err
		}

		var added []string
		entries, err := os.ReadDir(dest)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if _, err := os.Stat(filepath.Join(dest, e.Name(), "SKILL.md")); err != nil {
				continue
			}
			added = append(added, repoName+"/"+e.Name())
		}

		fmt.Printf("%s %d skill(s) from %s\n", style.OK("installed"), len(added), repoName)

		if bundleName != "" && len(added) > 0 {
			bundles, err := library.Bundles()
			if err != nil {
				return err
			}
			merged := append([]string{}, bundles[bundleName]...)
			merged = append(merged, added...)
			bundles[bundleName] = merged
			if err := library.WriteBundles(bundles); err != nil {
				return err
			}
			fmt.Printf("%s skills to bundle %q\n", style.OK("added"), bundleName)
		}
		return nil
	},
}

func repoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")
	if idx := strings.LastIndexAny(url, "/:"); idx >= 0 {
		return url[idx+1:]
	}
	return url
}
