package cmd

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

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
		cat, err := library.ReadCatalog()
		if err != nil {
			return err
		}
		bundles := cat.Bundles
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

		writeBundleShow(cmd.OutOrStdout(), names, cat, st)
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

// writeBundleShow renders each selected bundle as its own table, listing the
// skills `skl load` would install (nested ones included for recursive bundles).
func writeBundleShow(out io.Writer, names []string, cat *library.Catalog, st *state.State) {
	byID := make(map[string]library.Skill, len(cat.Skills))
	for _, s := range cat.Skills {
		byID[s.ID] = s
	}

	for i, name := range names {
		if i > 0 {
			fmt.Fprintln(out)
		}
		writeOneBundleShow(out, name, cat, st, byID)
	}
}

func writeOneBundleShow(out io.Writer, name string, cat *library.Catalog, st *state.State, byID map[string]library.Skill) {
	skills := slices.Clone(cat.Bundles[name])
	sort.Strings(skills)

	header := name
	if name == library.ReservedInboxBundle {
		header += style.Faint("  (derived: skills in no bundle)")
	}
	if cat.Recursive[name] {
		note := "recursive"
		if nested := cat.Nested(name); len(nested) > 0 {
			note += ": includes " + strings.Join(nested, ", ")
		}
		header += style.Faint("  (" + note + ")")
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
