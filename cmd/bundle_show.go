package cmd

import (
	"fmt"
	"slices"
	"sort"

	"skl/internal/library"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

func init() {
	bundleCmd.AddCommand(bundleShowCmd)
}

var bundleShowCmd = &cobra.Command{
	Use:     "show [name]",
	Aliases: []string{"cat"},
	Short:   "Show skills in a bundle (fzf-picks when no name given)",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if len(bundles) == 0 {
			fmt.Println(style.Faint("No bundles defined."))
			return nil
		}

		var name string
		if len(args) == 1 {
			name = args[0]
			if _, ok := bundles[name]; !ok {
				return fmt.Errorf("bundle %q does not exist", name)
			}
		} else {
			name, err = pickBundle(bundles, "show > ")
			if err != nil {
				return err
			}
		}

		skills := slices.Clone(bundles[name])
		sort.Strings(skills)

		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		st, err := mgr.Load()
		if err != nil {
			return err
		}

		header := name
		if name == library.ReservedInboxBundle {
			header += style.Faint("  (derived: skills in no bundle)")
		}
		fmt.Println(style.Header(header))

		if len(skills) == 0 {
			fmt.Println(style.Faint("  (empty)"))
			return nil
		}

		all, err := library.Skills()
		if err != nil {
			return err
		}
		byID := make(map[string]library.Skill, len(all))
		for _, s := range all {
			byID[s.ID] = s
		}

		var rows [][]string
		for _, id := range skills {
			mark := style.Faint("—")
			if _, ok := st.Loaded[id]; ok {
				mark = style.OK("loaded")
			}
			src := style.Faint("local")
			if s, ok := byID[id]; ok && s.External {
				src = style.Faint("ext: " + s.Repo)
			} else if !ok {
				src = style.Warn("missing")
			}
			rows = append(rows, []string{id, mark, src})
		}

		t := table.New().
			Border(lipgloss.HiddenBorder()).
			Headers("SKILL", "STATUS", "SOURCE").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				s := lipgloss.NewStyle().PaddingRight(2)
				if row == table.HeaderRow {
					return s.Bold(true).Faint(true)
				}
				return s
			})

		fmt.Println(t)
		return nil
	},
}

func pickBundle(bundles map[string][]string, prompt string) (string, error) {
	names := make([]string, 0, len(bundles))
	for n := range bundles {
		names = append(names, n)
	}
	sort.Strings(names)

	items := make([]picker.Item, 0, len(names))
	for _, n := range names {
		items = append(items, picker.Item{
			ID:      n,
			Display: fmt.Sprintf("%s\t(%d)", n, len(bundles[n])),
		})
	}
	chosen, err := picker.Pick(items, picker.Opts{Prompt: prompt})
	if err != nil {
		return "", err
	}
	if len(chosen) == 0 {
		return "", ErrCancelled
	}
	return chosen[0], nil
}
