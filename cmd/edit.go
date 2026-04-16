package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"skl/internal/library"
	"skl/internal/picker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:         "edit [skill]",
	Aliases:     []string{"e"},
	Annotations: map[string]string{"group": "Library:"},
	Short:       "Open a skill in $EDITOR, or print the library path",
	Long: `Open a skill's SKILL.md in $EDITOR for quick edits.

  skl edit              print the library skills path (use with cd)
  skl edit <skill>      open library/skills/<skill>/SKILL.md in $EDITOR
  skl edit --pick       fzf-pick a skill to edit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			p, err := library.SkillsPath()
			if err != nil {
				return err
			}
			fmt.Println(p)
			return nil
		}

		skillID := args[0]
		s, err := library.FindSkill(skillID)
		if err != nil {
			return err
		}
		manifest := filepath.Join(s.SrcPath, "SKILL.md")
		if _, err := os.Stat(manifest); err != nil {
			return fmt.Errorf("no SKILL.md at %s", manifest)
		}

		editor := resolveEditor()
		c := exec.Command(editor, manifest)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	editPickCmd := &cobra.Command{
		Use:   "pick",
		Short: "fzf-pick a skill to edit",
		RunE: func(cmd *cobra.Command, args []string) error {
			skills, err := library.Skills()
			if err != nil {
				return err
			}
			var items []picker.Item
			for _, s := range skills {
				items = append(items, picker.Item{ID: s.ID, Display: s.ID})
			}
			chosen, err := picker.Pick(items, picker.Opts{Prompt: "edit > "})
			if err != nil {
				return err
			}
			if len(chosen) == 0 {
				return ErrCancelled
			}
			s, err := library.FindSkill(chosen[0])
			if err != nil {
				return err
			}
			manifest := filepath.Join(s.SrcPath, "SKILL.md")
			editor := resolveEditor()
			c := exec.Command(editor, manifest)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	editCmd.AddCommand(editPickCmd)
}
