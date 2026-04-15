package cmd

import (
	"fmt"
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
	lsCmd.Flags().Bool("skills", false, "List all skills (instead of bundles)")
	rootCmd.AddCommand(lsCmd)
}

var lsCmd = &cobra.Command{
	Use:         "ls",
	Aliases:     []string{"list"},
	Annotations: map[string]string{"group": "Inspect:"},
	Short:       "List bundles (or `--skills` for individual skills)",
	RunE: func(cmd *cobra.Command, args []string) error {
		showSkills, _ := cmd.Flags().GetBool("skills")

		bundles, err := library.Bundles()
		if err != nil {
			return err
		}
		skills, err := library.Skills()
		if err != nil {
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

		if len(skills) == 0 && len(bundles) == 0 {
			fmt.Println(style.Faint("No skills in library yet."))
			fmt.Println(style.Faint(`Run "skl import" to seed from ~/.skills/, or`))
			fmt.Println(style.Faint(`"skl install <git-url>" to add a remote pack.`))
			return nil
		}

		if showSkills {
			return printSkills(skills, bundles, st)
		}
		return printBundles(bundles, st)
	},
}

func printBundles(bundles map[string][]string, st *state.State) error {
	loadedBundles := map[string]bool{}
	for _, b := range st.LoadedBundles() {
		loadedBundles[b] = true
	}

	names := make([]string, 0, len(bundles))
	for n := range bundles {
		names = append(names, n)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Println(style.Faint("No bundles defined. Create one with `skl bundle create <name> <skill...>`."))
		return nil
	}

	var rows [][]string
	for _, n := range names {
		mark := style.Faint("—")
		if loadedBundles[n] {
			mark = style.OK("loaded")
		}
		rows = append(rows, []string{n, fmt.Sprintf("%d", len(bundles[n])), mark})
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Headers("BUNDLE", "SKILLS", "STATUS").
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
}

func printSkills(skills []library.Skill, bundles map[string][]string, st *state.State) error {
	skillToBundles := map[string][]string{}
	for b, ids := range bundles {
		for _, id := range ids {
			skillToBundles[id] = append(skillToBundles[id], b)
		}
	}

	var rows [][]string
	for _, s := range skills {
		bs := skillToBundles[s.ID]
		sort.Strings(bs)
		bsStr := strings.Join(bs, ", ")
		if bsStr == "" {
			bsStr = style.Faint("(unbundled)")
		}
		mark := style.Faint("—")
		if _, ok := st.Loaded[s.ID]; ok {
			mark = style.OK("loaded")
		}
		src := style.Faint("local")
		if s.External {
			src = style.Faint("ext: " + s.Repo)
		}
		rows = append(rows, []string{s.ID, bsStr, mark, src})
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Headers("SKILL", "BUNDLES", "STATUS", "SOURCE").
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
}
