package cmd

import (
	"fmt"
	"io"
	"slices"
	"sort"

	"skl/internal/library"
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
	Use:     "show [name...]",
	Aliases: []string{"cat"},
	Short:   "Show skills in bundles (fzf multi-picks when no names given)",
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		if len(bundles) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), style.Faint("No bundles defined."))
			return nil
		}

		names := args
		if len(names) == 0 {
			names, err = pickBundles(bundles, "show > ")
			if err != nil {
				return err
			}
		}
		if err := validateBundleNames(names, bundles); err != nil {
			return err
		}

		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		st, err := mgr.Load()
		if err != nil {
			return err
		}

		all, err := library.Skills()
		if err != nil {
			return err
		}

		writeBundleShow(cmd.OutOrStdout(), names, bundles, st, all)
		return nil
	},
}

func validateBundleNames(names []string, bundles map[string][]string) error {
	for _, name := range names {
		if _, ok := bundles[name]; !ok {
			return fmt.Errorf("bundle %q does not exist", name)
		}
	}
	return nil
}

// writeBundleShow renders each selected bundle as its own table.
func writeBundleShow(out io.Writer, names []string, bundles map[string][]string, st *state.State, all []library.Skill) {
	byID := make(map[string]library.Skill, len(all))
	for _, s := range all {
		byID[s.ID] = s
	}

	for i, name := range names {
		if i > 0 {
			fmt.Fprintln(out)
		}
		writeOneBundleShow(out, name, bundles[name], st, byID)
	}
}

func writeOneBundleShow(out io.Writer, name string, bundleSkills []string, st *state.State, byID map[string]library.Skill) {
	skills := slices.Clone(bundleSkills)
	sort.Strings(skills)

	header := name
	if name == library.ReservedInboxBundle {
		header += style.Faint("  (derived: skills in no bundle)")
	}
	fmt.Fprintln(out, style.Header(header))

	if len(skills) == 0 {
		fmt.Fprintln(out, style.Faint("  (empty)"))
		return
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

	fmt.Fprintln(out, t)
}
