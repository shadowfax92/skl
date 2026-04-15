package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:         "import",
	Annotations: map[string]string{"group": "Library:"},
	Short:       "Copy current ~/.skills/ entries into the library",
	Long: `Walks ~/.skills/ (skipping dot-prefixed entries) and copies any skill not
already in the library into ~/.config/skl/library/skills/. Idempotent — does
not modify ~/.skills/ itself, and skips skills that already exist in the
library.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := library.EnsureLibrary(); err != nil {
			return err
		}
		liveRoot, err := live.LivePath()
		if err != nil {
			return err
		}
		dirs, err := live.LoadedDirs()
		if err != nil {
			return err
		}
		skillsDir, err := library.SkillsPath()
		if err != nil {
			return err
		}

		imported, skipped := 0, 0
		for _, d := range dirs {
			src := filepath.Join(liveRoot, d)
			if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
				continue
			}
			dst := filepath.Join(skillsDir, d)
			if _, err := os.Stat(dst); err == nil {
				skipped++
				continue
			}
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("importing %s: %w", d, err)
			}
			imported++
		}

		fmt.Printf("%s %d skill(s)  %s %d (already in library)\n",
			style.OK("imported"), imported, style.Faint("skipped"), skipped)
		return nil
	},
}
